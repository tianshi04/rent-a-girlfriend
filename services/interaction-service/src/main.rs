use sqlx::postgres::PgPoolOptions;
use std::net::SocketAddr;
use std::sync::Arc;
use std::time::Duration;
use tokio::net::TcpListener;
use tonic::transport::Server;
use tracing::{error, info, warn};
use tracing_subscriber::EnvFilter;

use interaction_service::application::chat_use_cases::ChatUseCases;
use interaction_service::application::review_use_cases::ReviewUseCases;
use interaction_service::infrastructure::broker::OutboxWorker;
use interaction_service::infrastructure::chat_lock_worker::ChatLockWorker;
use interaction_service::infrastructure::persistence::{
    SqlxChatRoomRepository, SqlxProcessedEventRepository, SqlxReviewRepository,
};
use interaction_service::interfaces::grpc::servicer::proto::interaction_service_server::InteractionServiceServer;
use interaction_service::interfaces::grpc::servicer::InteractionServicer;
use interaction_service::interfaces::http::{create_router, AppState};
use interaction_service::interfaces::listeners::BookingEventListener;

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

    let db_url = std::env::var("DATABASE_URL").unwrap_or_else(|_| {
        "postgres://postgres:postgres@localhost:5432/interaction_service?sslmode=disable"
            .to_string()
    });

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
    let processed_event_repo = Arc::new(SqlxProcessedEventRepository::new(pool.clone()));

    let chat_cases = Arc::new(ChatUseCases::new(chat_repo, processed_event_repo));
    let review_cases = Arc::new(ReviewUseCases::new(review_repo));

    // 5. Start Background Workers
    let (shutdown_tx, shutdown_rx) = tokio::sync::watch::channel(false);

    // A. Outbox Worker
    let outbox_worker = Arc::new(OutboxWorker::new(
        pool.clone(),
        kafka_brokers.clone(),
        kafka_topic_interaction,
        Duration::from_millis(outbox_polling_ms),
        outbox_batch_size,
    ));
    let outbox_shutdown_rx = shutdown_rx.clone();
    let mut outbox_handle = tokio::spawn(async move {
        outbox_worker.start(outbox_shutdown_rx).await;
    });

    // B. Booking Event Listener
    let booking_listener = Arc::new(BookingEventListener::new(
        chat_cases.clone(),
        kafka_brokers.clone(),
        kafka_topic_booking,
        kafka_group_id,
    ));
    let listener_shutdown_rx = shutdown_rx.clone();
    let mut listener_handle = tokio::spawn(async move {
        booking_listener.start(listener_shutdown_rx).await;
    });

    // C. Chat Lock Worker
    let chat_lock_worker = Arc::new(ChatLockWorker::new(
        chat_cases.clone(),
        Duration::from_secs(10), // poll every 10 seconds
        50,                      // batch size
    ));
    let chat_lock_shutdown_rx = shutdown_rx.clone();
    let mut chat_lock_handle = tokio::spawn(async move {
        chat_lock_worker.start(chat_lock_shutdown_rx).await;
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

    let mut http_shutdown_rx = shutdown_rx.clone();
    let http_server_fut =
        axum::serve(http_listener, http_router).with_graceful_shutdown(async move {
            let _ = http_shutdown_rx.changed().await;
            info!("Axum HTTP server graceful shutdown triggered.");
        });
    let mut http_handle = tokio::spawn(async move { http_server_fut.await });

    // B. Tonic gRPC Servicer & Server
    let grpc_servicer = InteractionServicer::new(chat_cases, review_cases);
    let grpc_addr: SocketAddr = format!("0.0.0.0:{}", grpc_port).parse()?;
    info!("Binding Tonic gRPC server to: {}", grpc_addr);

    let mut grpc_shutdown_rx = shutdown_rx.clone();
    let grpc_server_fut = Server::builder()
        .add_service(InteractionServiceServer::new(grpc_servicer))
        .serve_with_shutdown(grpc_addr, async move {
            let _ = grpc_shutdown_rx.changed().await;
            info!("Tonic gRPC server graceful shutdown triggered.");
        });
    let mut grpc_handle = tokio::spawn(grpc_server_fut);

    // 7. Concurrent Startup & Graceful Shutdown
    tokio::select! {
        res = &mut http_handle => {
            match res {
                Ok(Ok(())) => info!("HTTP server stopped."),
                Ok(Err(e)) => error!("HTTP server error: {:?}", e),
                Err(e) => error!("HTTP server join error: {:?}", e),
            }
        }
        res = &mut grpc_handle => {
            match res {
                Ok(Ok(())) => info!("gRPC server stopped."),
                Ok(Err(e)) => error!("gRPC server error: {:?}", e),
                Err(e) => error!("gRPC server join error: {:?}", e),
            }
        }
        _ = &mut outbox_handle => {
            warn!("Background Transactional Outbox worker exited unexpectedly.");
        }
        _ = &mut listener_handle => {
            warn!("Background Kafka Booking listener exited unexpectedly.");
        }
        _ = &mut chat_lock_handle => {
            warn!("Background Chat Lock worker exited unexpectedly.");
        }
        _ = shutdown_signal() => {
            info!("Shutdown signal received. Broadcasting shutdown to all tasks...");
            let _ = shutdown_tx.send(true);

            // Chờ tất cả kết thúc an toàn với thời gian timeout (15s)
            info!("Waiting for all tasks to complete gracefully...");

            let wait_all = async {
                let _ = tokio::join!(http_handle, grpc_handle, outbox_handle, listener_handle, chat_lock_handle);
            };

            tokio::select! {
                _ = wait_all => {
                    info!("All tasks shut down gracefully.");
                }
                _ = tokio::time::sleep(Duration::from_secs(15)) => {
                    warn!("Shutdown timed out. Some tasks did not exit in time. Forcing exit.");
                }
            }
        }
    }

    Ok(())
}

async fn shutdown_signal() {
    let ctrl_c = async {
        tokio::signal::ctrl_c()
            .await
            .expect("Failed to register Ctrl+C handler");
    };

    #[cfg(unix)]
    let terminate = async {
        tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())
            .expect("Failed to register SIGTERM handler")
            .recv()
            .await;
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {
            info!("Received SIGINT (Ctrl+C) signal.");
        }
        _ = terminate => {
            info!("Received SIGTERM signal.");
        }
    }
}
