pub mod application;
pub mod domain;
pub mod infrastructure;
pub mod interfaces;

use sqlx::postgres::PgPoolOptions;
use std::net::SocketAddr;
use std::sync::Arc;
use std::time::Duration;
use tokio::net::TcpListener;
use tonic::transport::Server;
use tracing::{error, info, warn};
use tracing_subscriber::EnvFilter;

use crate::application::chat_use_cases::ChatUseCases;
use crate::application::review_use_cases::ReviewUseCases;
use crate::infrastructure::broker::OutboxWorker;
use crate::infrastructure::persistence::{SqlxChatRoomRepository, SqlxReviewRepository};
use crate::interfaces::grpc::servicer::proto::interaction_service_server::InteractionServiceServer;
use crate::interfaces::grpc::servicer::InteractionServicer;
use crate::interfaces::http::{create_router, AppState};
use crate::interfaces::listeners::BookingEventListener;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // 1. Initialize dotenv & Logging
    dotenvy::dotenv().ok();

    // Set RUST_LOG environment filter, fallback to info
    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::from_default_env())
        .init();

    info!("Starting Rent-a-Girlfriend: Interaction Service (Rust)...");

    // 2. Load Configurations from Env
    let app_env = std::env::var("APP_ENV").unwrap_or_else(|_| "development".to_string());
    let server_port = std::env::var("SERVER_PORT").unwrap_or_else(|_| "8080".to_string());
    let grpc_port = std::env::var("GRPC_PORT").unwrap_or_else(|_| "50051".to_string());

    let db_host = std::env::var("DB_HOST").unwrap_or_else(|_| "localhost".to_string());
    let db_port = std::env::var("DB_PORT").unwrap_or_else(|_| "5432".to_string());
    let db_user = std::env::var("DB_USER").unwrap_or_else(|_| "postgres".to_string());
    let db_password = std::env::var("DB_PASSWORD").unwrap_or_else(|_| "postgres".to_string());
    let db_name = std::env::var("DB_NAME").unwrap_or_else(|_| "interaction_service".to_string());
    let db_sslmode = std::env::var("DB_SSLMODE").unwrap_or_else(|_| "disable".to_string());

    let kafka_brokers =
        std::env::var("KAFKA_BROKERS").unwrap_or_else(|_| "localhost:9092".to_string());
    let kafka_topic_interaction = std::env::var("KAFKA_TOPIC_INTERACTION")
        .unwrap_or_else(|_| "interaction-events".to_string());
    let kafka_topic_booking =
        std::env::var("KAFKA_TOPIC_BOOKING").unwrap_or_else(|_| "booking-events".to_string());
    let kafka_group_id =
        std::env::var("KAFKA_GROUP_ID").unwrap_or_else(|_| "interaction-service-group".to_string());

    let outbox_polling_ms = std::env::var("OUTBOX_POLLING_INTERVAL_MS")
        .unwrap_or_else(|_| "500".to_string())
        .parse::<u64>()
        .unwrap_or(500);
    let outbox_batch_size = std::env::var("OUTBOX_BATCH_SIZE")
        .unwrap_or_else(|_| "50".to_string())
        .parse::<i64>()
        .unwrap_or(50);

    info!("Configuration Loaded. Environment: {}", app_env);

    // 3. Database Connection Pool & Auto-Migrations
    let db_url = format!(
        "postgres://{}:{}@{}:{}/{}?sslmode={}",
        db_user, db_password, db_host, db_port, db_name, db_sslmode
    );

    info!("Connecting to PostgreSQL Database...");
    let pool = PgPoolOptions::new()
        .max_connections(10)
        .acquire_timeout(Duration::from_secs(5))
        .connect(&db_url)
        .await
        .map_err(|e| {
            error!("Failed to connect to PostgreSQL database: {:?}", e);
            e
        })?;

    info!("Successfully connected to database. Running migrations...");
    sqlx::migrate!("./migrations")
        .run(&pool)
        .await
        .map_err(|e| {
            error!("Database migration failed: {:?}", e);
            e
        })?;
    info!("Database migrations executed successfully.");

    // 4. Domain / Application / Persistence Assembly
    let chat_repo = Arc::new(SqlxChatRoomRepository::new(pool.clone()));
    let review_repo = Arc::new(SqlxReviewRepository::new(pool.clone()));

    let chat_cases = Arc::new(ChatUseCases::new(chat_repo));
    let review_cases = Arc::new(ReviewUseCases::new(review_repo));

    // 5. Start Background Workers
    // A. Outbox Worker
    let outbox_worker = Arc::new(OutboxWorker::new(
        pool.clone(),
        kafka_brokers.clone(),
        kafka_topic_interaction,
        Duration::from_millis(outbox_polling_ms),
        outbox_batch_size,
    ));
    let outbox_handle = tokio::spawn(async move {
        outbox_worker.start().await;
    });

    // B. Booking Event Listener
    let booking_listener = Arc::new(BookingEventListener::new(
        chat_cases.clone(),
        kafka_brokers.clone(),
        kafka_topic_booking,
        kafka_group_id,
    ));
    let listener_handle = tokio::spawn(async move {
        booking_listener.start().await;
    });

    // 6. Bind Interfaces
    // A. Axum HTTP Router & Server
    let app_state = AppState {
        chat_cases: chat_cases.clone(),
        review_cases: review_cases.clone(),
    };
    let http_router = create_router(app_state);
    let http_addr: SocketAddr = format!("0.0.0.0:{}", server_port).parse()?;

    info!("Binding Axum HTTP REST server to: {}", http_addr);
    let http_listener = TcpListener::bind(http_addr).await?;
    let http_server = axum::serve(http_listener, http_router);

    // B. Tonic gRPC Servicer & Server
    let grpc_servicer = InteractionServicer::new(chat_cases, review_cases);
    let grpc_addr: SocketAddr = format!("0.0.0.0:{}", grpc_port).parse()?;
    info!("Binding Tonic gRPC server to: {}", grpc_addr);
    let grpc_server = Server::builder()
        .add_service(InteractionServiceServer::new(grpc_servicer))
        .serve(grpc_addr);

    // 7. Concurrent Startup & Graceful Shutdown
    let shutdown = async {
        tokio::signal::ctrl_c()
            .await
            .expect("Failed to register Ctrl+C handler");
        info!("Shutdown signal (Ctrl+C) received. Stopping services gracefully...");
    };

    tokio::select! {
        res = http_server => {
            if let Err(e) = res {
                error!("HTTP server encountered an error: {:?}", e);
            }
        }
        res = grpc_server => {
            if let Err(e) = res {
                error!("gRPC server encountered an error: {:?}", e);
            }
        }
        _ = outbox_handle => {
            warn!("Background Transactional Outbox worker exited.");
        }
        _ = listener_handle => {
            warn!("Background Kafka Booking listener exited.");
        }
        _ = shutdown => {
            info!("Graceful shutdown complete. Exiting.");
        }
    }

    Ok(())
}
