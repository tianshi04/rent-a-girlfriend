use crate::application::ports::{ChatRoomRepository, ProcessedEventRepository, ReviewRepository};
use crate::domain::chat_room::{ChatMessage, ChatRoom, ChatRoomStatus};
use crate::domain::errors::DomainError;

use crate::domain::review::Review;
use crate::domain::value_objects::Rating;
use async_trait::async_trait;
use chrono::Utc;
use sqlx::{PgPool, Postgres, Row, Transaction};
use std::str::FromStr;
use std::sync::Arc;
use tokio::sync::Notify;
use uuid::Uuid;

pub struct SqlxChatRoomRepository {
    pool: PgPool,
    outbox_notify: Arc<Notify>,
}

impl SqlxChatRoomRepository {
    pub fn new(pool: PgPool, outbox_notify: Arc<Notify>) -> Self {
        Self {
            pool,
            outbox_notify,
        }
    }
}

#[async_trait]
impl ChatRoomRepository for SqlxChatRoomRepository {
    async fn save(&self, chat_room: &ChatRoom) -> Result<(), DomainError> {
        let mut tx: Transaction<'_, Postgres> = self
            .pool
            .begin()
            .await
            .map_err(|e| DomainError::DatabaseError(e.to_string()))?; // generic DB error mapping

        // 1. Save/Update ChatRoom
        let status_str = chat_room.status.as_str();
        sqlx::query(
            r#"
            INSERT INTO chat_rooms (room_id, booking_id, client_id, companion_id, status, lock_at, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
            ON CONFLICT (room_id) DO UPDATE SET
                status = EXCLUDED.status,
                lock_at = EXCLUDED.lock_at,
                updated_at = EXCLUDED.updated_at
            "#,
        )
        .bind(&chat_room.room_id)
        .bind(&chat_room.booking_id)
        .bind(&chat_room.client_id)
        .bind(&chat_room.companion_id)
        .bind(status_str)
        .bind(chat_room.lock_at)
        .bind(chat_room.created_at)
        .bind(chat_room.updated_at)
        .execute(&mut *tx)
        .await
        .map_err(|e| DomainError::DatabaseError(e.to_string()))?;

        // 2. Publish transactional event based on status
        let now_utc = Utc::now();
        let occurred_at = Some(prost_types::Timestamp {
            seconds: now_utc.timestamp(),
            nanos: now_utc.timestamp_subsec_nanos() as i32,
        });

        let (event_id, event_type, payload) = match chat_room.status {
            ChatRoomStatus::Active => {
                let event = crate::proto::ChatRoomCreated {
                    room_id: chat_room.room_id.clone(),
                    booking_id: chat_room.booking_id.clone(),
                    occurred_at,
                };
                (
                    Uuid::new_v4().to_string(),
                    "interaction.chat-room-created.v1".to_string(),
                    serde_json::to_value(&event).unwrap(),
                )
            }
            ChatRoomStatus::Locked => {
                let event = crate::proto::ChatRoomLocked {
                    room_id: chat_room.room_id.clone(),
                    booking_id: chat_room.booking_id.clone(),
                    occurred_at,
                };
                (
                    Uuid::new_v4().to_string(),
                    "interaction.chat-room-locked.v1".to_string(),
                    serde_json::to_value(&event).unwrap(),
                )
            }
        };

        let payload_str = payload.to_string();
        let now = Utc::now();
        sqlx::query(
            r#"
            INSERT INTO outbox (event_id, event_type, payload, booking_id, processed, created_at)
            VALUES ($1, $2, $3, $4, $5, $6)
            ON CONFLICT (event_id) DO NOTHING
            "#,
        )
        .bind(event_id)
        .bind(event_type)
        .bind(payload_str)
        .bind(&chat_room.booking_id)
        .bind(false)
        .bind(now)
        .execute(&mut *tx)
        .await
        .map_err(|e| DomainError::DatabaseError(e.to_string()))?;

        tx.commit()
            .await
            .map_err(|e| DomainError::DatabaseError(e.to_string()))?;
        self.outbox_notify.notify_one();
        Ok(())
    }

    async fn find_by_id(&self, room_id: &str) -> Result<Option<ChatRoom>, DomainError> {
        let row = sqlx::query(
            r#"
            SELECT room_id, booking_id, client_id, companion_id, status, lock_at, created_at, updated_at
            FROM chat_rooms WHERE room_id = $1
            "#,
        )
        .bind(room_id)
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::DatabaseError(e.to_string()))?;

        if let Some(r) = row {
            let status_str: String = r.get("status");
            let status =
                ChatRoomStatus::from_str(&status_str).map_err(DomainError::ChatRoomNotFound)?;
            Ok(Some(ChatRoom::new(
                r.get("room_id"),
                r.get("booking_id"),
                r.get("client_id"),
                r.get("companion_id"),
                status,
                r.get("lock_at"),
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
            SELECT room_id, booking_id, client_id, companion_id, status, lock_at, created_at, updated_at
            FROM chat_rooms WHERE booking_id = $1
            "#,
        )
        .bind(booking_id)
        .fetch_optional(&self.pool)
        .await
        .map_err(|e| DomainError::DatabaseError(e.to_string()))?;

        if let Some(r) = row {
            let status_str: String = r.get("status");
            let status =
                ChatRoomStatus::from_str(&status_str).map_err(DomainError::ChatRoomNotFound)?;
            Ok(Some(ChatRoom::new(
                r.get("room_id"),
                r.get("booking_id"),
                r.get("client_id"),
                r.get("companion_id"),
                status,
                r.get("lock_at"),
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
        .map_err(|e| DomainError::DatabaseError(e.to_string()))?;
        Ok(())
    }

    async fn get_messages(
        &self,
        room_id: &str,
        limit: i64,
        offset: i64,
    ) -> Result<Vec<ChatMessage>, DomainError> {
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
        .map_err(|e| DomainError::DatabaseError(e.to_string()))?;

        let messages = rows
            .into_iter()
            .map(|r| ChatMessage {
                message_id: r.get("message_id"),
                room_id: r.get("room_id"),
                sender_id: r.get("sender_id"),
                content: r.get("content"),
                created_at: r.get("created_at"),
            })
            .collect();

        Ok(messages)
    }

    async fn lock_expired_rooms(
        &self,
        now: chrono::DateTime<chrono::Utc>,
        limit: i64,
    ) -> Result<Vec<String>, DomainError> {
        let mut tx = self
            .pool
            .begin()
            .await
            .map_err(|e| DomainError::DatabaseError(e.to_string()))?;

        // 1. Select expired active rooms with FOR UPDATE SKIP LOCKED
        let rows = sqlx::query(
            r#"
            SELECT room_id, booking_id, client_id, companion_id, status, lock_at, created_at, updated_at
            FROM chat_rooms
            WHERE status = 'ACTIVE' AND lock_at <= $1
            ORDER BY lock_at ASC
            LIMIT $2
            FOR UPDATE SKIP LOCKED
            "#,
        )
        .bind(now)
        .bind(limit)
        .fetch_all(&mut *tx)
        .await
        .map_err(|e| DomainError::DatabaseError(e.to_string()))?;

        if rows.is_empty() {
            return Ok(Vec::new());
        }

        let mut locked_bookings = Vec::new();

        for r in rows {
            let room_id: String = r.get("room_id");
            let booking_id: String = r.get("booking_id");

            let status_locked = ChatRoomStatus::Locked.as_str();
            let now_utc = chrono::Utc::now();

            // Update status in chat_rooms to LOCKED
            sqlx::query(
                r#"
                UPDATE chat_rooms
                SET status = $1, updated_at = $2
                WHERE room_id = $3
                "#,
            )
            .bind(status_locked)
            .bind(now_utc)
            .bind(&room_id)
            .execute(&mut *tx)
            .await
            .map_err(|e| DomainError::DatabaseError(e.to_string()))?;

            // Insert ChatRoomLocked event into outbox
            let occurred_at = Some(prost_types::Timestamp {
                seconds: now_utc.timestamp(),
                nanos: now_utc.timestamp_subsec_nanos() as i32,
            });
            let event = crate::proto::ChatRoomLocked {
                room_id: room_id.clone(),
                booking_id: booking_id.clone(),
                occurred_at,
            };

            let event_id = Uuid::new_v4().to_string();
            let event_type = "interaction.chat-room-locked.v1".to_string();
            let payload = serde_json::to_value(&event).unwrap();
            let payload_str = payload.to_string();

            sqlx::query(
                r#"
                INSERT INTO outbox (event_id, event_type, payload, booking_id, processed, created_at)
                VALUES ($1, $2, $3, $4, $5, $6)
                "#,
            )
            .bind(event_id)
            .bind(event_type)
            .bind(payload_str)
            .bind(&booking_id)
            .bind(false)
            .bind(now_utc)
            .execute(&mut *tx)
            .await
            .map_err(|e| DomainError::DatabaseError(e.to_string()))?;

            locked_bookings.push(booking_id);
        }

        tx.commit()
            .await
            .map_err(|e| DomainError::DatabaseError(e.to_string()))?;
        self.outbox_notify.notify_one();

        Ok(locked_bookings)
    }

    async fn report_creation_failure(&self, booking_id: &str) -> Result<(), DomainError> {
        let now_utc = Utc::now();
        let occurred_at = Some(prost_types::Timestamp {
            seconds: now_utc.timestamp(),
            nanos: now_utc.timestamp_subsec_nanos() as i32,
        });

        let event = crate::proto::ChatRoomCreationFailed {
            booking_id: booking_id.to_string(),
            occurred_at,
        };

        let event_id = Uuid::new_v4().to_string();
        let event_type = "interaction.chat-room-creation-failed.v1".to_string();
        let payload = serde_json::to_value(&event).unwrap();
        let payload_str = payload.to_string();

        sqlx::query(
            r#"
            INSERT INTO outbox (event_id, event_type, payload, booking_id, processed, created_at)
            VALUES ($1, $2, $3, $4, $5, $6)
            ON CONFLICT (event_id) DO NOTHING
            "#,
        )
        .bind(event_id)
        .bind(event_type)
        .bind(payload_str)
        .bind(booking_id)
        .bind(false)
        .bind(now_utc)
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::DatabaseError(e.to_string()))?;

        self.outbox_notify.notify_one();
        Ok(())
    }
}

pub struct SqlxReviewRepository {
    pool: PgPool,
    outbox_notify: Arc<Notify>,
}

impl SqlxReviewRepository {
    pub fn new(pool: PgPool, outbox_notify: Arc<Notify>) -> Self {
        Self {
            pool,
            outbox_notify,
        }
    }
}

#[async_trait]
impl ReviewRepository for SqlxReviewRepository {
    async fn save(&self, review: &Review) -> Result<(), DomainError> {
        let mut tx = self
            .pool
            .begin()
            .await
            .map_err(|e| DomainError::DatabaseError(e.to_string()))?;

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
            .map_err(|e| DomainError::DatabaseError(e.to_string()))?;

        // 2. Publish transactional event based on visibility
        let now_utc = Utc::now();
        let occurred_at = Some(prost_types::Timestamp {
            seconds: now_utc.timestamp(),
            nanos: now_utc.timestamp_subsec_nanos() as i32,
        });

        let (event_id, event_type, payload) = if review.is_visible {
            let event = crate::proto::ReviewSubmitted {
                review_id: review.review_id.clone(),
                booking_id: review.booking_id.clone(),
                rating: rating_val,
                occurred_at,
            };
            (
                Uuid::new_v4().to_string(),
                "interaction.review-submitted.v1".to_string(),
                serde_json::to_value(&event).unwrap(),
            )
        } else {
            let event = crate::proto::ReviewHidden {
                review_id: review.review_id.clone(),
                booking_id: review.booking_id.clone(),
                occurred_at,
            };
            (
                Uuid::new_v4().to_string(),
                "interaction.review-hidden.v1".to_string(),
                serde_json::to_value(&event).unwrap(),
            )
        };

        let payload_str = payload.to_string();
        let now = Utc::now();
        sqlx::query(
            r#"
            INSERT INTO outbox (event_id, event_type, payload, booking_id, processed, created_at)
            VALUES ($1, $2, $3, $4, $5, $6)
            ON CONFLICT (event_id) DO NOTHING
            "#,
        )
        .bind(event_id)
        .bind(event_type)
        .bind(payload_str)
        .bind(&review.booking_id)
        .bind(false)
        .bind(now)
        .execute(&mut *tx)
        .await
        .map_err(|e| DomainError::DatabaseError(e.to_string()))?;

        tx.commit()
            .await
            .map_err(|e| DomainError::DatabaseError(e.to_string()))?;
        self.outbox_notify.notify_one();
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
        .map_err(|e| DomainError::DatabaseError(e.to_string()))?;

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
        .map_err(|e| DomainError::DatabaseError(e.to_string()))?;

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
        .map_err(|e| DomainError::DatabaseError(e.to_string()))?;

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

pub struct SqlxProcessedEventRepository {
    pool: PgPool,
}

impl SqlxProcessedEventRepository {
    pub fn new(pool: PgPool) -> Self {
        Self { pool }
    }
}

#[async_trait]
impl ProcessedEventRepository for SqlxProcessedEventRepository {
    async fn check_and_record(
        &self,
        event_id: &str,
        event_type: &str,
    ) -> Result<bool, DomainError> {
        let now = Utc::now();
        let result = sqlx::query(
            r#"
            INSERT INTO processed_events (event_id, event_type, processed_at)
            VALUES ($1, $2, $3)
            ON CONFLICT (event_id) DO NOTHING
            "#,
        )
        .bind(event_id)
        .bind(event_type)
        .bind(now)
        .execute(&self.pool)
        .await
        .map_err(|e| DomainError::DatabaseError(e.to_string()))?;

        Ok(result.rows_affected() == 0)
    }
}
