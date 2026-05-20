# Profile & Catalogue Service

Microservice quản lý hồ sơ Companion, Scenarios (kịch bản hẹn hò) và Media Assets (Voice Intro, Album ảnh) đồng thời cung cấp cổng tìm kiếm & tra cứu hồ sơ Companion (Catalogue Search) cho nền tảng Rent-a-Girlfriend.

---

## Vai trò kép của Service

Service này được thiết kế để xử lý đồng thời hai vai trò cốt lõi trong hệ thống (Ghi và Đọc tách biệt qua REST/gRPC):

### 1. Profile & Media Management (Commands - Ghi/Thay đổi trạng thái)
- **Companion Profile**: CRUD thông tin cá nhân của Companion, bao gồm Thành phố hoạt động (chọn ít nhất 1 thành phố).
- **Service Scenarios**: Tạo lập kịch bản và giá dịch vụ tự chọn (Kano-Coin).
- **Media Asset Management**: Upload voice intro giới thiệu bản thân (tối đa 30s) và tối đa 4 ảnh album phụ trợ. Tích hợp presigned URL của **Object Storage (tương thích chuẩn S3)** để upload trực tiếp an toàn và tối ưu băng thông.
- **Admin Approval**: Cơ chế phê duyệt hồ sơ thủ công của Admin để Companion chính thức "lên sóng".

### 2. Catalogue Search & Magazine View (Queries - Đọc/Tra cứu)
- **Catalogue Search (REST)**: Cho phép Client tìm kiếm và lọc danh sách Companion theo Tên, Thành phố, Khoảng giá, Điểm đánh giá (Average Rating). Hỗ trợ phân trang hiệu năng cao bằng **PostgreSQL Full-Text Search**.
- **Magazine View (REST)**: Xem hồ sơ chi tiết (ảnh đại diện, voice intro, album, kịch bản, và các reviews liên quan) phục vụ quyết định đặt lịch của Client.
- **Scenario Snapshot (gRPC)**: Hỗ trợ query nội bộ nhanh chóng cho `Booking Service` lấy thông số kịch bản tại thời điểm Client đặt lịch.

---

## Business Invariants & Rules

Service chịu trách nhiệm thực thi các quy tắc nghiệp vụ bất biến sau:
- `[INV-P01]` Phí dịch vụ kịch bản (`price`) luôn lớn hơn `0` Kano-Coin.
- `[INV-P02]` Thời lượng kịch bản (`duration`) phải chuẩn hóa thuộc các block quy định: 60, 120, 180 phút.
- `[INV-P03]` Một Companion chỉ được tạo tối đa `5` Scenarios chủ động để tránh spam.
- `[INV-P04]` Độ dài Voice Intro không được vượt quá `30` giây.
- `[INV-P05]` Dung lượng Voice Intro không được vượt quá `5MB` (Định dạng MP3).
- `BR-12` Dung lượng ảnh album không được vượt quá `2MB` mỗi ảnh (Tối đa 4 ảnh album + 1 ảnh đại diện = 5 ảnh).

---

## Tech Stack

- **Language**: Python 3.12 (Quản lý qua `uv` CLI)
- **REST Framework**: FastAPI + Uvicorn (Phục vụ Catalogue Queries)
- **gRPC Framework**: `grpcio` (Phục vụ Commands & Internal Queries)
- **Database**: PostgreSQL (SQLAlchemy ORM + Alembic Migrations)
- **Storage client**: `boto3` (Tương thích mọi S3-compatible APIs: AWS S3, MinIO, Cloudflare R2, DO Spaces)
- **Event Broker**: `aiokafka` (Transactional Outbox Pattern)
- **Testing**: `pytest` + `pytest-asyncio` + `openapi-core` (Contract testing)

---

## Quick Start

### 1. Thiết lập môi trường

```bash
# Copy cấu hình môi trường mẫu
cp .env.example .env

# Cài đặt dependencies và khởi tạo môi trường ảo qua uv
uv sync
```

### 2. Chạy Service

Service chạy đồng thời FastAPI web server (port `8080`) và gRPC server (port `50051`):

```bash
make run
```

---

## API Endpoints

### REST API (Queries & Presign)

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/health` | Health Check | Public |
| GET | `/api/v1/profile/companions` | Catalogue Search | Public |
| GET | `/api/v1/profile/companions/{id}` | Companion Detail (Magazine View) | Public |
| GET | `/api/v1/profile/me` | Get current companion profile | Bearer |
| POST | `/api/v1/profile/presigned-url` | Get Presigned URL upload metadata-first | Bearer |

### gRPC API (Commands & Internal Queries)

| RPC | Service | Description | Role |
|-----|---------|-------------|------|
| `CreateProfile` | `ProfileService` | Tạo hồ sơ Companion mới | Companion |
| `UpdateProfile` | `ProfileService` | Cập nhật hồ sơ Companion | Companion |
| `ApproveProfile` | `ProfileService` | Phê duyệt hồ sơ Companion | Admin |
| `RejectProfile` | `ProfileService` | Từ chối hồ sơ Companion | Admin |
| `CreateScenario` | `ProfileService` | Tạo kịch bản dịch vụ mới | Companion |
| `UpdateScenario` | `ProfileService` | Cập nhật kịch bản dịch vụ | Companion |
| `DeleteScenario` | `ProfileService` | Xóa kịch bản dịch vụ | Companion |
| `RegisterVoiceIntro` | `ProfileService` | Đăng ký file Voice Intro | Companion |
| `RegisterAlbumImage` | `ProfileService` | Đăng ký file ảnh album | Companion |
| `GetScenarioSnapshot` | `ProfileService` | Lấy snapshot kịch bản (Internal) | Internal |

---

## Testing

Chạy toàn bộ unit test, integration test và contract test:

```bash
make test-all
```
