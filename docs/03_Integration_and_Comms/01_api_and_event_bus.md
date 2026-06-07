# API & EVENT BUS (CHIẾN LƯỢC GIAO TIẾP LIÊN DỊCH VỤ)

Tài liệu này định nghĩa cách các Microservices giao tiếp nội bộ thông qua API đồng bộ và nền tảng sự kiện bất đồng bộ.

## 1. GIAO TIẾP ĐỒNG BỘ (SYNCHRONOUS API)

Sử dụng **RESTful API** hoặc **gRPC**. Phương pháp này áp dụng cho:
*   **Các hành vi truy vấn (Query):** Dữ liệu trả về là bắt buộc để có thể tiếp tục luồng xử lý và không làm thay đổi trạng thái của bên cấp dữ liệu (ví dụ: lấy hồ sơ người dùng).
*   **Các lệnh (Command) đặc biệt quan trọng:** Chỉ áp dụng cho các hành động yêu cầu tính **Atomicity** (nguyên tử) và **Strong Consistency** (nhất quán tức thì) ngay tại thời điểm bắt đầu luồng nghiệp vụ để đảm bảo trải nghiệm người dùng hoặc an toàn tài chính.

**Trường hợp sử dụng điển hình:**
1.  `Booking Service` gọi `Profile Service` để lấy *Snapshot* Scenario (Query).
2.  `Booking Service` gọi `Finance Service` (qua gRPC) để thực hiện lệnh `FreezeCoin` (Command). Nếu không khóa được tiền, Booking sẽ không được tạo để tránh trạng thái "treo" chờ đợi.

*Lưu ý:* Hạn chế lạm dụng Sync Command cho các bước sau của quy trình (như Payout/Refund) - những bước này nên sử dụng SAGA/Async để tăng khả năng chịu lỗi.

## 2. GIAO TIẾP BẤT ĐỒNG BỘ (ASYNCHRONOUS EVENT BUS)

Sử dụng **Message Broker (RabbitMQ / Kafka)** làm nền tảng truyền tải **Domain Events**. Đây là phương thức giao tiếp chủ đạo của hệ thống.

*   **Mô hình:** Publish/Subscribe (Pub/Sub). Producer phát Event và không cần chờ phản hồi, Consumer tự do subscribe theo nhu cầu.
*   **Trường hợp sử dụng:** Khi `Booking Service` bắn event `BookingCompleted`, nó không cần biết service nào đang nhận. `Finance Service` tự bắt lấy để chia hoa hồng, `Interaction Service` bắt lấy để khóa khung chat, và `Notification Service` bắt để gửi thông báo.

## 3. CƠ CHẾ TRUYỀN NHẬN SỰ KIỆN (PUSH & PULL MECHANICS)

Để tối ưu hóa hiệu năng, độ tin cậy và khả năng chịu lỗi, hệ thống chuẩn hóa cách thức giao tiếp với Message Broker (đặc biệt là Kafka) như sau:

### 3.1. Phía Gửi (Publish / Push) - Transactional Outbox Pattern
Các service thay đổi trạng thái không gửi trực tiếp tin nhắn lên Kafka trong tiến trình xử lý yêu cầu chính. Thay vào đó, hệ thống áp dụng **Transactional Outbox Pattern**:
1.  **Ghi nguyên tử (Atomic Write):** Dữ liệu nghiệp vụ mới và bản ghi sự kiện (lưu trong bảng `outbox` / `outbox_events`) được lưu vào cơ sở dữ liệu trong cùng một database transaction. Điều này đảm bảo nếu lưu nghiệp vụ thành công thì chắc chắn sự kiện được ghi nhận.
2.  **Quét và đẩy (Poll & Push):** Một background task (ví dụ: `OutboxWorker`, `OutboxPublisherWorker`) chạy độc lập quét bảng outbox định kỳ (mặc định mỗi `500ms` với lô `50` phần tử), thực hiện khóa bản ghi (`SELECT FOR UPDATE SKIP LOCKED` để hỗ trợ chạy đa thực thể song song), sau đó **push** (gửi) sự kiện lên Kafka Broker ngoài phạm vi giao dịch DB.
3.  **Xác nhận (Acknowledge):** Sau khi Kafka Broker xác nhận đã nhận tin nhắn thành công, worker cập nhật bản ghi outbox thành `published = true` để giải phóng.

