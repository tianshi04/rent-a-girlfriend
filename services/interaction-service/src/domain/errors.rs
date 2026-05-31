use std::fmt;

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum DomainError {
    ChatRoomLocked(String),
    UnauthorizedSender { room_id: String, user_id: String },
    InvalidRating(i32),
    ReviewAlreadyExists(String),
    ChatRoomAlreadyExists(String),
    ChatRoomNotFound(String),
    ReviewNotFound(String),
}

impl fmt::Display for DomainError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            DomainError::ChatRoomLocked(booking_id) => {
                write!(
                    f,
                    "[INV-I02] Chat room for booking {} is LOCKED and cannot accept new messages.",
                    booking_id
                )
            }
            DomainError::UnauthorizedSender { room_id, user_id } => {
                write!(
                    f,
                    "[INV-I01] User {} is not authorized to send messages in chat room {}.",
                    user_id, room_id
                )
            }
            DomainError::InvalidRating(rating) => {
                write!(
                    f,
                    "[INV-I05] Rating must be between 1 and 5. Received: {}.",
                    rating
                )
            }
            DomainError::ReviewAlreadyExists(booking_id) => {
                write!(
                    f,
                    "[INV-I04] Review already exists for booking {}.",
                    booking_id
                )
            }
            DomainError::ChatRoomAlreadyExists(booking_id) => {
                write!(f, "Chat room already exists for booking {}.", booking_id)
            }
            DomainError::ChatRoomNotFound(room_id) => {
                write!(f, "Chat room not found for ID: {}.", room_id)
            }
            DomainError::ReviewNotFound(review_id) => {
                write!(f, "Review not found for ID: {}.", review_id)
            }
        }
    }
}

impl std::error::Error for DomainError {}
