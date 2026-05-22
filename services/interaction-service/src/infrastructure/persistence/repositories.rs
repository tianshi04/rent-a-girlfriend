use async_trait::async_trait;
use sqlx::{PgPool, Postgres, Transaction, Row};
use uuid::Uuid;
use chrono::Utc;
use crate::domain::errors::DomainError;
use crate::domain::chat_room::{ChatRoom, ChatMessage, ChatRoomStatus};
use crate::domain::review::Review;
use crate::domain::value_objects::Rating;
use crate::domain::events::{ChatRoomCreatedEvent, ChatRoomLockedEvent, ReviewSubmittedEvent, ReviewHiddenEvent};
use crate::application::ports::{ChatRoomRepository, ReviewRepository};

pub struct SqlxChatRoomRepository {
    pool: PgPool,
}

impl SqlxChatRoomRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl ChatRoomRepository for SqlxChatRoomRepository {
    async fn save(&self, chat_room: &ChatRoom) -> Result<(), DomainError> {
        let mut tx: Transaction<'_, Postgres> = self.pool.begin().await
            .map_err(|e| DomainError::ChatRoomNotFound(e.to_string()))?; // generic DB error mapping

        // 1. Save/Update ChatRoom
        let status_str = chat_room.status.as_str();
        sqlx::query(
            r#"
            INSERT INTO chat_rooms (room_id, booking_id, client_id, companion_id, status, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7)
            ON CONFLICT (room_id) DO UPDATE SET
                status = EXCLUDED.status,
                updated_at = EXCLUDED.updated_at
            "#,
        )
        .bind(&chat_room.room_id)
        .bind(&chat_room.booking_id)
        .bind(&chat_room.client_id)
        .bind(&chat_room.companion_id)
        .bind(status_str)
        .bind(chat_room.created_at)
        .bind(chat_room.updated_at)
        .execute(&mut *tx)
        .await
        .map_err(|e| DomainError::ChatRoomNotFound(e.to_string()))?;

        // 2. Publish transactional event based on status
        let (event_id, event_type, payload) = match chat_room.status {
            ChatRoomStatus::Active => {
                let event = ChatRoomCreatedEvent {
                    room_id: chat_room.room_id.clone(),
                    booking_id: chat_room.booking_id.clone(),
                    client_id: chat_room.client_id.clone(),
                    companion_id: chat_room.companion_id.clone(),
                    occurred_at: Utc::now(),
                };
                (
                    Uuid::new_v4().to_string(),
                    "com.rentagf.interaction.ChatRoomCreated.v1".to_string(),
                    serde_json::to_value(&event).unwrap(),
                )
            }
            ChatRoomStatus::Locked => {
                let event = ChatRoomLockedEvent {
                    room_id: chat_room.room_id.clone(),
                    booking_id: chat_room.booking_id.clone(),
                    occurred_at: Utc::now(),
                };
                (
                    Uuid::new_v4().to_string(),
                    "com.rentagf.interaction.ChatRoomLocked.v1".to_string(),
                    serde_json::to_value(&event).unwrap(),
                )
            }
        };

        let payload_str = payload.to_string();
        let now = Utc::now();
        sqlx::query(
            r#"
            INSERT INTO outbox (event_id, event_type, payload, processed, created_at)
            VALUES ($1, $2, $3, $4, $5)
            ON CONFLICT (event_id) DO NOTHING
            "#,
        )
        .bind(event_id)
        .bind(event_type)
        .bind(payload_str)
        .bind(false)
        .bind(now)
        .execute(&mut *tx)
        .await
        .map_err(|e| DomainError::ChatRoomNotFound(e.to_string()))?;

        tx.commit().await.map_err(|e| DomainError::ChatRoomNotFound(e.to_string()))?;
        Ok(())
    }

    async fn find_by_id(&self, room_id: &str) -> Result<Option<ChatRoom>, DomainError> {
        let row = sqlx::query(
            r#"
            SELECT room_id, booking_id, client_id, companion_id, status, created_at, updated_at
            FROM chat_rooms WHERE room_id = $1
            "#,
        )
        .bind(room_id)
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::ChatRoomNotFound(e.to_string()))?;

