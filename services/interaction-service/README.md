# Interaction Service

A Supporting Bounded Context Microservice for the **Rent-a-Girlfriend** platform, implemented in **Rust** using async idiomatic best practices.

This service is responsible for:
1. **Booking Chat:** Provides a secure, transient communication channel between a Client and a Companion once a booking is accepted.
2. **Review & Rating System:** Allows Clients to submit a one-time rating (1-5 stars) and feedback comment once a booking scheduled time ends.

---

## 🚀 Technology Stack

- **Async Runtime:** [Tokio](https://tokio.rs/) (Multi-threaded async executor)
- **HTTP REST Framework:** [Axum](https://github.com/tokio-rs/axum) (v0.7)
- **gRPC Framework:** [Tonic](https://github.com/hyperium/tonic) (gRPC Client & Server)
- **Database Client:** [SQLx](https://github.com/launchbadge/sqlx) (PostgreSQL connection pool with compile-time query verification & async migrations)
- **Message Broker:** [rdkafka](https://github.com/fede1024/rust-rdkafka) (Librdkafka bindings for robust event consumption & publishing)
- **Serialization:** [Serde](https://serde.rs/) & [serde_json](https://github.com/serde-rs/json)

---

## 📁 Folder Structure

Adapting Domain-Driven Design (DDD) & Hexagonal Architecture:

```
services/interaction-service/
├── Cargo.toml
├── build.rs                  # Compile tonic gRPC proto files dynamically
├── Dockerfile
├── Makefile
├── .env.example
├── migrations/               # PostgreSQL DB schema migrations
│   └── 20260522000000_init_interaction.sql
└── src/
    ├── main.rs               # App bootstrapping and concurrency coordinator
    ├── domain/               # Core Bounded Context domain entities & rules
    │   ├── errors.rs         # Domain business errors mapping to HTTP/gRPC codes
    │   ├── value_objects.rs  # Rating (1-5) and ChatContent invariants
    │   ├── chat_room.rs      # ChatRoom Aggregate [INV-I01], [INV-I02], [INV-I03]
    │   ├── review.rs         # Review Aggregate [INV-I04], [INV-I05]
    │   └── events.rs         # CloudEvents-compliant domain event schemas
    ├── application/          # Pure orchestrator, use cases and port definitions
    │   ├── ports.rs          # Trait interfaces for repositories and publishers
    │   ├── chat_use_cases.rs # Coordinate booking chat workflows
    │   └── review_use_cases.rs # Coordinate companion reviews
    ├── infrastructure/       # Adapter implementations (DB persistence, Kafka broker)
    │   ├── persistence/
    │   │   └── repositories.rs # SQLx Postgres implementation with Outbox
    │   └── broker/
    │       └── outbox_worker.rs # Polling transactional outbox and publishing to Kafka
    └── interfaces/           # Outer layer exposing APIs and background listeners
        ├── grpc/
        │   └── servicer.rs   # Tonic gRPC handlers
        ├── http/
        │   ├── dto.rs        # Axum Request/Response payloads
        │   └── router.rs     # Axum routing & HTTP endpoints
        └── listeners/
            └── booking_event_listener.rs # Kafka booking events handler
```

---

## ⚙️ Configuration (.env)

| Variable | Description | Default |
| :--- | :--- | :--- |
| `SERVER_PORT` | HTTP REST API listening port | `8080` |
| `GRPC_PORT` | Tonic gRPC server listening port | `50051` |
| `APP_ENV` | Environment mode (`development`/`production`) | `development` |
| `RUST_LOG` | Logger filter configuration | `info` |
| `DB_HOST` | PostgreSQL Host | `localhost` |
| `DB_PORT` | PostgreSQL Port | `5432` |
| `DB_USER` | PostgreSQL Username | `postgres` |
| `DB_PASSWORD` | PostgreSQL Password | `postgres` |
| `DB_NAME` | PostgreSQL Database name | `interaction_service` |
| `DB_SSLMODE` | SSL requirement | `disable` |
| `KAFKA_BROKERS` | Kafka connection string | `localhost:9092` |
| `KAFKA_TOPIC_INTERACTION` | Kafka topic to publish events | `interaction-events` |
| `KAFKA_TOPIC_BOOKING` | Kafka topic to consume booking events | `booking-events` |
| `KAFKA_GROUP_ID` | Kafka consumer group ID | `interaction-service-group` |
| `OUTBOX_POLLING_INTERVAL_MS` | Outbox worker poll frequency | `500` |
| `OUTBOX_BATCH_SIZE` | Outbox processing batch size | `50` |

---

## 🛠️ Commands (Makefile)

Use `make` commands for easy local development:

```bash
# Run the service locally (loads .env automatically)
make run

# Run the full test suite
make test

# Build a release binary
make build
```

---

## 🔑 Authentication & Authorization (Istio Ambient Mode)

This microservice offloads JWT verification to the **Istio Waypoint Proxy** (Sidecar-less Service Mesh). The microservice expects the following headers to be injected by Istio on incoming requests:
- `user-id`: Identifies the authenticated client/companion.
- `user-role`: Identifies the user's role (e.g. `Client`, `Companion`).

---

## 📡 REST API Specifications

### Chat Messages

#### 1. Send Message
- **Endpoint:** `POST /api/v1/interaction/rooms/:room_id/messages`
- **Headers:**
  - `user-id` (Injected by Gateway, e.g. `client-uuid-1`)
- **Request Body:**
  ```json
  {
    "text": "Hello, excited for our appointment!"
  }
  ```
- **Response (201 Created):**
  ```json
  {
    "message_id": "message-uuid",
    "room_id": "room-uuid",
    "sender_id": "client-uuid-1",
    "content": "Hello, excited for our appointment!",
    "created_at": "2026-05-22T23:15:36Z"
  }
  ```

#### 2. Get Messages (Paginated)
- **Endpoint:** `GET /api/v1/interaction/rooms/:room_id/messages`
- **Query Params:**
  - `limit` (Default: `50`)
  - `offset` (Default: `0`)
- **Headers:**
  - `user-id` (Injected by Gateway, must belong to participants)
- **Response (200 OK):**
  ```json
  [
    {
      "message_id": "message-uuid",
      "room_id": "room-uuid",
      "sender_id": "client-uuid-1",
      "content": "Hello, excited for our appointment!",
      "created_at": "2026-05-22T23:15:36Z"
    }
  ]
  ```

---

### Reviews & Ratings

#### 1. Get Companion Reviews
- **Endpoint:** `GET /api/v1/interaction/reviews/companion/:companion_id`
- **Response (200 OK):**
  ```json
  [
    {
      "review_id": "review-uuid",
      "booking_id": "booking-uuid",
      "client_id": "client-uuid",
      "companion_id": "companion-uuid",
      "rating": 5,
      "comment": "Incredibly professional and friendly companion!",
      "created_at": "2026-05-22T23:15:36Z",
      "updated_at": "2026-05-22T23:15:36Z"
    }
  ]
  ```
