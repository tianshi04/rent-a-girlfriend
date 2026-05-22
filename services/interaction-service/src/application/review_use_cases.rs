use std::sync::Arc;
use crate::domain::errors::DomainError;
use crate::domain::review::Review;
use crate::domain::value_objects::Rating;
use crate::application::ports::ReviewRepository;

pub struct ReviewUseCases {
    repo: Arc<dyn ReviewRepository>,
}

impl ReviewUseCases {
    pub fn new(repo: Arc<dyn ReviewRepository>) -> Self {
        Self { repo }
    }

    pub async fn submit_review(
        &self,
        booking_id: String,
        client_id: String,
        companion_id: String,
        rating_value: i32,
        comment: String,
    ) -> Result<Review, DomainError> {
        // [INV-I04] Each booking_id can only have exactly 1 review
        if self.repo.find_by_booking_id(&booking_id).await?.is_some() {
            return Err(DomainError::ReviewAlreadyExists(booking_id));
        }

        // [INV-I05] Rating validation is performed by the Value Object constructor
        let rating = Rating::new(rating_value)?;
        let review = Review::create(booking_id, client_id, companion_id, rating, comment);

        self.repo.save(&review).await?;
        Ok(review)
    }

    pub async fn hide_review(&self, booking_id: &str, _reason: String) -> Result<(), DomainError> {
        let mut review = self
            .repo
            .find_by_booking_id(booking_id)
            .await?
            .ok_or_else(|| DomainError::ReviewNotFound(format!("Booking ID: {}", booking_id)))?;

        review.hide()?;
        self.repo.save(&review).await?;
        Ok(())
    }

    pub async fn get_companion_reviews(&self, companion_id: &str) -> Result<Vec<Review>, DomainError> {
        let reviews = self.repo.find_by_companion_id(companion_id).await?;
        Ok(reviews)
    }

    pub async fn get_booking_review(&self, booking_id: &str) -> Result<Option<Review>, DomainError> {
        let review = self.repo.find_by_booking_id(booking_id).await?;
        Ok(review)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::application::ports::MockReviewRepository;
    use crate::domain::review::Review;
    use crate::domain::value_objects::Rating;

    #[tokio::test]
    async fn test_submit_review_success() {
        let mut mock_repo = MockReviewRepository::new();

        // Mock find_by_booking_id to return None (no existing review)
        mock_repo.expect_find_by_booking_id()
            .with(mockall::predicate::eq("booking-123"))
            .times(1)
            .returning(|_| Ok(None));

        // Mock save to return Ok(())
        mock_repo.expect_save()
            .times(1)
            .returning(|_| Ok(()));

        let use_cases = ReviewUseCases::new(Arc::new(mock_repo));
        let res = use_cases.submit_review(
            "booking-123".to_string(),
            "client-789".to_string(),
            "companion-456".to_string(),
            5,
            "Outstanding service!".to_string(),
        ).await;

        assert!(res.is_ok());
        let review = res.unwrap();
        assert_eq!(review.booking_id, "booking-123");
        assert_eq!(review.client_id, "client-789");
        assert_eq!(review.companion_id, "companion-456");
        assert_eq!(review.rating.value(), 5);
        assert_eq!(review.comment, "Outstanding service!");
    }

    #[tokio::test]
    async fn test_submit_review_already_exists() {
        let mut mock_repo = MockReviewRepository::new();
        let existing_review = Review::create(
            "booking-123".to_string(),
            "client-789".to_string(),
            "companion-456".to_string(),
            Rating::new(4).unwrap(),
            "Good".to_string(),
        );

        // Mock find_by_booking_id to return the existing review
        mock_repo.expect_find_by_booking_id()
            .with(mockall::predicate::eq("booking-123"))
            .times(1)
            .returning(move |_| Ok(Some(existing_review.clone())));

        // save should NOT be called because it is blocked by [INV-I04]
        mock_repo.expect_save()
            .times(0);

        let use_cases = ReviewUseCases::new(Arc::new(mock_repo));
        let res = use_cases.submit_review(
            "booking-123".to_string(),
            "client-789".to_string(),
            "companion-456".to_string(),
            5,
            "Outstanding service!".to_string(),
        ).await;

        assert!(res.is_err());
        match res.unwrap_err() {
            DomainError::ReviewAlreadyExists(id) => assert_eq!(id, "booking-123"),
            _ => panic!("Expected ReviewAlreadyExists error"),
        }
    }

    #[tokio::test]
    async fn test_submit_review_invalid_rating() {
        let mut mock_repo = MockReviewRepository::new();

        // Mock find_by_booking_id to return None
        mock_repo.expect_find_by_booking_id()
            .with(mockall::predicate::eq("booking-123"))
            .times(1)
            .returning(|_| Ok(None));

        // save should NOT be called because rating validation fails
        mock_repo.expect_save()
            .times(0);

        let use_cases = ReviewUseCases::new(Arc::new(mock_repo));
        let res = use_cases.submit_review(
            "booking-123".to_string(),
            "client-789".to_string(),
            "companion-456".to_string(),
            6, // Invalid rating (max is 5)
            "Too good!".to_string(),
        ).await;

        assert!(res.is_err());
        match res.unwrap_err() {
            DomainError::InvalidRating(val) => assert_eq!(val, 6),
            _ => panic!("Expected InvalidRating error"),
        }
    }
}

