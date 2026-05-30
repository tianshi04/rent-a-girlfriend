# ADR 0006: Chiến lược Thiết kế Payload - Lựa chọn giữa Operational Flexibility và Type Safety

**Trạng thái:** Chấp nhận (Accepted)  
**Ngày:** 2026-05-21  

---

## 1. Ngữ cảnh (Context)

Trong **Notification Service**, thực thể `Notification` đóng vai trò là một Delivery Hub phân phối thông báo qua nhiều kênh truyền thông khác nhau (`SSE`, `FCM`, `EMAIL`). Mỗi kênh yêu cầu một cấu trúc dữ liệu payload khác nhau:
- **Email**: Cần `toEmail`, `subject`, `body`, `templateId`, `attachments`,...
- **FCM**: Cần `userId` (hoặc push token), `title`, `body`, `imageUrl`, `deepLink`, `clickAction`,...
- **SSE**: Cần cấu trúc dữ liệu JSON động để Frontend bắt và render trực tiếp trên UI.

Chúng ta cần quyết định cách thiết kế và biểu diễn cấu trúc `payload` của `Notification` trong mã nguồn Java (Domain Layer) và Database (PostgreSQL JSONB).

---

## 2. Các Phương án Thiết kế & Phân tích Trade-offs

### Phương án A: Thiết kế hướng an toàn kiểu dữ liệu (Strong Type Safety)
Sử dụng cấu trúc phân cấp Class (Hierarchy) hoặc kiểu Generic để định nghĩa Payload chặt chẽ cho từng kênh gửi:

```java
// Cách A1: Sử dụng Hierarchy Class
interface NotificationPayload {}
class EmailPayload implements NotificationPayload { String toEmail; String subject; String body; }
class FcmPayload implements NotificationPayload { String token; String title; String body; }

// Cách A2: Sử dụng Generic interface cho Sender
interface NotificationSender<T extends NotificationPayload> {
    boolean send(Notification notification, T payload);
}
```

* **Ưu điểm**:
  - **Type Safety**: Lập trình viên biết chính xác mỗi kênh cần truyền các trường dữ liệu nào, giảm thiểu lỗi runtime do truyền thiếu hoặc sai kiểu dữ liệu.
  - **IDE Autocomplete & Refactoring**: Dễ dàng tìm kiếm các tham chiếu, tự động hoàn thành code và refactor an toàn khi thay đổi trường.
* **Nhược điểm**:
  - **Rigidity (Cứng nhắc)**: Khi có một sự kiện mới yêu cầu bổ sung 1 trường dữ liệu mới (ví dụ: thêm `badgeCount` vào FCM), ta bắt buộc phải thay đổi code Java, cập nhật Class `FcmPayload`, biên dịch và deploy lại toàn bộ Service.
  - **Boilerplate**: Tạo ra số lượng lớp con (Classes) khổng lồ cho mỗi loại event và channel.
  - **Khó khăn trong việc tuần tự hóa (Serialization)**: Gặp nhiều trở ngại khi deserialize từ JSON trong DB lên đúng lớp con cụ thể tương ứng tại runtime.

---

### Phương án B: Thiết kế hướng linh hoạt vận hành (Operational Flexibility) - *LỰA CHỌN*
Sử dụng cấu trúc bản đồ khóa-giá trị động (`Map<String, Object>` trong Java và cột dữ liệu `JSONB` trong PostgreSQL):

```java
public class Notification {
    private Map<String, Object> payload;
    // ...
}
```

* **Ưu điểm**:
  - **Operational Flexibility (Linh hoạt vận hành cực cao)**: Cực kỳ linh hoạt khi hệ thống tiến hóa (evolve) liên tục. Ta có thể thêm, bớt hoặc thay đổi bất kỳ trường dữ liệu nào trong payload mà không cần sửa đổi Schema Database hay tạo thêm class Java mới.
  - **Tối giản mã nguồn (Low Boilerplate)**: Ít Class hơn, cấu trúc Domain tinh gọn, tập trung hoàn toàn vào logic nghiệp vụ cốt lõi và State Machine.
  - **Dễ dàng serialize/deserialize**: PostgreSQL hỗ trợ kiểu `JSONB` cực kỳ mạnh mẽ, ánh xạ trực tiếp sang `Map<String, Object>` thông qua Jackson/Hibernate một cách tự động và không có rủi ro về kiểu lớp cụ thể.
* **Nhược điểm**:
  - **Mất Type Safety**: Compiler không thể phát hiện nếu lập trình viên truyền thiếu hoặc truyền sai tên trường (ví dụ: truyền `"action_url"` thay vì `"actionUrl"`). Việc kiểm soát này phải đẩy lên tầng Application (Template Engine / Validation) tại runtime.

---

## 3. Quyết định (Decision)

Hệ thống quyết định chọn **Phương án B: Sử dụng Map<String, Object> kết hợp PostgreSQL JSONB** làm mô hình biểu diễn dữ liệu của Payload.

Chúng ta chủ ý đánh đổi tính **Type Safety** ở mức biên dịch (Compile-time) để lấy tính **Operational Flexibility** ở mức vận hành (Runtime). Quyết định này dựa trên các lý do thực tiễn:
1. **Bản chất của Notification Service**: Là một hệ thống Generic Subdomain (Delivery Hub). Nhiệm vụ cốt lõi của nó là vận chuyển thông điệp, không phải nắm giữ và xác thực chặt chẽ cấu trúc nghiệp vụ của từng thông điệp. Việc định cấu trúc thông điệp thuộc trách nhiệm của các dịch vụ nguồn (Booking, Finance, Dispute).
2. **Khả năng thay đổi nhanh chóng (High Velocity)**: Trong quá trình phát triển dự án Rent-a-Girlfriend, các yêu cầu hiển thị thông báo trên Mobile/Web sẽ liên tục thay đổi. Việc sử dụng cấu trúc `Map` giúp Notification Service hoàn toàn đứng ngoài các thay đổi này, giảm thiểu tối đa rủi ro lan truyền thay đổi (Change Propagation).

---

## 4. Hệ quả & Cơ chế Bù đắp (Mitigation Strategies)

Để khắc phục nhược điểm mất an toàn kiểu dữ liệu (Type Safety) của Phương án B, hệ thống áp dụng các cơ chế bù đắp sau:

1. **Tài liệu hóa tích hợp chặt chẽ**: Viết tài liệu [event-integration-guide.md](../event-integration-guide.md) chỉ rõ payload yêu cầu cho từng Smart Consumer Event.
2. **Hệ thống Template tập trung**: Nội dung hiển thị thực tế (`title`, `body`) sẽ không được truyền trực tiếp qua Payload mà được quản lý tập trung ở file [templates.yaml](../../config/templates.yaml). Lớp Template Engine sẽ chịu trách nhiệm biên dịch và ném lỗi rõ ràng nếu Payload truyền thiếu các biến template bắt buộc.
3. **Mã lỗi nghiệp vụ cụ thể**: Trả về mã lỗi runtime chi tiết nếu việc xử lý template bị thiếu biến (ví dụ: `ERR_TEMPLATE_INTERPOLATION_FAILED`), giúp developer của các service khác dễ dàng debug khi gửi tin sai payload.