        if let Some(r) = row {
            let status_str: String = r.get("status");
            let status = ChatRoomStatus::from_str(&status_str)
                .map_err(|e| DomainError::ChatRoomNotFound(e))?;
            Ok(Some(ChatRoom::new(
                r.get("room_id"),
                r.get("booking_id"),
                r.get("client_id"),
                r.get("companion_id"),
                status,
                r.get("created_at"),
                r.get("updated_at"),
            )))
        } else {
            Ok(None)
        }
    }

    async fn find_by_booking_id(&self, booking_id: &str) -> Result<Option<ChatRoom>, DomainError> {
        let row = sqlx::query(
            r#"
            SELECT room_id, booking_id, client_id, companion_id, status, created_at, updated_at
            FROM chat_rooms WHERE booking_id = $1
            "#,
        )
        .bind(booking_id)
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::ChatRoomNotFound(e.to_string()))?;

        if let Some(r) = row {
            let status_str: String = r.get("status");
            let status = ChatRoomStatus::from_str(&status_str)
                .map_err(|e| DomainError::ChatRoomNotFound(e))?;
            Ok(Some(ChatRoom::new(
                r.get("room_id"),
                r.get("booking_id"),
                r.get("client_id"),
                r.get("companion_id"),
                status,
                r.get("created_at"),
                r.get("updated_at"),
            )))
        } else {
            Ok(None)
        }
    }

    async fn save_message(&self, message: &ChatMessage) -> Result<(), DomainError> {
        sqlx::query(
            r#"
            INSERT INTO chat_messages (message_id, room_id, sender_id, content, created_at)
            VALUES ($1, $2, $3, $4, $5)
            "#,
        )
        .bind(&message.message_id)
        .bind(&message.room_id)
        .bind(&message.sender_id)
        .bind(&message.content)
        .bind(message.created_at)
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::ChatRoomNotFound(e.to_string()))?;
        Ok(())
    }

    async fn get_messages(&self, room_id: &str, limit: i64, offset: i64) -> Result<Vec<ChatMessage>, DomainError> {
        let rows = sqlx::query(
            r#"
            SELECT message_id, room_id, sender_id, content, created_at
            FROM chat_messages WHERE room_id = $1
            ORDER BY created_at ASC
            LIMIT $2 OFFSET $3
            "#,
        )
        .bind(room_id)
        .bind(limit)
        .bind(offset)
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::ChatRoomNotFound(e.to_string()))?;

        let messages = rows.into_iter().map(|r| ChatMessage {
            message_id: r.get("message_id"),
            room_id: r.get("room_id"),
            sender_id: r.get("sender_id"),
            content: r.get("content"),
            created_at: r.get("created_at"),
        }).collect();

        Ok(messages)
    }
}

pub struct SqlxReviewRepository {
    pool: PgPool,
}

