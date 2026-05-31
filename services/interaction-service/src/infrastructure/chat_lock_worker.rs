use std::sync::Arc;
use std::time::Duration;
use tokio::time::sleep;
use tracing::{error, info};

use crate::application::chat_use_cases::ChatUseCases;

pub struct ChatLockWorker {
    chat_cases: Arc<ChatUseCases>,
    polling_interval: Duration,
    batch_size: i64,
}

impl ChatLockWorker {
    pub fn new(chat_cases: Arc<ChatUseCases>, polling_interval: Duration, batch_size: i64) -> Self {
        Self {
            chat_cases,
            polling_interval,
            batch_size,
        }
    }

    pub async fn start(self: Arc<Self>, mut shutdown_rx: tokio::sync::watch::Receiver<bool>) {
        info!("Starting Chat Lock Background Worker...");

        loop {
            if *shutdown_rx.borrow() {
                info!("Chat lock worker received shutdown signal. Exiting loop.");
                break;
            }

            if let Err(e) = self.process_expired_rooms().await {
                error!("Error in ChatLockWorker processing expired rooms: {:?}", e);
            }

            tokio::select! {
                _ = sleep(self.polling_interval) => {}
                _ = shutdown_rx.changed() => {
                    info!("Chat lock worker received shutdown signal. Exiting loop.");
                    break;
                }
            }
        }
    }

    async fn process_expired_rooms(&self) -> Result<(), crate::domain::errors::DomainError> {
        let locked_bookings = self.chat_cases.lock_expired_rooms(self.batch_size).await?;

        if !locked_bookings.is_empty() {
            info!(
                "Successfully locked {} expired active chat rooms: {:?}",
                locked_bookings.len(),
                locked_bookings
            );
        }

        Ok(())
    }
}
