use chrono::{DateTime, Utc};
use uuid::Uuid;
use crate::domain::errors::DomainError;
use crate::domain::value_objects::Rating;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Review {
    pub review_id: String,
    pub booking_id: String,
    pub client_id: String,
    pub companion_id: String,
    pub rating: Rating,
    pub comment: String,
    pub is_visible: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

impl Review {
    #[allow(clippy::too_many_arguments)]
    pub fn new(
        review_id: String,
        booking_id: String,
        client_id: String,
        companion_id: String,
        rating: Rating,
        comment: String,
        is_visible: bool,
        created_at: DateTime<Utc>,
        updated_at: DateTime<Utc>,
    ) -> Self {
        Self {
            review_id,
            booking_id,
            client_id,
            companion_id,
            rating,
            comment,
            is_visible,
            created_at,
            updated_at,
        }
    }

    pub fn create(
        booking_id: String,
        client_id: String,
        companion_id: String,
        rating: Rating,
        comment: String,
    ) -> Self {
        let now = Utc::now();
        Self {
            review_id: Uuid::new_v4().to_string(),
            booking_id,
            client_id,
            companion_id,
            rating,
            comment,
            is_visible: true,
            created_at: now,
            updated_at: now,
        }
    }

    pub fn hide(&mut self) -> Result<(), DomainError> {
        self.is_visible = false;
        self.updated_at = Utc::now();
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_rating_creation_valid() {
        let rating = Rating::new(1);
        assert!(rating.is_ok());
        assert_eq!(rating.unwrap().value(), 1);

        let rating_five = Rating::new(5);
        assert!(rating_five.is_ok());
        assert_eq!(rating_five.unwrap().value(), 5);
    }

    #[test]
    fn test_rating_creation_invalid() {
        let rating_zero = Rating::new(0);
        assert!(rating_zero.is_err());
        match rating_zero.unwrap_err() {
            DomainError::InvalidRating(val) => assert_eq!(val, 0),
            _ => panic!("Expected InvalidRating error"),
        }

        let rating_six = Rating::new(6);
        assert!(rating_six.is_err());
        match rating_six.unwrap_err() {
            DomainError::InvalidRating(val) => assert_eq!(val, 6),
            _ => panic!("Expected InvalidRating error"),
        }
    }

    #[test]
    fn test_review_creation() {
        let rating = Rating::new(4).unwrap();
        let review = Review::create(
            "booking-123".to_string(),
            "client-789".to_string(),
            "companion-456".to_string(),
            rating,
            "Very nice companion!".to_string(),
        );

        assert_eq!(review.booking_id, "booking-123");
        assert_eq!(review.client_id, "client-789");
        assert_eq!(review.companion_id, "companion-456");
        assert_eq!(review.rating.value(), 4);
        assert_eq!(review.comment, "Very nice companion!");
        assert!(review.is_visible); // Reviews must be immediately publicly visible
    }

    #[test]
    fn test_review_hide() {
        let rating = Rating::new(5).unwrap();
        let mut review = Review::create(
            "booking-123".to_string(),
            "client-789".to_string(),
            "companion-456".to_string(),
            rating,
            "Amazing!".to_string(),
        );

        assert!(review.is_visible);
        assert!(review.hide().is_ok());
        assert!(!review.is_visible);
    }
}

