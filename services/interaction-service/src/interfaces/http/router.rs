use axum::{
    Json, Router,
    extract::{Path, Query, State},
    http::{HeaderMap, StatusCode},
    response::{IntoResponse, Response},
    routing::{get, post},
};
use serde_json::json;
use std::sync::Arc;

use super::dto::{
    ChatMessageResponse, PaginationQuery, ReviewCommandResponse, ReviewResponse,
    SendMessageRequest, SubmitReviewRequest,
};
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
        .route("/api/v1/interaction/reviews", post(submit_review_handler))
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

/// POST /api/v1/interaction/reviews
async fn submit_review_handler(
    State(state): State<AppState>,
    headers: HeaderMap,
    Json(payload): Json<SubmitReviewRequest>,
) -> Result<Response, Response> {
    let user_id = get_user_id(&headers)?;

    // Validate that the client ID matches the authenticated user-id header
    if payload.client_id != user_id {
        return Err((
            StatusCode::FORBIDDEN,
            Json(json!({
                "code": 7, // PermissionDenied
                "message": "You can only submit reviews for yourself.",
                "details": []
            })),
        )
            .into_response());
    }

    let review = state
        .review_cases
        .submit_review(
            payload.booking_id,
            payload.client_id,
            payload.companion_id,
            payload.rating,
            payload.comment,
        )
        .await
        .map_err(|e| ApiError(e).into_response())?;

    let resp = ReviewCommandResponse {
        review_id: review.review_id,
        status: "SUBMITTED".to_string(),
        message: "Review submitted successfully.".to_string(),
    };

    Ok((StatusCode::CREATED, Json(resp)).into_response())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::application::ports::{
        MockChatRoomRepository, MockProcessedEventRepository, MockReviewRepository,
    };
    use crate::domain::review::Review;
    use crate::domain::value_objects::Rating;
    use axum::body::Body;
    use axum::http::{Request, StatusCode};
    use std::sync::Arc;
    use tower::ServiceExt;

    fn setup_app(mock_review_repo: MockReviewRepository) -> Router {
        let mock_chat_repo = MockChatRoomRepository::new();
        let mock_event_repo = MockProcessedEventRepository::new();

        let chat_cases = Arc::new(ChatUseCases::new(
            Arc::new(mock_chat_repo),
            Arc::new(mock_event_repo),
        ));
        let review_cases = Arc::new(ReviewUseCases::new(Arc::new(mock_review_repo)));

        let state = AppState {
            chat_cases,
            review_cases,
        };
        create_router(state)
    }

    #[tokio::test]
    async fn test_submit_review_success() {
        let mut mock_repo = MockReviewRepository::new();

        mock_repo
            .expect_find_by_booking_id()
            .with(mockall::predicate::eq("booking-123"))
            .times(1)
            .returning(|_| Ok(None));

        mock_repo.expect_save().times(1).returning(|_| Ok(()));

        let app = setup_app(mock_repo);

        let request = Request::builder()
            .method("POST")
            .uri("/api/v1/interaction/reviews")
            .header("user-id", "client-123")
            .header("content-type", "application/json")
            .body(Body::from(
                r#"{
                "bookingId": "booking-123",
                "clientId": "client-123",
                "companionId": "companion-456",
                "rating": 5,
                "comment": "Outstanding!"
            }"#,
            ))
            .unwrap();

        let response: axum::response::Response = app.oneshot(request).await.unwrap();
        assert_eq!(response.status(), StatusCode::CREATED);

        let body_bytes = axum::body::to_bytes(response.into_body(), 1024)
            .await
            .unwrap();
        let body: serde_json::Value = serde_json::from_slice(&body_bytes).unwrap();

        assert_eq!(body["status"], "SUBMITTED");
        assert_eq!(body["message"], "Review submitted successfully.");
        assert!(!body["reviewId"].as_str().unwrap().is_empty());
    }

    #[tokio::test]
    async fn test_submit_review_unauthorized() {
        let mock_repo = MockReviewRepository::new();
        let app = setup_app(mock_repo);

        let request = Request::builder()
            .method("POST")
            .uri("/api/v1/interaction/reviews")
            .header("user-id", "different-client")
            .header("content-type", "application/json")
            .body(Body::from(
                r#"{
                "bookingId": "booking-123",
                "clientId": "client-123",
                "companionId": "companion-456",
                "rating": 5,
                "comment": "Outstanding!"
            }"#,
            ))
            .unwrap();

        let response: axum::response::Response = app.oneshot(request).await.unwrap();
        assert_eq!(response.status(), StatusCode::FORBIDDEN);

        let body_bytes = axum::body::to_bytes(response.into_body(), 1024)
            .await
            .unwrap();
        let body: serde_json::Value = serde_json::from_slice(&body_bytes).unwrap();
        assert_eq!(body["code"], 7); // PermissionDenied
    }

    #[tokio::test]
    async fn test_submit_review_missing_auth() {
        let mock_repo = MockReviewRepository::new();
        let app = setup_app(mock_repo);

        let request = Request::builder()
            .method("POST")
            .uri("/api/v1/interaction/reviews")
            .header("content-type", "application/json")
            .body(Body::from(
                r#"{
                "bookingId": "booking-123",
                "clientId": "client-123",
                "companionId": "companion-456",
                "rating": 5,
                "comment": "Outstanding!"
            }"#,
            ))
            .unwrap();

        let response: axum::response::Response = app.oneshot(request).await.unwrap();
        assert_eq!(response.status(), StatusCode::UNAUTHORIZED);
    }

    #[tokio::test]
    async fn test_submit_review_invalid_rating() {
        let mut mock_repo = MockReviewRepository::new();
        mock_repo
            .expect_find_by_booking_id()
            .with(mockall::predicate::eq("booking-123"))
            .times(1)
            .returning(|_| Ok(None));

        let app = setup_app(mock_repo);

        let request = Request::builder()
            .method("POST")
            .uri("/api/v1/interaction/reviews")
            .header("user-id", "client-123")
            .header("content-type", "application/json")
            .body(Body::from(
                r#"{
                "bookingId": "booking-123",
                "clientId": "client-123",
                "companionId": "companion-456",
                "rating": 6,
                "comment": "Too high!"
            }"#,
            ))
            .unwrap();

        let response: axum::response::Response = app.oneshot(request).await.unwrap();
        assert_eq!(response.status(), StatusCode::BAD_REQUEST);

        let body_bytes = axum::body::to_bytes(response.into_body(), 1024)
            .await
            .unwrap();
        let body: serde_json::Value = serde_json::from_slice(&body_bytes).unwrap();
        assert_eq!(body["code"], 3); // InvalidArgument
    }

    #[tokio::test]
    async fn test_submit_review_already_exists() {
        let mut mock_repo = MockReviewRepository::new();

        let existing_review = Review::create(
            "booking-123".to_string(),
            "client-123".to_string(),
            "companion-456".to_string(),
            Rating::new(4).unwrap(),
            "Existing".to_string(),
        );

        mock_repo
            .expect_find_by_booking_id()
            .with(mockall::predicate::eq("booking-123"))
            .times(1)
            .returning(move |_| Ok(Some(existing_review.clone())));

        let app = setup_app(mock_repo);

        let request = Request::builder()
            .method("POST")
            .uri("/api/v1/interaction/reviews")
            .header("user-id", "client-123")
            .header("content-type", "application/json")
            .body(Body::from(
                r#"{
                "bookingId": "booking-123",
                "clientId": "client-123",
                "companionId": "companion-456",
                "rating": 5,
                "comment": "Outstanding!"
            }"#,
            ))
            .unwrap();

        let response: axum::response::Response = app.oneshot(request).await.unwrap();
        assert_eq!(response.status(), StatusCode::CONFLICT);

        let body_bytes = axum::body::to_bytes(response.into_body(), 1024)
            .await
            .unwrap();
        let body: serde_json::Value = serde_json::from_slice(&body_bytes).unwrap();
        assert_eq!(body["code"], 6); // AlreadyExists
    }
}