### 3.2. Phía Nhận (Subscribe / Pull) - Continuous Long Polling & Idempotency
Tất cả các consumer đều áp dụng cơ chế chủ động **kéo (Pull)** sự kiện từ Broker theo vòng lặp liên tục kết hợp kiểm soát lỗi:
1.  **Long Polling (Không chờ rỗng):** Thay vì sử dụng khoảng nghỉ ngủ cố định (`sleep`), consumer gọi lệnh kéo tin nhắn liên tục. Lệnh kéo này được cấu hình chế độ Long Polling (ví dụ: `MinBytes: 1` hoặc `timeout_ms: 1000`). Nếu chưa có tin nhắn, kết nối sẽ treo (block) tại Broker cho đến khi có dữ liệu mới hoặc hết thời gian timeout. Điều này giúp hệ thống đạt độ trễ thời gian thực (real-time) và không làm tốn tài nguyên mạng.
2.  **Đọc trước, Commit sau (Manual Offset Commit):** Offset của tin nhắn chỉ được commit lên Kafka sau khi consumer đã xử lý thành công nghiệp vụ. Nếu xử lý lỗi, offset không được commit để Kafka có thể phân phối lại tin nhắn khi consumer khởi động lại (bảo đảm *At-Least-Once Processing*).
3.  **Bộ lọc trùng lặp (Idempotent Consumer):** Vì sự kiện có thể bị gửi lặp lại, consumer sử dụng `event_id` đi kèm trong phong bì CloudEvents để kiểm tra trùng lặp trong DB trước khi xử lý (sử dụng bảng `processed_events` hoặc tương đương). Nếu sự kiện đã được xử lý thành công trước đó, nó sẽ bị bỏ qua lập tức (*Exactly-Once Processing* ngữ nghĩa).

## 4. CHUẨN ĐỊNH DẠNG SỰ KIỆN (CLOUDEVENTS FORMAT)

Để các Bounded Context giao tiếp an toàn, mọi Event đẩy lên Kafka/RabbitMQ phải tuân thủ chuẩn "Envelope" của **CloudEvents**.

**Cấu trúc ví dụ cho `BookingAccepted`:**
```json
{
  "specversion": "1.0",
  "id": "evt_abc123", 
  "source": "/rent-a-gf/booking-context/booking/bk_999",
  "type": "com.rentagf.booking.BookingAccepted.v1",
  "datacontenttype": "application/json",
  "time": "2023-10-27T10:00:00Z",
  "data": {
    "bookingId": "bk_999",
    "sagaId": "saga_555",
    "companionId": "cmp_123",
    "price": 500
  },
  "extensions": {
    "correlationId": "req_xyz789"
  }
}
```

**Chi tiết các trường bắt buộc:**
*   `id`: UUID duy nhất cho từng event. Dùng để làm Idempotency Key ở Consumer.
*   `type`: Có định dạng `[domain].[context].[EventName].[version]`. Versioning (`.v1`) hỗ trợ nâng cấp cấu trúc payload mà không làm gãy các Consumer cũ.
*   `data`: Payload nghiệp vụ thực tế chứa các thông tin cần thiết.
*   `correlationId` (Extensions): Truyền xuyên suốt từ API Gateway đến tất cả các Context để Debug/Trace log trên hệ thống (Kibana/Datadog).
*   `sagaId` (Trong data hoặc extensions): Định danh của phiên giao dịch phân tán nếu event thuộc một phần của luồng SAGA.
