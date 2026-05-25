use chrono::{DateTime, Duration, Utc};
use rdkafka::config::ClientConfig;
use rdkafka::consumer::{Consumer, StreamConsumer};
use rdkafka::Message as KafkaMessage;
use serde::Deserialize;
use std::sync::Arc;
use tokio::time::sleep;
use tracing::{debug, error, info, warn};

use crate::application::chat_use_cases::ChatUseCases;

#[derive(Deserialize, Debug)]
struct BookingCloudEvent {
    id: String,
    #[serde(rename = "type")]
    event_type: String,
    data: BookingEventData,
}

#[derive(Deserialize, Debug)]
struct BookingEventData {
    #[serde(alias = "bookingId")]
    booking_id: String,
    #[serde(alias = "clientId")]
    client_id: Option<String>,
    #[serde(alias = "companionId")]
    companion_id: Option<String>,
    #[serde(alias = "endTime")]
    end_time: Option<String>,
}

pub struct BookingEventListener {
    chat_cases: Arc<ChatUseCases>,
    kafka_brokers: String,
    topic: String,
    group_id: String,
}

impl BookingEventListener {
    pub fn new(
        chat_cases: Arc<ChatUseCases>,
        kafka_brokers: String,
        topic: String,
        group_id: String,
    ) -> Self {
        Self {
            chat_cases,
            kafka_brokers,
            topic,
            group_id,
        }
    }

    pub async fn start(self: Arc<Self>) {
        info!("Starting Kafka Booking Event Listener...");

        let consumer: StreamConsumer = match ClientConfig::new()
            .set("bootstrap.servers", &self.kafka_brokers)
            .set("group.id", &self.group_id)
            .set("enable.partition.eof", "false")
            .set("session.timeout.ms", "6000")
            .set("enable.auto.commit", "true")
            .set("auto.offset.reset", "earliest")
            .create()
        {
            Ok(c) => c,
            Err(e) => {
                error!("Failed to create Kafka consumer for Booking Events: {}. Consumer loop disabled.", e);
                return;
            }
        };

        if let Err(e) = consumer.subscribe(&[&self.topic]) {
            error!(
                "Failed to subscribe to topic {}: {}. Consumer loop disabled.",
                self.topic, e
            );
            return;
        }

        loop {
            match consumer.recv().await {
                Ok(borrowed_message) => {
                    let payload = match borrowed_message.payload_view::<str>() {
                        None => continue,
                        Some(Ok(s)) => s,
                        Some(Err(e)) => {
                            error!("Error parsing message payload as string: {:?}", e);
                            continue;
                        }
                    };

                    if let Err(e) = self.handle_message(payload).await {
                        error!("Error handling booking event: {:?}", e);
                    }
                }
                Err(e) => {
                    error!("Error receiving message from Kafka: {:?}", e);
                    sleep(tokio::time::Duration::from_secs(2)).await;
                }
            }
        }
    }

    async fn handle_message(
        &self,
        payload: &str,
    ) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        let cloudevent: BookingCloudEvent = match serde_json::from_str(payload) {
            Ok(ce) => ce,
            Err(e) => {
                warn!(
                    "Ignoring non-compliant booking event. Failed to parse: {:?}",
                    e
                );
                return Ok(());
            }
        };

        info!(
            "Received booking event: {} with ID: {} for Booking ID: {}",
            cloudevent.event_type, cloudevent.id, cloudevent.data.booking_id
        );

        let booking_id = cloudevent.data.booking_id.clone();
        let chat_cases = self.chat_cases.clone();

        match cloudevent.event_type.as_str() {
            "com.rentagf.interaction.CreateChatRoom.v1" => {
                info!(
                    "CreateChatRoom command received for Booking ID: {}",
                    booking_id
                );
                if let (Some(client_id), Some(companion_id)) = (
                    cloudevent.data.client_id.clone(),
                    cloudevent.data.companion_id.clone(),
                ) {
                    match chat_cases
                        .create_chat_room(
                            booking_id.clone(),
                            client_id,
                            companion_id,
                            Some(cloudevent.id.clone()),
                        )
                        .await
                    {
                        Ok(_) => {
                            info!(
                                "Successfully created chat room via event for booking {}",
                                booking_id
                            );
                        }
                        Err(e) => {
                            error!(
                                "Failed to create chat room via event for booking {}: {:?}",
                                booking_id, e
                            );
                        }
                    }
                } else {
                    error!(
                        "Missing client_id or companion_id in CreateChatRoom command for booking {}",
                        booking_id
                    );
                }
            }
            "com.rentagf.booking.BookingCancelled.v1"
            | "com.rentagf.booking.BookingCancelledEarly.v1"
            | "com.rentagf.booking.BookingCancelledLate.v1" => {
                info!(
                    "Booking {} cancelled. Locking chat room immediately.",
                    booking_id
                );
                if let Err(e) = chat_cases
                    .lock_chat_room(&booking_id, Some(cloudevent.id.clone()))
                    .await
                {
                    error!(
                        "Failed to lock chat room for booking {}: {:?}",
                        booking_id, e
                    );
                }
            }
            "com.rentagf.booking.BookingCompleted.v1" => {
                let mut delay_seconds = 24 * 3600; // default 24 hours in seconds

                if let Some(end_time_str) = cloudevent.data.end_time {
                    if let Ok(end_time) = DateTime::parse_from_rfc3339(&end_time_str) {
                        let end_time_utc = end_time.with_timezone(&Utc);
                        let lock_time = end_time_utc + Duration::hours(24);
                        let now = Utc::now();
                        let diff = lock_time - now;
                        if diff.num_seconds() > 0 {
                            delay_seconds = diff.num_seconds() as u64;
                        } else {
                            delay_seconds = 0; // immediate lock since 24 hours has already passed
                        }
                    }
                }

                info!(
                    "Booking {} completed. Scheduling chat room lock in {} seconds (24h post-end).",
                    booking_id, delay_seconds
                );

                let event_id_for_task = cloudevent.id.clone();
                tokio::spawn(async move {
                    if delay_seconds > 0 {
                        sleep(tokio::time::Duration::from_secs(delay_seconds)).await;
                    }
                    info!("Executing scheduled lock for booking {}...", booking_id);
                    if let Err(e) = chat_cases
                        .lock_chat_room(&booking_id, Some(event_id_for_task))
                        .await
                    {
                        error!(
                            "Failed to lock chat room on completion schedule for booking {}: {:?}",
                            booking_id, e
                        );
                    } else {
                        info!(
                            "Successfully locked chat room on completion schedule for booking {}.",
                            booking_id
                        );
                    }
                });
            }
            _ => {
                // Ignore other event types
                debug!("Ignoring unrelated event type: {}", cloudevent.event_type);
            }
        }

        Ok(())
    }
}
