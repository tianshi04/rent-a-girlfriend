use chrono::Utc;
use cloudevents::{EventBuilder, EventBuilderV10};
use rdkafka::config::ClientConfig;
use rdkafka::producer::{FutureProducer, FutureRecord};
use serde_json::Value;
use sqlx::{PgPool, Row};
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::Notify;
use tokio::time::sleep;
use tracing::{error, info, warn};
use url::Url;

struct OutboxEvent {
    id: i64,
    event_id: String,
    event_type: String,
    payload: String,
    created_at: chrono::DateTime<chrono::Utc>,
}

pub struct OutboxWorker {
    pool: PgPool,
    kafka_brokers: String,
    topic: String,
    polling_interval: Duration,
    batch_size: i64,
    outbox_notify: Arc<Notify>,
}

impl OutboxWorker {
    pub fn new(
        pool: PgPool,
        kafka_brokers: String,
        topic: String,
        polling_interval: Duration,
        batch_size: i64,
        outbox_notify: Arc<Notify>,
    ) -> Self {
        Self {
            pool,
            kafka_brokers,
            topic,
            polling_interval,
            batch_size,
            outbox_notify,
        }
    }

    pub async fn start(self: Arc<Self>, mut shutdown_rx: tokio::sync::watch::Receiver<bool>) {
        info!("Starting Transactional Outbox Worker...");

        // Build FutureProducer from ClientConfig
        let producer: FutureProducer = match ClientConfig::new()
            .set("bootstrap.servers", &self.kafka_brokers)
            .set("message.timeout.ms", "5000")
            .create()
        {
            Ok(p) => p,
            Err(e) => {
                error!(
                    "Failed to create Kafka producer for Outbox: {}. Background publishing disabled.",
                    e
                );
                return;
            }
        };

        let producer = Arc::new(producer);

        loop {
            if *shutdown_rx.borrow() {
                info!("Outbox worker received shutdown signal. Exiting loop.");
                break;
            }

            let mut processed = 0;
            match self.process_batch(&producer).await {
                Ok(count) => {
                    processed = count;
                }
                Err(e) => {
                    error!("Error in Outbox process batch: {}", e);
                }
            }

            // Only sleep if no events were processed to avoid CPU spam and maximize throughput.
            if processed == 0 {
                tokio::select! {
                    _ = sleep(self.polling_interval) => {}
                    _ = self.outbox_notify.notified() => {
                        info!("Outbox worker proactively woken up by new event signal.");
                    }
                    _ = shutdown_rx.changed() => {
                        info!("Outbox worker received shutdown signal. Exiting loop.");
                        break;
                    }
                }
            }
        }
    }

    async fn process_batch(&self, producer: &FutureProducer) -> Result<usize, sqlx::Error> {
        let mut tx = self.pool.begin().await?;
        let now = Utc::now();
        let lock_duration = chrono::Duration::minutes(1);
        let locked_until = now + lock_duration;

        // 1. Fetch unprocessed and unlocked (or lock-expired) rows in FIFO order
        let rows = sqlx::query(
            r#"
            SELECT id, event_id, event_type, payload, created_at
            FROM outbox
            WHERE processed = false AND (locked_until IS NULL OR locked_until < $1)
            ORDER BY id ASC
            LIMIT $2
            FOR UPDATE SKIP LOCKED
            "#,
        )
        .bind(now)
        .bind(self.batch_size)
        .fetch_all(&mut *tx)
        .await?;

        if rows.is_empty() {
            tx.commit().await?;
            return Ok(0);
        }

        let ids: Vec<i64> = rows.iter().map(|row| row.get::<i64, _>("id")).collect();

        // 2. Lock the rows immediately in a fast update
        sqlx::query(
            r#"
            UPDATE outbox
            SET locked_until = $1
            WHERE id = ANY($2)
            "#,
        )
        .bind(locked_until)
        .bind(&ids)
        .execute(&mut *tx)
        .await?;

        // Map database rows to in-memory events before committing to release the database connection
        let events: Vec<OutboxEvent> = rows
            .into_iter()
            .map(|row| OutboxEvent {
                id: row.get("id"),
                event_id: row.get("event_id"),
                event_type: row.get("event_type"),
                payload: row.get("payload"),
                created_at: row.get("created_at"),
            })
            .collect();

        // Commit transaction immediately to release PostgreSQL locks
        tx.commit().await?;

        info!("Fetched and locked {} events from outbox", events.len());

        let mut processed_count = 0;
        for event in events {
            // Parse event data from outbox payload
            let data: Value = serde_json::from_str(&event.payload).unwrap_or(Value::Null);

            // Construct standard CloudEvents envelope using the official SDK
            let source_url = Url::parse("http://rent-a-gf/interaction-service").unwrap();
            let cloudevent = EventBuilderV10::new()
                .id(&event.event_id)
                .source(source_url)
                .ty(&event.event_type)
                .time(event.created_at)
                .data("application/json", data)
                .build()
                .map_err(|e| sqlx::Error::Protocol(e.to_string()))?;

            let cloudevent_str = serde_json::to_string(&cloudevent)
                .map_err(|e| sqlx::Error::Protocol(e.to_string()))?;

            // 3. Publish to Kafka topic asynchronously (completely outside DB transaction)
            let record = FutureRecord::to(&self.topic)
                .key(&event.event_id)
                .payload(&cloudevent_str);

            match producer.send(record, Duration::from_secs(3)).await {
                Ok(_) => {
                    // 4. Mark event as processed atomically in database
                    sqlx::query(
                        r#"
                        UPDATE outbox
                        SET processed = true, locked_until = NULL
                        WHERE id = $1
                        "#,
                    )
                    .bind(event.id)
                    .execute(&self.pool)
                    .await?;
                    info!(
                        "Successfully published and marked processed event: {}",
                        event.event_id
                    );
                    processed_count += 1;
                }
                Err((e, _)) => {
                    warn!(
                        "Failed to publish event {} to Kafka: {}. Releasing lock to retry.",
                        event.event_id, e
                    );
                    // Release the lock immediately so another poll cycle or another worker can pick it up
                    if let Err(unlock_err) = sqlx::query(
                        r#"
                        UPDATE outbox
                        SET locked_until = NULL
                        WHERE id = $1
                        "#,
                    )
                    .bind(event.id)
                    .execute(&self.pool)
                    .await
                    {
                        error!(
                            "Failed to release lock for event {}: {}",
                            event.id, unlock_err
                        );
                    }
                    // Exit the batch loop and let the next polling cycle retry
                    break;
                }
            }
        }

        Ok(processed_count)
    }
}
