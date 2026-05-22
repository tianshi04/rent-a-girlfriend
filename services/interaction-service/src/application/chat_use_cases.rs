use std::sync::Arc;
use crate::domain::errors::DomainError;
use crate::domain::chat_room::{ChatRoom, ChatMessage};
use crate::domain::value_objects::ChatContent;
use crate::application::ports::ChatRoomRepository;

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