impl SqlxReviewRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl ReviewRepository for SqlxReviewRepository {
    async fn save(&self, review: &Review) -> Result<(), DomainError> {
        let mut tx = self.pool.begin().await
            .map_err(|e| DomainError::ReviewNotFound(e.to_string()))?;

        // 1. Save/Update Review
        let rating_val = review.rating.value();
        sqlx::query(
            r#"
            INSERT INTO reviews (review_id, booking_id, client_id, companion_id, rating, comment, is_visible, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
            ON CONFLICT (review_id) DO UPDATE SET
                is_visible = EXCLUDED.is_visible,
                updated_at = EXCLUDED.updated_at
            "#,
        )
        .bind(&review.review_id)
        .bind(&review.booking_id)
        .bind(&review.client_id)
        .bind(&review.companion_id)
        .bind(rating_val)
        .bind(&review.comment)
        .bind(review.is_visible)
        .bind(review.created_at)
        .bind(review.updated_at)
        .execute(&mut *tx)
        .await
        .map_err(|e| DomainError::ReviewNotFound(e.to_string()))?;

        // 2. Publish transactional event based on visibility
        let (event_id, event_type, payload) = if review.is_visible {
            let event = ReviewSubmittedEvent {
                review_id: review.review_id.clone(),
                booking_id: review.booking_id.clone(),
                client_id: review.client_id.clone(),
                companion_id: review.companion_id.clone(),
                rating: rating_val,
                comment: review.comment.clone(),
                occurred_at: Utc::now(),
            };
            (
                Uuid::new_v4().to_string(),
                "com.rentagf.interaction.ReviewSubmitted.v1".to_string(),
                serde_json::to_value(&event).unwrap(),
            )
        } else {
            let event = ReviewHiddenEvent {
                review_id: review.review_id.clone(),
                booking_id: review.booking_id.clone(),
                occurred_at: Utc::now(),
            };
            (
                Uuid::new_v4().to_string(),
                "com.rentagf.interaction.ReviewHidden.v1".to_string(),
                serde_json::to_value(&event).unwrap(),
            )
        };

        let payload_str = payload.to_string();
        let now = Utc::now();
        sqlx::query(
            r#"
            INSERT INTO outbox (event_id, event_type, payload, processed, created_at)
            VALUES ($1, $2, $3, $4, $5)
            ON CONFLICT (event_id) DO NOTHING
            "#,
        )
        .bind(event_id)
        .bind(event_type)
        .bind(payload_str)
        .bind(false)
        .bind(now)
        .execute(&mut *tx)
        .await
        .map_err(|e| DomainError::ReviewNotFound(e.to_string()))?;

        tx.commit().await.map_err(|e| DomainError::ReviewNotFound(e.to_string()))?;
        Ok(())
    }

    async fn find_by_id(&self, review_id: &str) -> Result<Option<Review>, DomainError> {
        let row = sqlx::query(
            r#"
            SELECT review_id, booking_id, client_id, companion_id, rating, comment, is_visible, created_at, updated_at
            FROM reviews WHERE review_id = $1
            "#,
        )
        .bind(review_id)
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::ReviewNotFound(e.to_string()))?;

        if let Some(r) = row {
            let rating_val: i32 = r.get("rating");
            let rating = Rating::new(rating_val)?;
            Ok(Some(Review::new(
                r.get("review_id"),
                r.get("booking_id"),
                r.get("client_id"),
                r.get("companion_id"),
                rating,
                r.get("comment"),
                r.get("is_visible"),
                r.get("created_at"),
                r.get("updated_at"),
            )))
        } else {
            Ok(None)
        }
    }

    async fn find_by_booking_id(&self, booking_id: &str) -> Result<Option<Review>, DomainError> {
        let row = sqlx::query(
            r#"
            SELECT review_id, booking_id, client_id, companion_id, rating, comment, is_visible, created_at, updated_at
            FROM reviews WHERE booking_id = $1
            "#,
        )
        .bind(booking_id)
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::ReviewNotFound(e.to_string()))?;

        if let Some(r) = row {
            let rating_val: i32 = r.get("rating");
            let rating = Rating::new(rating_val)?;
            Ok(Some(Review::new(
                r.get("review_id"),
                r.get("booking_id"),
                r.get("client_id"),
                r.get("companion_id"),
                rating,
                r.get("comment"),
                r.get("is_visible"),
                r.get("created_at"),
                r.get("updated_at"),
            )))
        } else {
            Ok(None)
        }
    }

    async fn find_by_companion_id(&self, companion_id: &str) -> Result<Vec<Review>, DomainError> {
        let rows = sqlx::query(
            r#"
            SELECT review_id, booking_id, client_id, companion_id, rating, comment, is_visible, created_at, updated_at
            FROM reviews WHERE companion_id = $1 AND is_visible = TRUE
            ORDER BY created_at DESC
            "#,
        )
        .bind(companion_id)
        .fetch_all(&self.pool)
        .await
        .map_err(|e| DomainError::ReviewNotFound(e.to_string()))?;

        let mut reviews = Vec::new();
        for r in rows {
            let rating_val: i32 = r.get("rating");
            let rating = Rating::new(rating_val)?;
            reviews.push(Review::new(
                r.get("review_id"),
                r.get("booking_id"),
                r.get("client_id"),
                r.get("companion_id"),
                rating,
                r.get("comment"),
                r.get("is_visible"),
                r.get("created_at"),
                r.get("updated_at"),
            ));
        }
        Ok(reviews)
    }
}
