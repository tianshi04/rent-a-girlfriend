use crate::application::chat_use_cases::ChatUseCases;
use crate::application::review_use_cases::ReviewUseCases;
use crate::domain::errors::DomainError;
use std::sync::Arc;
use tonic::{Request, Response, Status};

// Include the generated Tonic proto structures from Cargo's OUT_DIR
pub mod proto {
    tonic::include_proto!("interaction.v1");
}

use proto::interaction_service_server::InteractionService;
use proto::{
    ChatCommandResponse, CreateChatRoomRequest, HideReviewRequest, LockChatRoomRequest,
    ReviewCommandResponse, SubmitReviewRequest,
};

pub struct InteractionServicer {
    chat_cases: Arc<ChatUseCases>,
    review_cases: Arc<ReviewUseCases>,
}

impl InteractionServicer {
    pub fn new(chat_cases: Arc<ChatUseCases>, review_cases: Arc<ReviewUseCases>) -> Self {
        Self {
            chat_cases,
            review_cases,
        }
    }
}

/// Helper mapping domain exceptions into standard gRPC errors
fn map_domain_error(err: DomainError) -> Status {
    match err {
        DomainError::ChatRoomLocked(msg) => {
            Status::failed_precondition(format!("[INV-I02] Room locked: {}", msg))
        }
        DomainError::UnauthorizedSender { room_id, user_id } => Status::permission_denied(format!(
            "[INV-I01] User {} unauthorized in room {}",
            user_id, room_id
        )),
        DomainError::InvalidRating(r) => Status::invalid_argument(format!(
            "[INV-I05] Invalid rating value: {}. Must be 1-5.",
            r
        )),
        DomainError::ReviewAlreadyExists(booking_id) => Status::already_exists(format!(
            "[INV-I04] Review already exists for booking: {}",
            booking_id
        )),
        DomainError::ChatRoomNotFound(msg) => Status::not_found(msg),
        DomainError::ReviewNotFound(msg) => Status::not_found(msg),
        DomainError::ChatRoomAlreadyExists(msg) => Status::already_exists(msg),
    }
}

#[tonic::async_trait]
impl InteractionService for InteractionServicer {
    async fn create_chat_room(
        &self,
        request: Request<CreateChatRoomRequest>,
    ) -> Result<Response<ChatCommandResponse>, Status> {
        let req = request.into_inner();
        let chat_room = self
            .chat_cases
            .create_chat_room(req.booking_id, req.client_id, req.companion_id, None)
            .await
            .map_err(map_domain_error)?;

        Ok(Response::new(ChatCommandResponse {
            room_id: chat_room.room_id,
            status: chat_room.status.as_str().to_string(),
            message: "Chat room created successfully.".to_string(),
        }))
    }

    async fn lock_chat_room(
        &self,
        request: Request<LockChatRoomRequest>,
    ) -> Result<Response<ChatCommandResponse>, Status> {
        let req = request.into_inner();
        self.chat_cases
            .lock_chat_room(&req.booking_id, None)
            .await
            .map_err(map_domain_error)?;

        Ok(Response::new(ChatCommandResponse {
            room_id: "".to_string(),
            status: "LOCKED".to_string(),
            message: "Chat room locked successfully.".to_string(),
        }))
    }

    async fn submit_review(
        &self,
        request: Request<SubmitReviewRequest>,
    ) -> Result<Response<ReviewCommandResponse>, Status> {
        let req = request.into_inner();
        let review = self
            .review_cases
            .submit_review(
                req.booking_id,
                req.client_id,
                req.companion_id,
                req.rating,
                req.comment,
            )
            .await
            .map_err(map_domain_error)?;

        Ok(Response::new(ReviewCommandResponse {
            review_id: review.review_id,
            status: "SUBMITTED".to_string(),
            message: "Review submitted successfully.".to_string(),
        }))
    }

    async fn hide_review(
        &self,
        request: Request<HideReviewRequest>,
    ) -> Result<Response<ReviewCommandResponse>, Status> {
        let req = request.into_inner();
        self.review_cases
            .hide_review(&req.booking_id, req.reason)
            .await
            .map_err(map_domain_error)?;

        Ok(Response::new(ReviewCommandResponse {
            review_id: "".to_string(),
            status: "HIDDEN".to_string(),
            message: "Review hidden successfully.".to_string(),
        }))
    }
}
