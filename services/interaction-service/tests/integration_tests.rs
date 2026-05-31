use sqlx::postgres::PgPoolOptions;
use testcontainers_modules::{postgres::Postgres, testcontainers::runners::AsyncRunner};
use uuid::Uuid;

use interaction_service::application::ports::{ChatRoomRepository, ReviewRepository};
use interaction_service::domain::chat_room::ChatRoom;
use interaction_service::domain::review::Review;
use interaction_service::domain::value_objects::{ChatContent, Rating};
use interaction_service::infrastructure::persistence::{
    SqlxChatRoomRepository, SqlxReviewRepository,
};

// Helper function to spin up Postgres and return the pool and container handle
async fn setup_test_container() -> (sqlx::PgPool, impl Drop) {
    let container = Postgres::default()
        .start()
        .await
        .expect("Failed to start Postgres container");

    let host = container.get_host().await.unwrap();
    let port = container.get_host_port_ipv4(5432).await.unwrap();

    let db_url = format!("postgres://postgres:postgres@{host}:{port}/postgres");

    let pool = PgPoolOptions::new()
        .max_connections(5)
        .connect(&db_url)
        .await
        .expect("Failed to connect to test Postgres pool");

    // Run migrations
    sqlx::migrate!("./migrations")
        .run(&pool)
        .await
        .expect("Failed to run migrations on test container database");

    (pool, container)
}

#[tokio::test]
async fn test_chat_room_repository_integration() {
    let (pool, _container) = setup_test_container().await;
    let repo = SqlxChatRoomRepository::new(
        pool.clone(),
        std::sync::Arc::new(tokio::sync::Notify::new()),
    );

    let booking_id = Uuid::new_v4().to_string();
    let client_id = Uuid::new_v4().to_string();
    let companion_id = Uuid::new_v4().to_string();

    // 1. Create a ChatRoom
    let chat_room = ChatRoom::create(booking_id.clone(), client_id.clone(), companion_id.clone());
    let room_id = chat_room.room_id.clone();

    repo.save(&chat_room)
        .await
        .expect("Failed to save chat room");

    // 2. Find the ChatRoom by ID
    let found_by_id = repo
        .find_by_id(&room_id)
        .await
        .expect("Failed to find room by ID");
    assert!(found_by_id.is_some());
    let found = found_by_id.unwrap();
    assert_eq!(found.room_id, room_id);
    assert_eq!(found.booking_id, booking_id);
    assert_eq!(found.client_id, client_id);
    assert_eq!(found.companion_id, companion_id);

    // 3. Find the ChatRoom by Booking ID
    let found_by_booking = repo
        .find_by_booking_id(&booking_id)
        .await
        .expect("Failed to find room by booking ID");
    assert!(found_by_booking.is_some());
    assert_eq!(found_by_booking.unwrap().room_id, room_id);

    // 4. Verify that the outbox contains the created event
    let outbox_event: (String, String, bool) =
        sqlx::query_as("SELECT event_type, payload, processed FROM outbox WHERE event_type = $1")
            .bind("com.rentagf.interaction.ChatRoomCreated.v1")
            .fetch_one(&pool)
            .await
            .expect("Failed to fetch outbox event");

    assert_eq!(outbox_event.0, "com.rentagf.interaction.ChatRoomCreated.v1");
    assert!(!outbox_event.2); // processed should be false initially

    // 5. Save and get messages
    let msg1 = chat_room
        .send_message(&client_id, ChatContent::new("Hello companion!".to_string()))
        .expect("Failed to create chat message");
    repo.save_message(&msg1)
        .await
        .expect("Failed to save chat message");

    let msg2 = chat_room
        .send_message(&companion_id, ChatContent::new("Hello client!".to_string()))
        .expect("Failed to create chat message");
    repo.save_message(&msg2)
        .await
        .expect("Failed to save chat message");

    let messages = repo
        .get_messages(&room_id, 10, 0)
        .await
        .expect("Failed to retrieve messages");
    assert_eq!(messages.len(), 2);
    assert_eq!(messages[0].sender_id, client_id);
    assert_eq!(messages[0].content, "Hello companion!");
    assert_eq!(messages[1].sender_id, companion_id);
    assert_eq!(messages[1].content, "Hello client!");

    // 6. Lock room and verify status update + outbox lock event
    let mut mutable_room = found;
    mutable_room.lock().expect("Failed to lock room");
    repo.save(&mutable_room)
        .await
        .expect("Failed to update locked chat room");

    let updated_room = repo
        .find_by_id(&room_id)
        .await
        .expect("Failed to find room after lock")
        .unwrap();
    assert_eq!(
        updated_room.status,
        interaction_service::domain::chat_room::ChatRoomStatus::Locked
    );

    // Verify lock event in outbox
    let lock_event: (String, bool) =
        sqlx::query_as("SELECT event_type, processed FROM outbox WHERE event_type = $1")
            .bind("com.rentagf.interaction.ChatRoomLocked.v1")
            .fetch_one(&pool)
            .await
            .expect("Failed to fetch outbox lock event");
    assert_eq!(lock_event.0, "com.rentagf.interaction.ChatRoomLocked.v1");
}

