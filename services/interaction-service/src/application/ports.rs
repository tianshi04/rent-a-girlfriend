use crate::domain::chat_room::{ChatMessage, ChatRoom};
use crate::domain::errors::DomainError;
use crate::domain::review::Review;
use async_trait::async_trait;

#[cfg_attr(test, mockall::automock)]
#[async_trait]
pub trait ChatRoomRepository: Send + Sync {
    async fn save(&self, chat_room: &ChatRoom) -> Result<(), DomainError>;
    async fn find_by_id(&self, room_id: &str) -> Result<Option<ChatRoom>, DomainError>;
    async fn find_by_booking_id(&self, booking_id: &str) -> Result<Option<ChatRoom>, DomainError>;
    async fn save_message(&self, message: &ChatMessage) -> Result<(), DomainError>;
    async fn get_messages(
        &self,
        room_id: &str,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<ChatMessage>, DomainError>;
    async fn lock_expired_rooms(
        &self,
        now: chrono::DateTime<chrono::Utc>,
        limit: i64,
    ) -> Result<Vec<String>, DomainError>;
    async fn find_rooms_by_user_id(&self, user_id: &str) -> Result<Vec<ChatRoom>, DomainError>;
    async fn report_creation_failure(&self, booking_id: &str) -> Result<(), DomainError>;
}

#[cfg_attr(test, mockall::automock)]
#[async_trait]
pub trait ReviewRepository: Send + Sync {
    async fn save(&self, review: &Review) -> Result<(), DomainError>;
    async fn find_by_id(&self, review_id: &str) -> Result<Option<Review>, DomainError>;
    async fn find_by_booking_id(&self, booking_id: &str) -> Result<Option<Review>, DomainError>;
    async fn find_by_companion_id(&self, companion_id: &str) -> Result<Vec<Review>, DomainError>;
}

#[async_trait]
pub trait EventPublisher: Send + Sync {
    async fn publish_outbox(
        &self,
        event_id: &str,
        event_type: &str,
        payload: serde_json::Value,
    ) -> Result<(), DomainError>;
}

#[cfg_attr(test, mockall::automock)]
#[async_trait]
pub trait ProcessedEventRepository: Send + Sync {
    /// Returns true if the event has already been processed, otherwise records it and returns false.
    async fn check_and_record(&self, event_id: &str, event_type: &str)
    -> Result<bool, DomainError>;
}
