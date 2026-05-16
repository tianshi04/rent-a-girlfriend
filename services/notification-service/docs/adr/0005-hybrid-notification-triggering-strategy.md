# ADR 0005: Chiến lược Kích hoạt Thông báo Hybrid (Hybrid Notification Triggering Strategy)

**Trạng thái:** Đề xuất (Proposed)
**Ngày:** 2026-05-12

## Ngữ cảnh (Context)

Theo thiết kế ban đầu tại [ADR 0001](./0001-notification-service-scope-and-goals.md), Notification Service đóng vai trò là một "Bưu điện" thụ động (Passive Delivery Hub). Toàn bộ nội dung thông báo (`Title`, `Body`) phải được các Core Services (Booking, Finance, v.v.) đóng gói sẵn và gửi qua sự kiện `NotificationRequested`.

Tuy nhiên, cách tiếp cận này bộc lộ một số hạn chế:
1.  **Core Services bị "nhiễm" logic hiển thị**: Việc định dạng văn bản thông báo (ví dụ: "Bạn có lịch hẹn mới...") nằm trong mã nguồn của Booking Service làm giảm tính thuần túy của Domain Logic.
2.  **Khó quản lý trải nghiệm người dùng (UX)**: Khi muốn thay đổi văn phong hoặc hỗ trợ đa ngôn ngữ cho thông báo, lập trình viên phải sửa code ở nhiều service khác nhau.
3.  **Tăng Coupling về tri thức**: Core Service phải "biết" cách định dạng dữ liệu cho Notification Service.

## Quyết định (Decision)

Hệ thống sẽ chuyển sang mô hình **Hybrid (Lai)** để kích hoạt việc gửi thông báo, kết hợp giữa sự chủ động và thụ động:

### 1. Cơ chế Chủ động (Smart Consumer - Ưu tiên)
Notification Service sẽ trực tiếp lắng nghe các **Domain Events** quan trọng từ Message Broker (ví dụ: `rentagf.booking.created.v1`, `rentagf.payment.failed.v1`).
- **Event Translator Layer**: Bên trong Notification Service sẽ có một lớp chuyên trách việc "dịch" các Domain Events này. 
- Lớp này sẽ trích xuất dữ liệu cần thiết từ Event Payload và tự động tạo ra nội dung thông báo (`Title`, `Body`) dựa trên các Template đã định nghĩa.
- Điều này giúp các Core Services hoàn toàn không cần biết đến sự tồn tại của hệ thống thông báo (**Loose Coupling**).

### 2. Cơ chế Thụ động (Passive Subscriber - Duy trì)
Vẫn giữ lại sự kiện `NotificationRequested` (đã định nghĩa tại [Event Catalog](../docs/event-catalog.md)) cho các trường hợp:
- Các thông báo mang tính chất hệ thống/admin không gắn liền với một Domain Event cụ thể.
- Các service bên thứ ba hoặc các thành phần legacy chưa chuyển đổi sang kiến trúc Domain Event.
- Các trường hợp đặc biệt yêu cầu Core Service kiểm soát hoàn toàn nội dung tin nhắn.

## Lý do (Rationale)

- **Loose Coupling (Cấu trúc)**: Đúng như nguyên lý của Event-Driven Architecture, Core Services chỉ tập trung vào việc thực hiện nghiệp vụ và thông báo cho thế giới biết "tôi đã làm xong việc X". 
- **Tập trung hóa UX**: Việc quản lý nội dung, template và chiến lược gửi tin (SSE vs FCM) được quy về một mối, giúp dễ dàng bảo trì và nhất quán thương hiệu.
- **Khả năng mở rộng**: Khi thêm một loại thông báo mới cho cùng một Domain Event, chúng ta chỉ cần cập nhật Notification Service mà không làm phiền đến đội phát triển Core Service.

## Hệ quả (Consequences)

### Tích cực:
- Core Services sạch hơn, mã nguồn tập trung 100% vào Business Logic.
- Dễ dàng triển khai đa ngôn ngữ (Localization) cho thông báo tại một nơi duy nhất.
- Giảm thiểu việc sửa đổi code ở nhiều nơi khi có thay đổi về yêu cầu hiển thị.

### Thách thức / Ràng buộc:
- **Knowledge Coupling**: Notification Service bây giờ cần phải "hiểu" schema của Domain Events từ các service khác.
- **Contract Management**: Cần có cơ chế quản lý Schema (ví dụ: Schema Registry) hoặc viết Integration Tests chặt chẽ để đảm bảo khi Core Service thay đổi cấu trúc Event, Notification Service không bị "gãy" (Broken).
- **Smart Consumer Complexity**: Notification Service sẽ phức tạp hơn một chút vì phải chứa thêm lớp Translator cho từng loại Event.

---
*Ghi chú: Quyết định này sẽ dẫn đến việc cập nhật lại [Event Catalog](../docs/event-catalog.md) và lớp [Interfaces Layer](../README.md#🏗️-cấu-trúc-kiến-trúc-high-level-architecture) của dự án.*
