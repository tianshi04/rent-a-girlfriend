# ADR 0001: Sử dụng biến môi trường DATABASE_URL duy nhất cho kết nối PostgreSQL

**Trạng thái:** Accepted

**Ngày:** 2026-05-25

## Ngữ cảnh (Context)

Trước đây, `interaction-service` (viết bằng Rust) sử dụng nhiều biến môi trường riêng biệt để định cấu hình kết nối tới cơ sở dữ liệu PostgreSQL bao gồm: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, và `DB_SSLMODE`. Trong mã nguồn (`src/main.rs`), các biến này được đọc thủ công và ghép lại thành một chuỗi URL kết nối dạng:

```rust
postgres://{user}:{password}@{host}:{port}/{dbname}?sslmode={sslmode}
```

Việc duy trì quá nhiều biến môi trường riêng lẻ cho cùng một mục đích kết nối cơ sở dữ liệu gây ra một số nhược điểm:
1. Tăng độ phức tạp của tệp cấu hình `.env.example` và các tệp cấu hình triển khai (như Docker Compose, Kubernetes manifests, hoặc các hệ thống CI/CD).
2. Không đồng nhất với các tiêu chuẩn thiết kế hiện đại (ví dụ: mô hình Twelve-Factor App khuyến nghị cấu hình kết nối dịch vụ bên thứ ba thông qua một URI duy nhất).
3. Thiếu linh hoạt khi thay đổi driver hoặc định dạng kết nối (như kết nối thông qua connection pooling, socket, hoặc các tùy chọn bổ sung đặc thù).

## Quyết định (Decision)

Chúng tôi quyết định loại bỏ các biến môi trường cấu hình cơ sở dữ liệu riêng lẻ (`DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`) và thay thế bằng một biến môi trường duy nhất là **`DATABASE_URL`**.

Trong mã nguồn Rust, kết nối cơ sở dữ liệu sẽ đọc trực tiếp từ `DATABASE_URL` với cơ chế fallback mặc định phục vụ cho môi trường phát triển cục bộ:

```rust
let db_url = std::env::var("DATABASE_URL").unwrap_or_else(|_| {
    "postgres://postgres:postgres@localhost:5432/interaction_service?sslmode=disable".to_string()
});
```

Đồng thời, cập nhật `.env.example` và `README.md` để hướng dẫn nhà phát triển sử dụng biến cấu hình mới này.

## Hệ quả (Consequences)

### Điểm tích cực (Positives)
- **Đơn giản hóa cấu hình:** Giảm số lượng biến môi trường từ 6 xuống còn 1. Giúp tệp cấu hình `.env` gọn gàng hơn.
- **Tiêu chuẩn hóa:** Phù hợp với chuẩn cấu hình của SQLx và các nền tảng Cloud/Deployment hiện đại (như Heroku, AWS ECS/EKS, Neon, Render, Supabase) vốn cung cấp sẵn chuỗi kết nối PostgreSQL thông qua biến `DATABASE_URL`.
- **Dễ bảo trì:** Mã nguồn khởi tạo Connection Pool trong `main.rs` ngắn gọn hơn, không cần thực hiện định dạng chuỗi thủ công.

### Đánh đổi (Negatives)
- Các tài liệu cũ hoặc các cấu hình môi trường đang chạy cục bộ (nếu có) sẽ cần cập nhật tệp `.env` để sử dụng `DATABASE_URL` thay cho các biến cũ.
