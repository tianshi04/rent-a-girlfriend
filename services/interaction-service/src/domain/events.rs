use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct ChatRoomCreatedEvent {
    pub room_id: String,
    pub booking_id: String,
    pub client_id: String,
    pub companion_id: String,
    pub occurred_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct ChatRoomLockedEvent {
    pub room_id: String,
    pub booking_id: String,
    pub occurred_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct ChatMessageSentEvent {
    pub message_id: String,
    pub room_id: String,
    pub booking_id: String,
    pub sender_id: String,
    pub recipient_id: String,
    pub content: String,
    pub occurred_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct ReviewSubmittedEvent {
    pub review_id: String,
    pub booking_id: String,
    pub client_id: String,
    pub companion_id: String,
    pub rating: i32,
    pub comment: String,
    pub occurred_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct ReviewHiddenEvent {
    pub review_id: String,
    pub booking_id: String,
    pub occurred_at: DateTime<Utc>,
}
