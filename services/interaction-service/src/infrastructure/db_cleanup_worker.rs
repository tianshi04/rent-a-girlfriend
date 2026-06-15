use chrono::Utc;
use sqlx::PgPool;
use std::sync::Arc;
use std::time::Duration;
use tokio::time::sleep;
use tracing::{error, info};

pub struct DbCleanupWorker {
    pool: PgPool,
    cleanup_interval: Duration,
    processed_events_retention: chrono::Duration,
    outbox_retention: chrono::Duration,
}

impl DbCleanupWorker {
    pub fn new(
        pool: PgPool,
        cleanup_interval: Duration,
        processed_events_retention: chrono::Duration,
        outbox_retention: chrono::Duration,
    ) -> Self {
        Self {
            pool,
            cleanup_interval,
            processed_events_retention,
            outbox_retention,
        }
    }

    pub async fn start(self: Arc<Self>, mut shutdown_rx: tokio::sync::watch::Receiver<bool>) {
        info!("Starting Database Cleanup Background Worker...");

        loop {
            if *shutdown_rx.borrow() {
                info!("Database cleanup worker received shutdown signal. Exiting loop.");
                break;
            }

            if let Err(e) = self.run_cleanup().await {
                error!("Error in DbCleanupWorker running cleanup: {:?}", e);
            }

            tokio::select! {
                _ = sleep(self.cleanup_interval) => {}
                _ = shutdown_rx.changed() => {
                    info!("Database cleanup worker received shutdown signal. Exiting loop.");
                    break;
                }
            }
        }
    }

    async fn run_cleanup(&self) -> Result<(), sqlx::Error> {
        let now = Utc::now();
        let processed_events_cutoff = now - self.processed_events_retention;
        let outbox_cutoff = now - self.outbox_retention;

        info!(
            "Running DB cleanup. Processed events cutoff: {}, Outbox cutoff: {}",
            processed_events_cutoff.to_rfc3339(),
            outbox_cutoff.to_rfc3339()
        );

        // 1. Delete old processed_events
        let deleted_events = sqlx::query(
            r#"
            DELETE FROM processed_events
            WHERE processed_at < $1
            "#,
        )
        .bind(processed_events_cutoff)
        .execute(&self.pool)
        .await?;

        if deleted_events.rows_affected() > 0 {
            info!(
                "Cleaned up {} old processed event IDs",
                deleted_events.rows_affected()
            );
        }

        // 2. Delete old processed outbox events
        let deleted_outbox = sqlx::query(
            r#"
            DELETE FROM outbox
            WHERE processed = true AND created_at < $1
            "#,
        )
        .bind(outbox_cutoff)
        .execute(&self.pool)
        .await?;

        if deleted_outbox.rows_affected() > 0 {
            info!(
                "Cleaned up {} old processed outbox records",
                deleted_outbox.rows_affected()
            );
        }

        Ok(())
    }
}
