use axum::{
    Json, Router,
    extract::{Path, Query, State},
    http::{HeaderMap, StatusCode},
    response::{IntoResponse, Response},
    routing::{get, post},
};
use serde_json::json;
use std::sync::Arc;

use super::dto::{ChatMessageResponse, PaginationQuery, ReviewResponse, SendMessageRequest};
use crate::application::chat_use_cases::ChatUseCases;
use crate::application::review_use_cases::ReviewUseCases;
use crate::domain::errors::DomainError;

#[derive(Clone)]
pub struct AppState {
    pub chat_cases: Arc<ChatUseCases>,
    pub review_cases: Arc<ReviewUseCases>,
}

pub struct ApiError(pub DomainError);

impl IntoResponse for ApiError {
    fn into_response(self) -> Response {
        let status = match &self.0 {
            DomainError::ChatRoomLocked(_) => StatusCode::FORBIDDEN,
            DomainError::UnauthorizedSender { .. } => StatusCode::FORBIDDEN,
            DomainError::InvalidRating(_) => StatusCode::BAD_REQUEST,
            DomainError::ReviewAlreadyExists(_) => StatusCode::CONFLICT,
            DomainError::ChatRoomAlreadyExists(_) => StatusCode::CONFLICT,
            DomainError::ChatRoomNotFound(_) => StatusCode::NOT_FOUND,
            DomainError::ReviewNotFound(_) => StatusCode::NOT_FOUND,
            DomainError::DatabaseError(_) => StatusCode::INTERNAL_SERVER_ERROR,
        };

        let code = match &self.0 {
            DomainError::ChatRoomLocked(_) => 7,         // PermissionDenied
            DomainError::UnauthorizedSender { .. } => 7, // PermissionDenied
            DomainError::InvalidRating(_) => 3,          // InvalidArgument
            DomainError::ReviewAlreadyExists(_) => 6,    // AlreadyExists
            DomainError::ChatRoomAlreadyExists(_) => 6,  // AlreadyExists
            DomainError::ChatRoomNotFound(_) => 5,       // NotFound
            DomainError::ReviewNotFound(_) => 5,         // NotFound
            DomainError::DatabaseError(_) => 13,         // Internal
        };

        let body = Json(json!({
            "code": code,
            "message": self.0.to_string(),
            "details": []
        }));

        (status, body).into_response()
    }
}

#[allow(clippy::result_large_err)]
fn get_user_id(headers: &HeaderMap) -> Result<String, Response> {
    if let Some(user_id_str) = headers.get("user-id").and_then(|val| val.to_str().ok()) {
        return Ok(user_id_str.to_string());
    }
    Err((
        StatusCode::UNAUTHORIZED,
        Json(json!({
            "code": 16, // Unauthenticated
            "message": "Missing or invalid 'user-id' header. Authentication is offloaded to Istio Waypoint Gateway.",
            "details": []
        })),
    ).into_response())
}

pub fn create_router(state: AppState) -> Router {
    Router::new()
        .route(
            "/api/v1/interaction/rooms/{room_id}/messages",
            post(send_message_handler).get(get_messages_handler),
        )
        .route(
            "/api/v1/interaction/reviews/companion/{companion_id}",
            get(get_companion_reviews_handler),
        )
        .with_state(state)
}

/// POST /api/v1/interaction/rooms/:room_id/messages
async fn send_message_handler(
    State(state): State<AppState>,
    Path(room_id): Path<String>,
    headers: HeaderMap,
    Json(payload): Json<SendMessageRequest>,
) -> Result<Response, Response> {
    let user_id = get_user_id(&headers)?;

    let msg = state
        .chat_cases
        .send_message(&room_id, &user_id, payload.text)
        .await
        .map_err(|e| ApiError(e).into_response())?;

    let resp = ChatMessageResponse {
        message_id: msg.message_id,
        room_id: msg.room_id,
        sender_id: msg.sender_id,
        content: msg.content,
        created_at: msg.created_at.to_rfc3339(),
    };

    Ok((StatusCode::CREATED, Json(resp)).into_response())
}

/// GET /api/v1/interaction/rooms/:room_id/messages
async fn get_messages_handler(
    State(state): State<AppState>,
    Path(room_id): Path<String>,
    headers: HeaderMap,
    Query(query): Query<PaginationQuery>,
) -> Result<Response, Response> {
    let user_id = get_user_id(&headers)?;

    let limit = query.limit.unwrap_or(50);
    let offset = query.offset.unwrap_or(0);

    let messages = state
        .chat_cases
        .get_messages(&room_id, &user_id, limit, offset)
        .await
        .map_err(|e| ApiError(e).into_response())?;

    let resp: Vec<ChatMessageResponse> = messages
        .into_iter()
        .map(|msg| ChatMessageResponse {
            message_id: msg.message_id,
            room_id: msg.room_id,
            sender_id: msg.sender_id,
            content: msg.content,
            created_at: msg.created_at.to_rfc3339(),
        })
        .collect();

    Ok(Json(resp).into_response())
}

/// GET /api/v1/interaction/reviews/companion/:companion_id
async fn get_companion_reviews_handler(
    State(state): State<AppState>,
    Path(companion_id): Path<String>,
) -> Result<Response, Response> {
    let reviews = state
        .review_cases
        .get_companion_reviews(&companion_id)
        .await
        .map_err(|e| ApiError(e).into_response())?;

    let resp: Vec<ReviewResponse> = reviews
        .into_iter()
        .map(|rev| ReviewResponse {
            review_id: rev.review_id,
            booking_id: rev.booking_id,
            client_id: rev.client_id,
            companion_id: rev.companion_id,
            rating: rev.rating.value(),
            comment: rev.comment,
            created_at: rev.created_at.to_rfc3339(),
            updated_at: rev.updated_at.to_rfc3339(),
        })
        .collect();

    Ok(Json(resp).into_response())
}