#[tokio::test]
async fn test_review_repository_integration() {
    let (pool, _container) = setup_test_container().await;
    let repo = SqlxReviewRepository::new(
        pool.clone(),
        std::sync::Arc::new(tokio::sync::Notify::new()),
    );

    let booking_id = Uuid::new_v4().to_string();
    let client_id = Uuid::new_v4().to_string();
    let companion_id = Uuid::new_v4().to_string();
    let rating = Rating::new(5).unwrap();

    // 1. Submit a Review
    let review = Review::create(
        booking_id.clone(),
        client_id.clone(),
        companion_id.clone(),
        rating,
        "Absolutely amazing experience!".to_string(),
    );
    let review_id = review.review_id.clone();

    repo.save(&review).await.expect("Failed to save review");

    // 2. Find the Review by ID
    let found_by_id = repo
        .find_by_id(&review_id)
        .await
        .expect("Failed to find review by ID");
    assert!(found_by_id.is_some());
    let found = found_by_id.unwrap();
    assert_eq!(found.booking_id, booking_id);
    assert_eq!(found.client_id, client_id);
    assert_eq!(found.rating.value(), 5);
    assert_eq!(found.comment, "Absolutely amazing experience!");
    assert!(found.is_visible);

    // 3. Find the Review by Booking ID
    let found_by_booking = repo
        .find_by_booking_id(&booking_id)
        .await
        .expect("Failed to find review by booking ID");
    assert!(found_by_booking.is_some());
    assert_eq!(found_by_booking.unwrap().review_id, review_id);

    // 4. Find Reviews by Companion ID
    let companion_reviews = repo
        .find_by_companion_id(&companion_id)
        .await
        .expect("Failed to find companion reviews");
    assert_eq!(companion_reviews.len(), 1);
    assert_eq!(companion_reviews[0].review_id, review_id);

    // 5. Verify outbox contains ReviewSubmitted event
    let outbox_event: (String, bool) =
        sqlx::query_as("SELECT event_type, processed FROM outbox WHERE event_type = $1")
            .bind("com.rentagf.interaction.ReviewSubmitted.v1")
            .fetch_one(&pool)
            .await
            .expect("Failed to fetch review submitted outbox event");
    assert_eq!(outbox_event.0, "com.rentagf.interaction.ReviewSubmitted.v1");

    // 6. Hide the Review and verify hidden status + outbox hidden event
    let mut mutable_review = found;
    mutable_review.hide().expect("Failed to hide review");
    repo.save(&mutable_review)
        .await
        .expect("Failed to update hidden review");

    let updated_review = repo
        .find_by_id(&review_id)
        .await
        .expect("Failed to find review after hide")
        .unwrap();
    assert!(!updated_review.is_visible);

    // Fetch companion reviews should now return 0 visible reviews
    let companion_reviews_after_hide = repo
        .find_by_companion_id(&companion_id)
        .await
        .expect("Failed to get companion reviews");
    assert_eq!(companion_reviews_after_hide.len(), 0);

    // Verify outbox contains ReviewHidden event
    let hide_event: (String, bool) =
        sqlx::query_as("SELECT event_type, processed FROM outbox WHERE event_type = $1")
            .bind("com.rentagf.interaction.ReviewHidden.v1")
            .fetch_one(&pool)
            .await
            .expect("Failed to fetch review hidden outbox event");
    assert_eq!(hide_event.0, "com.rentagf.interaction.ReviewHidden.v1");
}
