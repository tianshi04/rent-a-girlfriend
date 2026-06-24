use crate::application::ports::{ChatRoomRepository, ProcessedEventRepository};
use crate::domain::chat_room::{ChatMessage, ChatRoom};
use crate::domain::errors::DomainError;
use crate::domain::value_objects::ChatContent;
use chrono::{DateTime, Utc};
use std::sync::Arc;

pub struct ChatUseCases {
    repo: Arc<dyn ChatRoomRepository>,
    processed_event_repo: Arc<dyn ProcessedEventRepository>,
}

impl ChatUseCases {
    pub fn new(
        repo: Arc<dyn ChatRoomRepository>,
        processed_event_repo: Arc<dyn ProcessedEventRepository>,
    ) -> Self {
        Self {
            repo,
            processed_event_repo,
        }
    }

    pub async fn create_chat_room(
        &self,
        booking_id: String,
        client_id: String,
        companion_id: String,
        event_id: Option<String>,
    ) -> Result<ChatRoom, DomainError> {
        if let Some(ref ev_id) = event_id {
            let already_processed = self
                .processed_event_repo
                .check_and_record(ev_id, "interaction.create-chat-room.v1")
                .await?;
            if already_processed {
                if let Some(existing) = self.repo.find_by_booking_id(&booking_id).await? {
                    return Ok(existing);
                }
                return Err(DomainError::ChatRoomNotFound(format!(
                    "Booking ID: {}",
                    booking_id
                )));
            }
        }

        // If chat room already exists for this booking, return it (idempotency self-healing)
        if let Some(existing) = self.repo.find_by_booking_id(&booking_id).await? {
            // Save the existing room again to write a new ChatRoomCreated event into the outbox
            // (with a new event UUID). This allows the outbox worker to republish the success event
            // so the SAGA coordinator can receive it on retries and doesn't get stuck in WAITING_FOR_CHAT.
            self.repo.save(&existing).await?;
            return Ok(existing);
        }

        let chat_room = ChatRoom::create(booking_id, client_id, companion_id);
        self.repo.save(&chat_room).await?;
        Ok(chat_room)
    }

    pub async fn lock_chat_room(
        &self,
        booking_id: &str,
        event_id: Option<String>,
    ) -> Result<(), DomainError> {
        if let Some(ref ev_id) = event_id {
            let already_processed = self
                .processed_event_repo
                .check_and_record(ev_id, "booking.booking-cancelled.v1")
                .await?;
            if already_processed {
                return Ok(());
            }
        }

        let mut chat_room = self
            .repo
            .find_by_booking_id(booking_id)
            .await?
            .ok_or_else(|| DomainError::ChatRoomNotFound(format!("Booking ID: {}", booking_id)))?;

        chat_room.lock()?;
        self.repo.save(&chat_room).await?;
        Ok(())
    }

    pub async fn schedule_chat_room_lock(
        &self,
        booking_id: &str,
        lock_at: DateTime<Utc>,
        event_id: Option<String>,
    ) -> Result<(), DomainError> {
        if let Some(ref ev_id) = event_id {
            let already_processed = self
                .processed_event_repo
                .check_and_record(ev_id, "booking.booking-completed.v1")
                .await?;
            if already_processed {
                return Ok(());
            }
        }

        let mut chat_room = self
            .repo
            .find_by_booking_id(booking_id)
            .await?
            .ok_or_else(|| DomainError::ChatRoomNotFound(format!("Booking ID: {}", booking_id)))?;

        if chat_room.status == crate::domain::chat_room::ChatRoomStatus::Active {
            chat_room.lock_at = Some(lock_at);
            self.repo.save(&chat_room).await?;
        }
        Ok(())
    }

    pub async fn lock_expired_rooms(&self, limit: i64) -> Result<Vec<String>, DomainError> {
        self.repo.lock_expired_rooms(Utc::now(), limit).await
    }

    pub async fn send_message(
        &self,
        room_id: &str,
        sender_id: &str,
        text: String,
    ) -> Result<ChatMessage, DomainError> {
        let chat_room = self
            .repo
            .find_by_id(room_id)
            .await?
            .ok_or_else(|| DomainError::ChatRoomNotFound(room_id.to_string()))?;

        let chat_content = ChatContent::new(text);
        let message = chat_room.send_message(sender_id, chat_content)?;

        self.repo.save_message(&message).await?;
        Ok(message)
    }

