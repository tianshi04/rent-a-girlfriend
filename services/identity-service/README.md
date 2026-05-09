# Identity Service

Microservice quản lý xác thực, phân quyền và định danh người dùng cho nền tảng Rent-a-Girlfriend.

## Chức năng chính
- **Google OAuth + PKCE**: Đăng nhập qua Google với PKCE Code Challenge
- **JWT Token Pair**: Access Token (RS256) + Refresh Token Rotation
- **JWKS Endpoint**: `/.well-known/jwks.json` cho Istio Waypoint xác thực tập trung
- **Companion Upgrade**: Client gửi yêu cầu nâng cấp → Admin duyệt
- **Chế tài vi phạm**: Tự động khóa tài khoản khi đạt ngưỡng (đọc từ DB)

## Tech Stack
- **Language**: Go 1.25
- **Framework**: Gin
- **Database**: PostgreSQL (GORM)
- **Architecture**: Hexagonal (Ports & Adapters)

## Quick Start

```bash
# Copy env
cp .env.example .env
# Edit .env with your Google OAuth credentials

# Run with Docker
make docker-up

# Or run locally (requires PostgreSQL)
make tidy
make run
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/.well-known/jwks.json` | JWKS public keys |
| GET | `/api/v1/auth/google/init` | PKCE init → auth URL |
| POST | `/api/v1/auth/google/callback` | OAuth callback → JWT pair |
| POST | `/api/v1/auth/refresh` | Refresh token rotation |
| POST | `/api/v1/auth/logout` | Revoke refresh token |
| POST | `/api/v1/upgrade-requests` | Request companion upgrade |
| GET | `/api/v1/accounts/:id` | Get account info |
| GET | `/api/v1/admin/upgrade-requests` | List upgrade requests |
| PUT | `/api/v1/admin/upgrade-requests/:id/approve` | Approve upgrade |
| PUT | `/api/v1/admin/upgrade-requests/:id/reject` | Reject upgrade |
| PUT | `/api/v1/admin/accounts/:id/lock` | Lock account |
| PUT | `/api/v1/admin/accounts/:id/unlock` | Unlock account |

## Testing

```bash
make test
```
