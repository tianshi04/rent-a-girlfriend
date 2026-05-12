# ADR 0002: Lựa chọn Hệ quản trị CSDL cho Notification Service

**Trạng thái:** Chấp nhận (Accepted)
**Ngày:** 2026-05-10

## Ngữ cảnh (Context)
Notification Service cần lưu trữ hai thực thể chính: `Notification` (Nội dung thông báo) và `DeliveryAttempt` (Lịch sử các lần thử gửi qua SSE, FCM, Email). 
Đặc thù của service này là lượng dữ liệu ghi vào (Write) sẽ rất lớn, nhưng cũng đòi hỏi tính toàn vẹn dữ liệu cao để xử lý logic Retry (đếm số lần thử nghiệm chính xác, tránh gửi trùng lặp).

Chúng ta cần quyết định chọn giữa hệ cơ sở dữ liệu NoSQL (như MongoDB - lưu dạng Document lồng nhau) hay RDBMS (như PostgreSQL - lưu 2 bảng quan hệ 1-N).

## Quyết định (Decision)
Chọn **PostgreSQL** làm cơ sở dữ liệu chính cho Notification Service.

### Lý do (Rationale):
1. **Tính toàn vẹn dữ liệu (Data Integrity)**: Việc cập nhật trạng thái (`Status`, `RetryCount`) của một thông báo cần đảm bảo tính ACID, đặc biệt trong môi trường xử lý bất đồng bộ nhiều worker. RDBMS xử lý lock dòng (row-level lock) tốt hơn cho các case này.
2. **Khả năng truy vấn (Querying & Audit)**: Phân tách `DeliveryAttempt` thành một bảng riêng biệt giúp dễ dàng JOIN và thực hiện các câu query thống kê (Ví dụ: "Hôm nay có bao nhiêu nỗ lực gửi FCM bị lỗi mạng?").
3. **Tính linh hoạt với JSONB**: Mặc dù dùng CSDL quan hệ, PostgreSQL hỗ trợ kiểu dữ liệu `JSONB` rất mạnh mẽ. Ta hoàn toàn có thể lưu cấu trúc `Payload` động của Notification vào cột JSONB mà không lo bị cứng nhắc về Schema.

## Hệ quả (Consequences)

### Tích cực (Positives)
- Cấu trúc dữ liệu minh bạch, ràng buộc khóa ngoại (Foreign Key) chặt chẽ giữa `Notification` và `DeliveryAttempt`.
- Đảm bảo tính nhất quán (Consistency) cao khi cập nhật trạng thái song song.
- Công cụ ORM (như Gorm/Prisma) hỗ trợ rất tốt.

### Đánh đổi / Tiêu cực (Trade-offs / Negatives)
- **Tốc độ ghi (Write Performance)**: Ghi vào 2 bảng riêng biệt có khóa ngoại sẽ chậm hơn đôi chút so với việc append vào một mảng trong MongoDB Document.
- **Dung lượng (Storage)**: RDBMS sẽ tốn dung lượng hơn cho việc lưu trữ các cấu trúc bảng cứng và Index.
- **Vấn đề mở rộng (Scalability)**: Do dữ liệu thông báo sinh ra liên tục (append-heavy), bảng `DeliveryAttempt` sẽ phình to rất nhanh. Chúng ta sẽ phải có kế hoạch **Table Partitioning** (phân mảnh bảng theo ngày/tháng) hoặc thiết lập Job dọn rác (Archiving) sau 30-90 ngày để tránh giảm hiệu năng.
