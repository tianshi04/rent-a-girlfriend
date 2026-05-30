# 🗺️ ÁNH XẠ SỰ KIỆN DOMAIN (DOMAIN EVENT MAPPING)

Tài liệu này định nghĩa cách **Notification Service** (với vai trò Smart Consumer) xử lý các sự kiện từ các service khác để tạo ra thông báo cho người dùng.

> [!NOTE]
> Đây là việc thực hiện hóa chiến lược Hybrid quy định tại [ADR-0005](../adr/0005-hybrid-notification-triggering-strategy.md).

---

## 1. LỚP DỊCH SỰ KIỆN (EVENT TRANSLATOR LAYER)

Mỗi sự kiện dưới đây khi được nhận từ Message Broker sẽ được chuyển qua một bộ lọc (Translator) để xác định:
- **Người nhận (RecipientId)**: Lấy từ Payload của sự kiện.
- **Loại thông báo (Type)**: TRANSACTIONAL, INTERACTION, hoặc PROMOTIONAL.
- **Nội dung (Content)**: Tự động tạo dựa trên Template.

---

## 2. DANH SÁCH MAPPING CHI TIẾT

### 2.1. Booking Context
| Sự kiện Domain | Loại | Người nhận | Tiêu đề (Title) | Nội dung (Body Template) |
| :--- | :--- | :--- | :--- | :--- |
| `com.rentagf.booking.BookingRequested.v1` | TRANSACTIONAL | `companionId` | Yêu cầu mới! 🔔 | Bạn có một yêu cầu đặt lịch mới từ khách hàng. |
| `com.rentagf.booking.BookingAccepted.v1` | TRANSACTIONAL | `clientId` | Yêu cầu được chấp nhận ✅ | Companion đã chấp nhận yêu cầu #{{bookingId}} của bạn. |
| `com.rentagf.booking.BookingRejected.v1` | TRANSACTIONAL | `clientId` | Yêu cầu bị từ chối ❌ | Rất tiếc, yêu cầu đặt lịch #{{bookingId}} đã bị từ chối. Lý do: {{reason}}. |
| `com.rentagf.booking.BookingCancelled.v1` | TRANSACTIONAL | `_dynamic` (đối phương) | Lịch hẹn bị hủy ⚠️ | Lịch hẹn #{{bookingId}} đã bị hủy bởi {{actorRole}}. |
| `com.rentagf.booking.BookingAcceptSagaFailed.v1` | TRANSACTIONAL | `clientId` | Lỗi chấp nhận yêu cầu ❌ | Rất tiếc, hệ thống gặp sự cố khi chấp nhận yêu cầu #{{bookingId}}. Chúng tôi đã hoàn trả lại Kano-Coin của bạn. |

### 2.2. Finance Context
| Sự kiện Domain | Loại | Người nhận | Tiêu đề (Title) | Nội dung (Body Template) |
| :--- | :--- | :--- | :--- | :--- |
| `com.rentagf.finance.KanoCoinDeposited.v1` | TRANSACTIONAL | `userId` | Nạp tiền thành công 💰 | Bạn vừa nạp thành công {{amount}} Kano-Coin. |
| `com.rentagf.finance.DepositFailed.v1` | TRANSACTIONAL | `userId` | Nạp tiền thất bại ❌ | Giao dịch nạp {{amount}} Kano-Coin của bạn đã thất bại. Lý do: {{reason}}. |
| `com.rentagf.finance.PayoutProcessed.v1` | TRANSACTIONAL | `companionId` | Thu nhập mới 💸 | Bạn đã nhận được thanh toán {{payoutAmount}} Kano-Coin cho booking #{{bookingId}}. |

### 2.3. Interaction Context
| Sự kiện Domain | Loại | Người nhận | Tiêu đề (Title) | Nội dung (Body Template) |
| :--- | :--- | :--- | :--- | :--- |
| `com.rentagf.interaction.ChatMessageSent.v1` | INTERACTION | `recipientId` | Tin nhắn từ {{senderName}} | {{snippet}} |
| `com.rentagf.interaction.ReviewSubmitted.v1` | INTERACTION | `companionId` | Đánh giá mới! ⭐ | Bạn vừa nhận được đánh giá {{rating}}⭐ từ khách hàng. |

### 2.4. Profile Context
| Sự kiện Domain | Loại | Người nhận | Tiêu đề (Title) | Nội dung (Body Template) |
| :--- | :--- | :--- | :--- | :--- |
| `com.rentagf.profile.ProfileApproved.v1` | PROMOTIONAL | `companionId` | Chào mừng Companion! 🎉 | Hồ sơ của bạn đã được duyệt. Hãy bắt đầu nhận lịch ngay! |
| `com.rentagf.profile.ProfileRejected.v1` | TRANSACTIONAL | `companionId` | Hồ sơ bị từ chối 😔 | Rất tiếc, hồ sơ Companion của bạn chưa được duyệt. Lý do: {{reason}}. |
| `com.rentagf.profile.ScenarioRejected.v1` | TRANSACTIONAL | `companionId` | Kịch bản bị từ chối ❌ | Kịch bản của bạn đã bị từ chối. Lý do: {{reason}}. |
| `com.rentagf.profile.VoiceIntroRejected.v1` | TRANSACTIONAL | `companionId` | Voice Intro bị từ chối 🎤 | Giới thiệu giọng nói của bạn đã bị từ chối. Lý do: {{reason}}. |
| `com.rentagf.profile.AlbumImageRejected.v1` | TRANSACTIONAL | `companionId` | Ảnh bị từ chối 📷 | Ảnh album của bạn đã bị từ chối. Lý do: {{reason}}. |

