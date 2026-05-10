# ADR 0001: Mục tiêu và Phạm vi (Scope) của Notification Service

**Trạng thái:** Chấp nhận (Accepted)
**Ngày:** 2026-05-10

## Ngữ cảnh (Context)
Hệ thống cần một service chuyên biệt để xử lý việc gửi thông báo tới người dùng (Client & Companion) trên đa nền tảng (Web, Mobile). Cần xác định rõ ranh giới Bounded Context để tránh việc Notification Service phình to, ôm đồm logic nghiệp vụ (business logic) của các service khác.

## Quyết định (Decision)

### 1. Mục tiêu cốt lõi (Core Goal)
- Đóng vai trò là **Delivery Hub** (Bưu điện): Nhận thông điệp đã được đóng gói sẵn và giao đến đúng user với độ trễ thấp nhất. Đảm bảo luồng nghiệp vụ chính của hệ thống không bị block khi việc gửi thông báo gặp sự cố.

### 2. Định nghĩa Kênh giao tiếp
- **SSE (Server-Sent Events):** Sử dụng cho trạng thái **Online**. Đẩy thông báo theo thời gian thực khi user đang duy trì kết nối (đang mở App hoặc Web tab).
- **FCM (Firebase Cloud Messaging):** Sử dụng cho trạng thái **Offline / Background**. Gửi Push Notification để đánh thức thiết bị khi user không duy trì kết nối SSE.

### 3. Luồng Fallback (Routing Logic)
1. Kiểm tra trạng thái kết nối SSE của `userId`.
2. Nếu có kết nối Active: Đẩy thông báo qua tất cả các kết nối SSE hiện có.
3. Nếu không có kết nối SSE (hoặc gửi SSE thất bại): Chuyển sang gửi Push Notification qua FCM.

### 4. Phạm vi (In-Scope vs Out-Scope)
**In-Scope:**
- Quản lý vòng đời của các kết nối SSE.
- Giao tiếp với API của Firebase để gửi FCM.
- Tích hợp với dịch vụ Email (SMTP/SendGrid/AWS SES) để gửi Email theo yêu cầu.
- Đảm bảo cơ chế Retry (tối đa 3 lần theo [INV-N01]).
- **Lưu trữ thông báo (Persistence):** Lưu thông báo vào Database để hỗ trợ in-app Notification Center (Inbox) cho User và phục vụ việc tra cứu (Audit).

**Out-Scope (Không thực hiện):**
- **Không tự tạo nội dung:** Không query DB để lấy thông tin booking ráp vào template. Service phát ra yêu cầu gửi phải cung cấp sẵn `Title` và `Body`.
- **Không tự quyết định kênh gửi:** Notification Service không quyết định một event cụ thể nào sẽ dùng Email hay Push. Nó chỉ tuân theo cấu hình (Policy) từ service gốc gửi tới (Ví dụ: `requireEmail: true`).
- **Không quản lý FCM Token:** Việc lưu trữ, thêm mới, xóa Device Token thuộc về Identity/Profile Service. Khi cần gửi FCM, service sẽ truy xuất Token từ Identity/Profile.
- **Không lập lịch (Cronjob):** Không tự query xem ai sắp đến giờ hẹn để nhắc. Logic nhắc nhở thuộc về Booking Service.
- **Không Publish Event Trạng thái (MVP):** Mặc dù Event Catalog có định nghĩa các Outbound Events (như `NotificationDeliveryCompleted`), việc viết code để Publish các event này lên Broker tạm thời bị hoãn lại (deferred) cho đến khi có một service khác thực sự cần lắng nghe.
- **Không tích hợp gRPC thực ngay lập tức (MVP Phase 1):** Ở giai đoạn đầu, hệ thống sẽ ưu tiên hoàn thiện luồng **SSE** trước. Việc lấy FCM Token từ Profile/Identity Service qua gRPC sẽ được **Mock data** (giả lập) để tối ưu tốc độ dev. Khi luồng SSE ổn định, mới thay thế Mock bằng gRPC client thật.

## Hệ quả (Consequences)
- **Tích cực:** Giữ cho Notification Service cực kỳ nhẹ, độc lập và dễ scale.
- **Tiêu cực / Ràng buộc:** Các service khác (như Booking) sẽ phải chịu trách nhiệm format nội dung text và tự quản lý các job hẹn giờ.
