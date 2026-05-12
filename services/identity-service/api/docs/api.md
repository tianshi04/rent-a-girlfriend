# Identity Service API Documentation & Developer Guide

Welcome to the Identity Service developer onboarding guide. This service handles authentication, user accounts, and companion upgrade requests.

## 1. REST API

The REST API is used for synchronous client interactions, such as login and administrative tasks.

- **OpenAPI Specification**: [openapi.yaml](../api/openapi/openapi.yaml)
- **Base Path**: `/api/v1`

### Key Endpoints

| Method | Path | Description | Roles |
|--------|------|-------------|-------|
| `GET` | `/auth/google/init` | Start Google OAuth flow | Public |
| `POST` | `/auth/google/callback` | Exchange OAuth code for tokens | Public |
| `POST` | `/upgrade-requests` | Request companion status | Authenticated |
| `GET` | `/admin/upgrade-requests` | List all upgrade requests | Admin |
| `PUT` | `/admin/accounts/:id/lock` | Lock a user account | Admin |

### Authentication
JWT verification is offloaded to the **Istio Waypoint**. The service expects the following headers to be injected:
- `X-User-Id`: The UUID of the authenticated user.
- `X-User-Role`: The role of the user (`CLIENT`, `COMPANION`, `ADMIN`).

---

## 2. gRPC Interface

The gRPC interface is used for high-performance internal communication between microservices.

- **Proto Definition**: [identity.proto](../api/proto/identity.proto)
- **Port**: `50051`

### Available Services

`IdentityService` implements methods for fetching account details and managing lifecycle commands.

```protobuf
service IdentityService {
  rpc GetAccount(GetAccountRequest) returns (AccountResponse);
  rpc LockAccount(LockAccountRequest) returns (MessageResponse);
  // ...
}
```

---

## 3. Events (AsyncAPI)

The Identity Service publishes domain events to Kafka for asynchronous integration.

- **AsyncAPI Specification**: [asyncapi.yaml](../api/asyncapi/asyncapi.yaml)

### Core Events

| Event Type | Description | Trigger |
|------------|-------------|---------|
| `com.rentagf.identity.UserRegistered.v1` | New user created | First login via Google |
| `com.rentagf.identity.AccountLocked.v1` | Account locked | Admin action or violation threshold |
| `com.rentagf.identity.RoleUpgraded.v1` | Role changed to COMPANION | Admin approval of request |

---

## 4. Local Development

### Configuration
Update the [`.env`](../.env) file with your local database and OAuth credentials.

### Run Server
```bash
go run cmd/server/main.go
```

### Run Migrations
Migrations are applied automatically on server startup from the `migrations/` directory.
```bash
# Manual run (optional)
go run cmd/migrate/main.go
```
