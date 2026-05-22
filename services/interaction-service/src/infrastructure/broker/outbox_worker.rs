use std::sync::Arc;
use std::time::Duration;
use tokio::time::sleep;
use sqlx::{PgPool, Row};
use rdkafka::producer::{FutureProducer, FutureRecord};
use rdkafka::config::ClientConfig;
use serde_json::Value;
use tracing::{info, error, warn};

pub struct OutboxWorker {
    pool: PgPool,
    kafka_brokers: String,
    topic: String,
    polling_interval: Duration,
    batch_size: i64,
}

impl OutboxWorker {
    pub fn new(
        pool: PgPool,
        kafka_brokers: String,
        topic: String,
        polling_interval: Duration,
        batch_size: i64,
    ) -> Self {
        Self {
            pool,
            kafka_brokers,
            topic,
            polling_interval,
            batch_size,
        }
    }

    pub async fn start(self: Arc<Self>) {
        info!("Starting Transactional Outbox Worker...");

        // Build FutureProducer from ClientConfig
        let producer: FutureProducer = match ClientConfig::new()
            .set("bootstrap.servers", &self.kafka_brokers)
            .set("message.timeout.ms", "5000")
            .create()
        {
            Ok(p) => p,
            Err(e) => {
                error!("Failed to create Kafka producer for Outbox: {}. Background publishing disabled.", e);
                return;
            }
        };

        let producer = Arc::new(producer);

        loop {
            if let Err(e) = self.process_batch(&producer).await {
                error!("Error in Outbox process batch: {}", e);
            }
            sleep(self.polling_interval).await;
        }
    }

    async fn process_batch(&self, producer: &FutureProducer) -> Result<(), sqlx::Error> {
        // 1. Fetch unprocessed rows in FIFO order
        let rows = sqlx::query(
            r#"
            SELECT id, event_id, event_type, payload, created_at
            FROM outbox
            WHERE processed = false
            ORDER BY id ASC
            LIMIT $1
            "#,
        )
        .bind(self.batch_size)
        .fetch_all(&self.pool)
        .await?;

        if rows.is_empty() {
            return Ok(());
        }

        info!("Fetched {} events from outbox to publish", rows.len());

        for row in rows {
            let id: i64 = row.get("id");
            let event_id: String = row.get("event_id");
            let event_type: String = row.get("event_type");
            let payload: String = row.get("payload");
            let created_at: chrono::DateTime<chrono::Utc> = row.get("created_at");

            // Parse event data from outbox payload
            let data: Value = serde_json::from_str(&payload)
                .unwrap_or(Value::Null);

            // Construct standard GFM CloudEvents JSON envelope
            let cloudevent = serde_json::json!({
                "specversion": "1.0",
                "id": event_id,
                "source": "/rent-a-gf/interaction-service",
                "type": event_type,
                "datacontenttype": "application/json",
                "time": created_at.to_rfc3339(),
                "data": data
            });

            let cloudevent_str = cloudevent.to_string();

            // 2. Publish to Kafka topic asynchronously
            let record = FutureRecord::to(&self.topic)
                .key(&event_id)
                .payload(&cloudevent_str);

            match producer.send(record, Duration::from_secs(3)).await {
                Ok(_) => {
                    // 3. Mark event as processed atomically in database
                    sqlx::query(
                        r#"
                        UPDATE outbox
                        SET processed = true
                        WHERE id = $1
                        "#,
                    )
                    .bind(id)
                    .execute(&self.pool)
                    .await?;
                    info!("Successfully published and marked processed event: {}", event_id);
                }
                Err((e, _)) => {
                    warn!("Failed to publish event {} to Kafka: {}. Retrying later.", event_id, e);
                    // Do not mark processed; exit batch loop and let the next polling cycle retry
                    break;
                }
            }
        }

        Ok(())
    }
}
