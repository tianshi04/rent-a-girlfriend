use serde::{Deserialize, Serialize};

#[derive(Debug, Deserialize)]
pub struct SendMessageRequest {
    pub text: String,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ChatMessageResponse {
    pub message_id: String,
    pub room_id: String,
    pub sender_id: String,
    pub content: String,
    pub created_at: String,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ReviewResponse {
    pub review_id: String,
    pub booking_id: String,
    pub client_id: String,
    pub companion_id: String,
    pub rating: i32,
    pub comment: String,
    pub created_at: String,
    pub updated_at: String,
}

#[derive(Debug, Deserialize)]
pub struct PaginationQuery {
    pub limit: Option<i64>,
    pub offset: Option<i64>,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct SubmitReviewRequest {
    pub booking_id: String,
    pub client_id: String,
    pub companion_id: String,
    pub rating: i32,
    pub comment: String,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ReviewCommandResponse {
    pub review_id: String,
    pub status: String,
    pub message: String,
}

#[derive(Debug, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct ChatRoomResponse {
    pub room_id: String,
    pub booking_id: String,
    pub client_id: String,
    pub companion_id: String,
    pub status: String,
    pub lock_at: Option<String>,
    pub created_at: String,
    pub updated_at: String,
}
