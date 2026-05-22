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
        if let Some(_) = self.repo.find_by_booking_id(&booking_id).await? {
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