### 2.5. Identity Context
| Sự kiện Domain | Loại | Người nhận | Tiêu đề (Title) | Nội dung (Body Template) |
| :--- | :--- | :--- | :--- | :--- |
| `com.rentagf.identity.UpgradeApproved.v1` | TRANSACTIONAL | `userId` | Nâng cấp thành công! 🚀 | Bạn đã được nâng cấp thành Companion. Hãy hoàn thiện hồ sơ ngay! |
| `com.rentagf.identity.UpgradeRejected.v1` | TRANSACTIONAL | `userId` | Yêu cầu nâng cấp bị từ chối 😔 | Rất tiếc, yêu cầu nâng cấp Companion đã bị từ chối. Lý do: {{reason}}. |
| `com.rentagf.identity.AccountLocked.v1` | TRANSACTIONAL | `userId` | Tài khoản bị khóa 🔒 | Tài khoản của bạn đã bị khóa. Lý do: {{reason}}. |

### 2.6. Dispute Context
| Sự kiện Domain | Loại | Người nhận | Tiêu đề (Title) | Nội dung (Body Template) |
| :--- | :--- | :--- | :--- | :--- |
| `com.rentagf.dispute.ReportCreated.v1` | TRANSACTIONAL | `accusedId` | Bạn bị báo cáo ⚠️ | Bạn đã bị báo cáo vi phạm. Lý do: {{reason}}. |
| `com.rentagf.dispute.DisputeResolved.v1` | TRANSACTIONAL | `_dynamic` (cả hai bên) | Kết quả khiếu nại 📋 | Khiếu nại cho booking đã được giải quyết: {{resolution}}. |

---

## 3. QUẢN LÝ TEMPLATE QUA YAML (TEMPLATE MANAGEMENT)

Để đảm bảo tính linh hoạt, toàn bộ nội dung thông báo được tách rời khỏi mã nguồn và quản lý tập trung tại file:
`[config/templates.yaml](../config/templates.yaml)`

### Cấu trúc file YAML:
- **`events`**: Chứa danh sách các Domain Event Type theo chuẩn PascalCase và tiền tố `com.rentagf.`.
- **`recipient_field`**: Chỉ định trường nào trong Event Payload chứa ID người nhận theo chuẩn camelCase (giúp hệ thống tự động trích xuất).
- **`channels`**: Danh sách các kênh ưu tiên gửi.
- **`template`**: Chứa nội dung đa ngôn ngữ (`vi`, `en`). Hỗ trợ Placeholder dạng `{{fieldName}}` (camelCase).

---

## 4. LOGIC XỬ LÝ (PROCESSING LOGIC)

1.  **Giải mã (Consume)**: Nhận CloudEvent từ Broker.
2.  **Tra cứu (Lookup)**: Tìm cấu hình trong `templates.yaml` dựa trên trường `type` của sự kiện.
3.  **Trích xuất Recipient**: Lấy ID người nhận từ payload dựa trên cấu hình `recipient_field`.
4.  **Ráp nội dung (Interpolate)**: 
    - Thay thế các Placeholder `{{...}}` bằng dữ liệu thực tế từ payload.
    - Chọn ngôn ngữ dựa trên cấu hình của User (Mặc định là `vi`).
5.  **Lưu trữ (Persistence)**: 
    - Tạo một bản ghi `Notification` trong Database với trạng thái `PENDING`. 
    - Việc này đảm bảo User có thể thấy thông báo trong phần "Inbox" kể cả khi việc gửi Realtime thất bại.
6.  **Định tuyến & Gửi (Route & Deliver)**: 
    - Kiểm tra trạng thái kết nối SSE. Nếu Online, đẩy tin qua SSE.
    - Nếu Offline hoặc SSE thất bại, chuyển sang gửi Push Notification qua FCM.
7.  **Cập nhật (Update)**: Đánh dấu trạng thái `COMPLETED` hoặc `FAILED` vào Database để phục vụ việc Audit và Retry.

---

## 5. QUẢN LÝ THAY ĐỔI (SCHEMA CHANGES)

- Nếu một service core thay đổi Payload (ví dụ: thêm trường `amount`), chúng ta chỉ cần cập nhật nội dung Placeholder trong file YAML và logic trích xuất ở lớp Translator.
- Khuyến nghị: Sử dụng **Schema Registry** để phát hiện sớm các thay đổi gây gãy (Breaking Changes).
