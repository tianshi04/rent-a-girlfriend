use async_trait::async_trait;
use crate::domain::errors::DomainError;
use crate::domain::chat_room::{ChatRoom, ChatMessage};
use crate::domain::review::Review;

#[async_trait]
pub trait ChatRoomRepository: Send + Sync {
    async fn save(&self, chat_room: &ChatRoom) -> Result<(), DomainError>;
    async fn find_by_id(&self, room_id: &str) -> Result<Option<ChatRoom>, DomainError>;
    async fn find_by_booking_id(&self, booking_id: &str) -> Result<Option<ChatRoom>, DomainError>;
    async fn save_message(&self, message: &ChatMessage) -> Result<(), DomainError>;
    async fn get_messages(&self, room_id: &str, limit: i64, offset: i64) -> Result<Vec<ChatMessage>, DomainError>;
}

#[async_trait]
pub trait ReviewRepository: Send + Sync {
    async fn save(&self, review: &Review) -> Result<(), DomainError>;
    async fn find_by_id(&self, review_id: &str) -> Result<Option<Review>, DomainError>;
    async fn find_by_booking_id(&self, booking_id: &str) -> Result<Option<Review>, DomainError>;
    async fn find_by_companion_id(&self, companion_id: &str) -> Result<Vec<Review>, DomainError>;
}

#[async_trait]
pub trait EventPublisher: Send + Sync {
    async fn publish_outbox(&self, event_id: &str, event_type: &str, payload: serde_json::Value) -> Result<(), DomainError>;
}
