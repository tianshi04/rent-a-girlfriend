# Booking Service

**Rent-a-Girlfriend Platform** — Core Subdomain

Quản lý vòng đời (State Machine) của cuộc hẹn: Request → Accept/Reject → Cancel → Complete.

## Tech Stack
- **Language:** Go 1.23
- **Framework:** Gin
- **Database:** PostgreSQL 16 + GORM
- **Message Broker:** Apache Kafka (Phase 2)
- **Architecture:** Hexagonal (Ports & Adapters)

## Quick Start

```bash
# Run with Docker
docker compose up --build

# Run locally (requires PostgreSQL)
cp .env.example .env
go run ./cmd/server

# Run tests
go test ./internal/... -v
```

## API Endpoints

| Method | Path | Description |
|:---|:---|:---|
| `POST` | `/api/v1/bookings` | Create booking request |
| `GET` | `/api/v1/bookings` | List bookings (filters: clientId, companionId, status) |
| `GET` | `/api/v1/bookings/:id` | Get booking detail |
| `PUT` | `/api/v1/bookings/:id/accept` | Companion accepts |
| `PUT` | `/api/v1/bookings/:id/reject` | Companion rejects |
| `PUT` | `/api/v1/bookings/:id/cancel` | Cancel booking |
| `GET` | `/health` | Health check |

## Project Structure

```
cmd/server/           → Entry point
internal/
  bootstrap/          → Config, DB init, DI wiring
  domain/             → Aggregates, VOs, Events, Repository ports
  application/        → Command/Query handlers, External service ports
  infrastructure/     → GORM repository, Stub clients
  interfaces/http/    → Gin handlers, Router, DTOs
migrations/           → SQL migrations
```

## Phases
- **Phase 1** ✅ Domain + Application + REST API + PostgreSQL + Docker
- **Phase 2** 🔜 SAGA Orchestrator + Kafka + Outbox
- **Phase 3** 🔜 CQRS Projections + Auto-complete/Expiration workers