    pub async fn get_messages(
        &self,
        room_id: &str,
        requester_id: &str,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<ChatMessage>, DomainError> {
        let chat_room = self
            .repo
            .find_by_id(room_id)
            .await?
            .ok_or_else(|| DomainError::ChatRoomNotFound(room_id.to_string()))?;

        // [INV-I01] Only participant IDs can view/read messages in the room
        chat_room.validate_sender(requester_id)?;

        let messages = self.repo.get_messages(room_id, limit, offset).await?;
        Ok(messages)
    }

    pub async fn get_user_rooms(&self, user_id: &str) -> Result<Vec<ChatRoom>, DomainError> {
        self.repo.find_rooms_by_user_id(user_id).await
    }

    pub async fn report_creation_failure(&self, booking_id: &str) -> Result<(), DomainError> {
        self.repo.report_creation_failure(booking_id).await
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::application::ports::{MockChatRoomRepository, MockProcessedEventRepository};
    use crate::domain::chat_room::ChatRoom;

    #[tokio::test]
    async fn test_create_chat_room_success() {
        let mut mock_repo = MockChatRoomRepository::new();
        let mock_processed_repo = MockProcessedEventRepository::new();

        // Mock find_by_booking_id to return None (room doesn't exist yet)
        mock_repo
            .expect_find_by_booking_id()
            .with(mockall::predicate::eq("booking-123"))
            .times(1)
            .returning(|_| Ok(None));

        // Mock save to return Ok(())
        mock_repo.expect_save().times(1).returning(|_| Ok(()));

        let use_cases = ChatUseCases::new(Arc::new(mock_repo), Arc::new(mock_processed_repo));
        let res = use_cases
            .create_chat_room(
                "booking-123".to_string(),
                "client-789".to_string(),
                "companion-456".to_string(),
                None,
            )
            .await;

        assert!(res.is_ok());
        let room = res.unwrap();
        assert_eq!(room.booking_id, "booking-123");
        assert_eq!(room.client_id, "client-789");
        assert_eq!(room.companion_id, "companion-456");
    }

    #[tokio::test]
    async fn test_create_chat_room_already_exists() {
        let mut mock_repo = MockChatRoomRepository::new();
        let mock_processed_repo = MockProcessedEventRepository::new();
        let existing_room = ChatRoom::create(
            "booking-123".to_string(),
            "client-789".to_string(),
            "companion-456".to_string(),
        );
        let expected_room = existing_room.clone();

        // Mock find_by_booking_id to return the existing room (idempotency case)
        mock_repo
            .expect_find_by_booking_id()
            .with(mockall::predicate::eq("booking-123"))
            .times(1)
            .returning(move |_| Ok(Some(existing_room.clone())));

        // Save SHOULD be called 1 time to trigger outbox event (idempotency self-healing)
        mock_repo.expect_save().times(1).returning(|_| Ok(()));

        let use_cases = ChatUseCases::new(Arc::new(mock_repo), Arc::new(mock_processed_repo));
        let res = use_cases
            .create_chat_room(
                "booking-123".to_string(),
                "client-789".to_string(),
                "companion-456".to_string(),
                None,
            )
            .await;

        assert!(res.is_ok());
        assert_eq!(res.unwrap().room_id, expected_room.room_id);
    }

    #[tokio::test]
    async fn test_create_chat_room_duplicate_event_id() {
        let mut mock_repo = MockChatRoomRepository::new();
        let mut mock_processed_repo = MockProcessedEventRepository::new();
        let existing_room = ChatRoom::create(
            "booking-123".to_string(),
            "client-789".to_string(),
            "companion-456".to_string(),
        );
        let expected_room = existing_room.clone();

        // Mock processed repo to return true (duplicate event)
        mock_processed_repo
            .expect_check_and_record()
            .with(
                mockall::predicate::eq("event-xyz"),
                mockall::predicate::eq("interaction.create-chat-room.v1"),
            )
            .times(1)
            .returning(|_, _| Ok(true));

        // Mock find_by_booking_id to return the existing room
        mock_repo
            .expect_find_by_booking_id()
            .with(mockall::predicate::eq("booking-123"))
            .times(1)
            .returning(move |_| Ok(Some(existing_room.clone())));

        // Save should NOT be called because event was already processed
        mock_repo.expect_save().times(0);

        let use_cases = ChatUseCases::new(Arc::new(mock_repo), Arc::new(mock_processed_repo));
        let res = use_cases
            .create_chat_room(
                "booking-123".to_string(),
                "client-789".to_string(),
                "companion-456".to_string(),
                Some("event-xyz".to_string()),
            )
            .await;

        assert!(res.is_ok());
        assert_eq!(res.unwrap().room_id, expected_room.room_id);
    }

    #[tokio::test]
    async fn test_send_message_success() {
        let mut mock_repo = MockChatRoomRepository::new();
        let mock_processed_repo = MockProcessedEventRepository::new();
        let room = ChatRoom::create(
            "booking-123".to_string(),
            "client-789".to_string(),
            "companion-456".to_string(),
        );

        // Mock find_by_id to return the room
        mock_repo
            .expect_find_by_id()
            .with(mockall::predicate::eq("room-abc"))
            .times(1)
            .returning(move |_| Ok(Some(room.clone())));

        // Mock save_message to return Ok(())
        mock_repo
            .expect_save_message()
            .times(1)
            .returning(|_| Ok(()));

        let use_cases = ChatUseCases::new(Arc::new(mock_repo), Arc::new(mock_processed_repo));
        let res = use_cases
            .send_message("room-abc", "client-789", "Hello there!".to_string())
            .await;

        assert!(res.is_ok());
        let msg = res.unwrap();
        assert_eq!(msg.sender_id, "client-789");
        assert_eq!(msg.content, "Hello there!");
    }

    #[tokio::test]
    async fn test_send_message_unauthorized() {
        let mut mock_repo = MockChatRoomRepository::new();
        let mock_processed_repo = MockProcessedEventRepository::new();
        let room = ChatRoom::create(
            "booking-123".to_string(),
            "client-789".to_string(),
            "companion-456".to_string(),
        );

        // Mock find_by_id to return the room
        mock_repo
            .expect_find_by_id()
            .with(mockall::predicate::eq("room-abc"))
            .times(1)
            .returning(move |_| Ok(Some(room.clone())));

        // save_message should NOT be called because sender validation fails
        mock_repo.expect_save_message().times(0);

        let use_cases = ChatUseCases::new(Arc::new(mock_repo), Arc::new(mock_processed_repo));
        let res = use_cases
            .send_message("room-abc", "unauthorized-stranger", "Hello!".to_string())
            .await;

        assert!(res.is_err());
        match res.unwrap_err() {
            DomainError::UnauthorizedSender { .. } => {}
            _ => panic!("Expected UnauthorizedSender error"),
        }
    }

    #[tokio::test]
    async fn test_schedule_chat_room_lock_success() {
        let mut mock_repo = MockChatRoomRepository::new();
        let mut mock_processed_repo = MockProcessedEventRepository::new();
        let room = ChatRoom::create(
            "booking-123".to_string(),
            "client-789".to_string(),
            "companion-456".to_string(),
        );

        mock_processed_repo
            .expect_check_and_record()
            .with(
                mockall::predicate::eq("event-xyz"),
                mockall::predicate::eq("booking.booking-completed.v1"),
            )
            .times(1)
            .returning(|_, _| Ok(false));

        mock_repo
            .expect_find_by_booking_id()
            .with(mockall::predicate::eq("booking-123"))
            .times(1)
            .returning(move |_| Ok(Some(room.clone())));

        let target_lock_time = Utc::now() + chrono::Duration::hours(24);
        mock_repo
            .expect_save()
            .withf(move |r| r.lock_at == Some(target_lock_time))
            .times(1)
            .returning(|_| Ok(()));

        let use_cases = ChatUseCases::new(Arc::new(mock_repo), Arc::new(mock_processed_repo));
        let res = use_cases
            .schedule_chat_room_lock(
                "booking-123",
                target_lock_time,
                Some("event-xyz".to_string()),
            )
            .await;

        assert!(res.is_ok());
    }

    #[tokio::test]
    async fn test_report_creation_failure() {
        let mut mock_repo = MockChatRoomRepository::new();
        let mock_processed_repo = MockProcessedEventRepository::new();

        mock_repo
            .expect_report_creation_failure()
            .with(mockall::predicate::eq("booking-123"))
            .times(1)
            .returning(|_| Ok(()));

        let use_cases = ChatUseCases::new(Arc::new(mock_repo), Arc::new(mock_processed_repo));
        let res = use_cases.report_creation_failure("booking-123").await;
        assert!(res.is_ok());
    }

    #[tokio::test]
    async fn test_get_user_rooms_success() {
        let mut mock_repo = MockChatRoomRepository::new();
        let mock_processed_repo = MockProcessedEventRepository::new();
        let room = ChatRoom::create(
            "booking-123".to_string(),
            "client-789".to_string(),
            "companion-456".to_string(),
        );
        let rooms_list = vec![room];
        let expected_list = rooms_list.clone();

        mock_repo
            .expect_find_rooms_by_user_id()
            .with(mockall::predicate::eq("client-789"))
            .times(1)
            .returning(move |_| Ok(rooms_list.clone()));

        let use_cases = ChatUseCases::new(Arc::new(mock_repo), Arc::new(mock_processed_repo));
        let res = use_cases.get_user_rooms("client-789").await;
        assert!(res.is_ok());
        let returned_rooms = res.unwrap();
        assert_eq!(returned_rooms.len(), 1);
        assert_eq!(returned_rooms[0].room_id, expected_list[0].room_id);
    }
}
