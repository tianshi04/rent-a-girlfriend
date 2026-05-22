use crate::application::ports::ChatRoomRepository;
use crate::domain::chat_room::{ChatMessage, ChatRoom};
use crate::domain::errors::DomainError;
use crate::domain::value_objects::ChatContent;
use std::sync::Arc;

pub struct ChatUseCases {
    repo: Arc<dyn ChatRoomRepository>,
}

impl ChatUseCases {
    pub fn new(repo: Arc<dyn ChatRoomRepository>) -> Self {
        Self { repo }
    }

    pub async fn create_chat_room(
        &self,
        booking_id: String,
        client_id: String,
        companion_id: String,
    ) -> Result<ChatRoom, DomainError> {
        // If chat room already exists for this booking, return it (idempotency)
        if let Some(existing) = self.repo.find_by_booking_id(&booking_id).await? {
            return Ok(existing);
        }

        let chat_room = ChatRoom::create(booking_id, client_id, companion_id);
        self.repo.save(&chat_room).await?;
        Ok(chat_room)
    }

    pub async fn lock_chat_room(&self, booking_id: &str) -> Result<(), DomainError> {
        let mut chat_room = self
            .repo
            .find_by_booking_id(booking_id)
            .await?
            .ok_or_else(|| DomainError::ChatRoomNotFound(format!("Booking ID: {}", booking_id)))?;

        chat_room.lock()?;
        self.repo.save(&chat_room).await?;
        Ok(())
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
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::application::ports::MockChatRoomRepository;
    use crate::domain::chat_room::ChatRoom;

    #[tokio::test]
    async fn test_create_chat_room_success() {
        let mut mock_repo = MockChatRoomRepository::new();

        // Mock find_by_booking_id to return None (room doesn't exist yet)
        mock_repo
            .expect_find_by_booking_id()
            .with(mockall::predicate::eq("booking-123"))
            .times(1)
            .returning(|_| Ok(None));

        // Mock save to return Ok(())
        mock_repo.expect_save().times(1).returning(|_| Ok(()));

        let use_cases = ChatUseCases::new(Arc::new(mock_repo));
        let res = use_cases
            .create_chat_room(
                "booking-123".to_string(),
                "client-789".to_string(),
                "companion-456".to_string(),
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

        // Save should NOT be called since the room already exists
        mock_repo.expect_save().times(0);

        let use_cases = ChatUseCases::new(Arc::new(mock_repo));
        let res = use_cases
            .create_chat_room(
                "booking-123".to_string(),
                "client-789".to_string(),
                "companion-456".to_string(),
            )
            .await;

        assert!(res.is_ok());
        assert_eq!(res.unwrap().room_id, expected_room.room_id);
    }

    #[tokio::test]
    async fn test_send_message_success() {
        let mut mock_repo = MockChatRoomRepository::new();
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

        let use_cases = ChatUseCases::new(Arc::new(mock_repo));
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

        let use_cases = ChatUseCases::new(Arc::new(mock_repo));
        let res = use_cases
            .send_message("room-abc", "unauthorized-stranger", "Hello!".to_string())
            .await;

        assert!(res.is_err());
        match res.unwrap_err() {
            DomainError::UnauthorizedSender { .. } => {}
            _ => panic!("Expected UnauthorizedSender error"),
        }
    }
}
