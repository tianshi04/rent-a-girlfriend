# 🎯 DESIGN PATTERN RECORD

Tài liệu này ghi chép lại các mẫu thiết kế (Design Patterns) được áp dụng trong **Notification Service** nhằm giải quyết các bài toán cụ thể về kiến trúc, tính linh hoạt và khả năng mở rộng của hệ thống.

---

## 🛠️ Behavioral Design Patterns

### 1. Strategy Pattern (Mẫu Chiến lược)

* **Tập tin liên quan**: 
  - [NotificationSender.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/application/port/outbound/NotificationSender.java) (Strategy Interface)
  - [EmailPort.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/application/port/outbound/EmailPort.java) (Concrete Strategy)
  - [FcmPort.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/application/port/outbound/FcmPort.java) (Concrete Strategy)
  - [SsePort.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/application/port/outbound/SsePort.java) (Concrete Strategy)

#### 📝 Tại sao sử dụng?
Hệ thống Notification Service hỗ trợ gửi tin qua nhiều kênh truyền thông (Channels) khác nhau như Server-Sent Events (SSE), Firebase Cloud Messaging (FCM), và Email (SMTP). Mỗi kênh có cách thức kết nối, định dạng dữ liệu, thư viện ngoài và cơ chế xác thực riêng biệt.
Nếu viết chung toàn bộ logic gửi tin này vào một UseCase, mã nguồn sẽ bị phình to với các câu lệnh điều kiện `if-else` hoặc `switch-case`, dẫn đến vi phạm nguyên tắc Single Responsibility Principle (SRP) và Open/Closed Principle (OCP).

#### 💡 Cách giải quyết và Cơ chế hoạt động
* Thiết lập một interface chung duy nhất `NotificationSender` ở tầng Application Outbound Port định nghĩa phương thức gửi tin `SendResult send(Notification)` và xác định kênh hỗ trợ `DeliveryChannel getChannel()`.
* Cho 3 Outbound Ports chuyên biệt (`EmailPort`, `FcmPort`, `SsePort`) kế thừa từ interface này. Các adapter ở tầng Infrastructure sẽ tự do triển khai thuật toán gửi tin cụ thể của từng kênh.
* Sử dụng `SendResult` để chuẩn hóa kết quả gửi tin (thành công/thất bại, mã lỗi, và tính khôi phục của lỗi `recoverable`) giúp Domain tự đưa ra quyết định chuyển trạng thái nghiệp vụ mà không cần phụ thuộc vào lỗi kỹ thuật cụ thể của từng kênh hạ tầng.

#### 📈 Ưu điểm đạt được
* **Open/Closed Principle (OCP)**: Khi hệ thống cần hỗ trợ thêm kênh gửi tin mới (ví dụ: SMS, Telegram, Slack), nhà phát triển chỉ cần tạo thêm một Strategy mới mà không cần chỉnh sửa bất kỳ dòng mã nào trong UseCase gửi tin cốt lõi.
* **Tách biệt Trách nhiệm**: Các thay đổi về cấu hình SMTP hay Token FCM hoàn toàn bị cô lập trong adapter tương ứng, không gây rủi ro lan truyền lỗi sang các kênh khác hay tầng Domain nghiệp vụ.

---

## 🏗️ Structural & Behavioral Design Patterns

### 1. Registry Pattern (Mẫu Đăng ký Tập trung)

* **Tập tin liên quan**:
  - [NotificationSenderRegistry.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/application/registry/NotificationSenderRegistry.java) (Registry)

#### 📝 Tại sao sử dụng?
Sau khi định nghĩa các Strategy ở trên, UseCase cần có một cơ chế để lựa chọn động Strategy gửi tin phù hợp dựa trên tham số `DeliveryChannel` được yêu cầu tại thời điểm runtime (ví dụ: gửi mail nếu user offline, gửi SSE nếu user online).
Việc hardcode hoặc trực tiếp autowire từng Strategy Bean vào UseCase sẽ làm tăng độ kết dính (tight coupling) và giảm tính linh hoạt khi mở rộng.

#### 💡 Cách giải quyết và Cơ chế hoạt động
* Lớp `NotificationSenderRegistry` đóng vai trò là một Registry quản lý tập trung toàn bộ các chiến lược gửi tin.
* Sử dụng cơ chế Dependency Injection của Spring Framework để tự động thu thập tất cả các Bean kế thừa từ `NotificationSender` thông qua constructor: `public NotificationSenderRegistry(List<NotificationSender> senderList)`.
* Ánh xạ các Strategy này vào một Map có khóa là `DeliveryChannel` tại thời điểm khởi tạo ứng dụng: `senders = senderList.stream().collect(Collectors.toMap(NotificationSender::getChannel, Function.identity()))`.
* Cung cấp phương thức truy xuất động cực kỳ an toàn: `Optional<NotificationSender> getSender(DeliveryChannel channel)`.

#### 📈 Ưu điểm đạt được
* **Loose Coupling (Kết nối lỏng)**: UseCase không cần biết có bao nhiêu Strategy gửi tin đang tồn tại hay cách chúng được khởi tạo. Nó chỉ giao tiếp trực tiếp với Registry và nhận về Strategy mong muốn.
* **Auto-Discovery**: Khi một Strategy mới được khai báo là `@Component` hoặc `@Bean`, Spring sẽ tự động tiêm nó vào Registry mà không cần thay đổi bất kỳ dòng cấu hình thủ công nào.
