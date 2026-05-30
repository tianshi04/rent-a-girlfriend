# ADR 0008: Chọn Java Domain Model làm Single Source of Truth cho các Enum Trạng thái và Phân tách Enum Kỹ thuật của Protobuf

**Trạng thái:** Chấp nhận (Accepted)
**Ngày:** 2026-05-21

## Ngữ cảnh (Context)
*   Hệ thống cần quản lý chặt chẽ vòng đời của một thông báo (`NotificationStatus`) và trạng thái của từng nỗ lực gửi (`AttemptStatus`) để phục vụ máy trạng thái (State Machine) và cơ chế tự động thử lại (Retry).
*   Có sự khác biệt về mặt ràng buộc công nghệ giữa **Protobuf Contract (API/Event Contract bên ngoài)** và **Java Domain Model & DB Schema (Logic nghiệp vụ bên trong)**:
    *   **Protobuf3** bắt buộc phần tử đầu tiên của `enum` phải có index bằng `0` và thường được đặt tên có hậu tố `_UNSPECIFIED` để làm "vùng đệm an toàn" khi deserialize dữ liệu lỗi hoặc dữ liệu từ phiên bản SDK cũ (nhằm đảm bảo tính tương thích ngược - Backward Compatibility).
    *   Nếu chúng ta đưa trạng thái kỹ thuật `UNSPECIFIED` này vào Java Domain Model hoặc Database, nó sẽ làm ô nhiễm (pollute) Domain nghiệp vụ thuần túy, buộc chúng ta phải viết thêm code để xử lý các kịch bản `if (status == UNSPECIFIED)` vô nghĩa trong nghiệp vụ.

## Quyết định (Decision)

### 1. Chọn Java Domain Model & Database Schema làm Single Source of Truth
Chúng tôi quyết định lấy bộ trạng thái nghiệp vụ đã được định nghĩa và bảo vệ chặt chẽ bởi các quy tắc bất biến (Invariants) trong Java Domain Model làm nguồn chuẩn duy nhất (Single Source of Truth).
*   **NotificationStatus:** `PENDING`, `PROCESSING`, `COMPLETED`, `FAILED`.
*   **AttemptStatus:** `PENDING`, `SUCCESS`, `FAILED_RECOVERABLE`, `FAILED_UNRECOVERABLE`.

### 2. Phân tách Sạch sẽ Domain Enum và Protobuf Enum (Hexagonal Architecture)
Để dung hòa giữa ràng buộc công nghệ của Protobuf và tính thuần khiết của Domain Model, chúng tôi thực hiện phân tách sạch sẽ:
*   **Ở phía ngoài (Protobuf Contract):** Các file `.proto` (như [attempt_status.proto](../../../../contracts/notification/v1/enums/attempt_status.proto)) bắt buộc phải định nghĩa giá trị mặc định `_UNSPECIFIED = 0` làm phần tử đầu tiên để tuân thủ cú pháp Proto3 và đảm bảo tính tương thích mạng.
    *   **QUYẾT ĐỊNH QUAN TRỌNG - TUYỆT ĐỐI KHÔNG ĐƯỢC XÓA:** Kể cả khi tài liệu nghiệp vụ (`docs/*.md`) và Java Domain model không có trạng thái `UNSPECIFIED`, chúng ta **tuyệt đối KHÔNG** được xóa dòng `_UNSPECIFIED = 0` này khỏi các file `.proto`. Việc giữ lại giá trị này là bắt buộc trong Proto3 để làm vùng đệm an toàn khi deserialization. Nếu xóa đi, hệ thống sẽ mất khả năng tương thích ngược (Backward Compatibility), dễ dẫn đến lỗi crash hệ thống khi giao tiếp mạng nếu nhận được thông điệp chứa trạng thái chưa được định nghĩa hoặc từ các phiên bản SDK cũ/mới lệch nhau.
*   **Ở phía trong (Java Domain & Database):** Các Domain Enum (`NotificationStatus.java`, `AttemptStatus.java`) và Database Schema **tuyệt đối KHÔNG** chứa giá trị `UNSPECIFIED` để tránh ô nhiễm domain nghiệp vụ thuần túy.

### 3. Thiết lập lớp Dịch trạng thái (State Translation Layer)
Lớp **Event Translator / Mapper** ở tầng Infrastructure sẽ đóng vai trò là "người gác cổng" (Gatekeeper):
*   Khi tiêu thụ Event từ Message Broker (Kafka) sử dụng Protobuf, Translator sẽ ánh xạ (map) Protobuf enum sang Domain enum sạch.
*   Nếu gặp giá trị `UNSPECIFIED` hoặc giá trị không hợp lệ từ bên ngoài, Translator sẽ chặn lại ngay lập tức tại cửa ngõ Infrastructure (ném lỗi hoặc ghi log lỗi kỹ thuật), tuyệt đối không cho phép dữ liệu rác đi vào phá hỏng tính toàn vẹn của Domain Layer.

---

## Hệ quả (Consequences)

### Tích cực (Tác động tốt):
*   **Domain sạch sẽ:** Giữ cho Domain Model thuần khiết nghiệp vụ, cực kỳ dễ đọc, dễ viết Unit Test độc lập mà không bị ô nhiễm bởi các ràng buộc kỹ thuật của hạ tầng.
*   **Bảo vệ hạ tầng tốt:** Tận dụng tối đa khả năng tương thích ngược (Backward Compatibility) của Protobuf giúp hệ thống phân tán không bị crash khi các service khác chạy lệch phiên bản SDK.
*   **Dễ debug:** Database chỉ chứa các dữ liệu nghiệp vụ hợp lệ và chính xác, giúp việc tra cứu log và audit cực kỳ trực quan.

### Đánh đổi (Ràng buộc):
*   Tốn thêm một lượng nhỏ boilerplate code ở lớp Mapper để dịch chuyển qua lại giữa Protobuf Object và Domain Model. Tuy nhiên, đây là cái giá hoàn toàn xứng đáng cho một thiết kế Hexagonal Architecture chuẩn chỉnh và an toàn.
