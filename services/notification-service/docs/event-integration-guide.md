# 📡 HƯỚNG DẪN TÍCH HỢP NOTIFICATION SERVICE (Event Integration Guide)

Tài liệu này dành cho **các team phát triển services khác** (Booking, Finance, Interaction, Profile, Identity, Dispute). Nó define rõ: Notification Service cần gì từ hệ thống để hoạt động.

> [!IMPORTANT]
> **Nguyên tắc cốt lõi:**
> 1. **Events phải self-contained** — Notification KHÔNG gọi ngược source service để lấy thêm data.
> 2. **Idempotency** — Trùng lặp thông báo (dựa trên cặp composite key `idempotency_key` và `user_id`) sẽ bị reject tự động.
> 3. **Notification KHÔNG quyết định** gửi hay không gửi — chỉ thực thi theo policy đính kèm hoặc cấu hình mặc định.

---

## 1. CHIẾN LƯỢC LẮNG NGHE (Subscription Strategy)

Notification Service hoạt động theo mô hình **Hybrid** (ADR-0005):

| Strategy | Mô tả | Source Service cần làm gì |
|:---|:---|:---|
| 🟢 **Smart Consumer** | Notification tự lắng nghe Domain Event, tự dịch thành nội dung thông báo qua Template | Chỉ cần publish event đúng payload — không cần biết đến Notification |
| 🟡 **Passive Subscriber** | Service gửi event `notification.requested.v1` với nội dung đã format sẵn | Tự format `title`, `body` rồi gửi |

> [!TIP]
> **Khi nào dùng Passive Subscriber?**
> - Thông báo mang tính hệ thống/admin không gắn với Domain Event cụ thể
> - Service bên thứ 3 hoặc legacy chưa có Domain Event
> - Trường hợp đặc biệt cần kiểm soát hoàn toàn nội dung
> - Mở rộng nhanh: bất kỳ service nào cũng có thể gửi notification mà không cần Notification Service cập nhật code

---

## 2. BẢNG ĐĂNG KÝ EVENT (Event Subscription Registry)

### 2.1. Booking Context

| Event Type | Strategy | Recipient Field | Required Payload Fields | Template Variables |
|:---|:---|:---|:---|:---|
| `rentagf.booking.requested.v1` | 🟢 Smart | `companion_id` | `booking_id`, `client_id`, `companion_id` | — |
| `rentagf.booking.accepted.v1` | 🟢 Smart | `client_id` | `booking_id`, `client_id`, `companion_id` | `booking_id` |
| `rentagf.booking.rejected.v1` | 🟢 Smart | `client_id` | `booking_id`, `client_id`, `companion_id`, `reason` | `booking_id` |
| `rentagf.booking.cancelled.v1` | 🟢 Smart | (đối phương) | `booking_id`, `client_id`, `companion_id`, `actor_role` | `booking_id`, `actor_role` |

> **Lưu ý `booking.cancelled`**: Recipient là "đối phương" — nếu `actor_role = CLIENT` thì gửi cho `companion_id`, ngược lại gửi cho `client_id`.

### 2.2. Finance Context

| Event Type | Strategy | Recipient Field | Required Payload Fields | Template Variables |
|:---|:---|:---|:---|:---|
| `rentagf.finance.topup.completed.v1` | 🟢 Smart | `user_id` | `user_id`, `amount`, `transaction_id` | `amount` |
| `rentagf.finance.payout.processed.v1` | 🟢 Smart | `companion_id` | `booking_id`, `companion_id`, `payout_amount` | `booking_id`, `payout_amount` |

### 2.3. Interaction Context

| Event Type | Strategy | Recipient Field | Required Payload Fields | Template Variables |
|:---|:---|:---|:---|:---|
| `rentagf.chat.message.sent.v1` | 🟢 Smart | `recipient_id` | `room_id`, `sender_id`, `recipient_id`, `sender_name`, `snippet` | `sender_name`, `snippet` |
| `rentagf.interaction.review.submitted.v1` | 🟢 Smart | `companion_id` | `review_id`, `booking_id`, `client_id`, `companion_id`, `rating` | `rating` |

### 2.4. Profile Context

| Event Type | Strategy | Recipient Field | Required Payload Fields | Template Variables |
|:---|:---|:---|:---|:---|
| `rentagf.profile.companion.approved.v1` | 🟢 Smart | `companion_id` | `companion_id` | — |
| `rentagf.profile.companion.rejected.v1` | 🟢 Smart | `companion_id` | `companion_id`, `reason` | `reason` |
| `rentagf.profile.scenario.rejected.v1` | 🟢 Smart | `companion_id` | `companion_id`, `scenario_id`, `reason` | `reason` |
| `rentagf.profile.voice_intro.rejected.v1` | 🟢 Smart | `companion_id` | `companion_id`, `reason` | `reason` |
| `rentagf.profile.album_image.rejected.v1` | 🟢 Smart | `companion_id` | `companion_id`, `reason` | `reason` |

### 2.5. Identity Context

| Event Type | Strategy | Recipient Field | Required Payload Fields | Template Variables |
|:---|:---|:---|:---|:---|
| `rentagf.identity.upgrade.approved.v1` | 🟢 Smart | `user_id` | `user_id` | — |
| `rentagf.identity.upgrade.rejected.v1` | 🟢 Smart | `user_id` | `user_id`, `reason` | `reason` |
| `rentagf.identity.account.locked.v1` | 🟢 Smart | `user_id` | `user_id`, `reason` | `reason` |

### 2.6. Dispute Context

| Event Type | Strategy | Recipient Field | Required Payload Fields | Template Variables |
|:---|:---|:---|:---|:---|
| `rentagf.dispute.report.created.v1` | 🟢 Smart | `accused_id` | `dispute_id`, `booking_id`, `reporter_id`, `accused_id`, `reason` | `reason` |
| `rentagf.dispute.resolved.v1` | 🟢 Smart | cả `client_id` và `companion_id` | `dispute_id`, `booking_id`, `client_id`, `companion_id`, `resolution` | `resolution` |

### 2.7. Generic (Passive Channel)

| Event Type | Strategy | Mô tả |
|:---|:---|:---|
| `rentagf.notification.requested.v1` | 🟡 Passive | Nhận payload đã format sẵn, deliver trực tiếp |

---

## 3. PASSIVE CHANNEL CONTRACT

Khi một service muốn gửi notification mà **không** thông qua Smart Consumer, sử dụng event sau:

**Event Type**: `rentagf.notification.requested.v1`

**CloudEvents Envelope:**
```json
{
  "specversion": "1.0",
  "type": "rentagf.notification.requested.v1",
  "source": "/services/<tên-service>",
  "id": "<uuid>",
  "time": "2026-05-19T20:00:00Z",
  "datacontenttype": "application/json",
  "data": {
    "user_id": "user-uuid",
    "classification_type": "TRANSACTIONAL",
    "priority": "HIGH",
    "content": {
      "title": "Tiêu đề thông báo",
      "body": "Nội dung thông báo",
      "action_url": "rentagf://deep-link/path",
      "image_url": "https://storage.rentagf.com/img.png"
    },
    "policy_overrides": {
      "require_push": true,
      "require_email": false
    }
  }
}
```

**Chi tiết trường dữ liệu:**

| Trường | Kiểu | Bắt buộc | Mô tả |
|:---|:---|:---|:---|
| `user_id` | string (UUID) | ✅ | Người nhận thông báo |
| `classification_type` | enum | ✅ | `TRANSACTIONAL`, `INTERACTION`, `PROMOTIONAL` |
| `priority` | enum | ✅ | `HIGH`, `MEDIUM`, `LOW` |
| `content.title` | string | ✅ | Tiêu đề hiển thị |
| `content.body` | string | ✅ | Nội dung hiển thị |
| `content.action_url` | string | ❌ | Deep link khi user bấm vào |
| `content.image_url` | string | ❌ | Ảnh đính kèm |
| `policy_overrides.require_push` | bool | ❌ | Ép buộc gửi Push bất chấp user Online |
| `policy_overrides.require_email` | bool | ❌ | Ép buộc gửi Email song song |

---

## 4. EVENTS KHÔNG LẮNG NGHE (Ignore — Hiện tại)

| Event | Lý do |
|:---|:---|
| `finance.coin.frozen/escrowed.v1` | SAGA nội bộ, user không cần biết |
| `chat.room.created.v1` | User đã biết từ booking.accepted |
| `profile.created/updated.v1` | Thao tác của chính user |
| `scenario.created/updated/deleted.v1` | Thao tác nội bộ companion |
| `voice/album.uploaded.v1` | Chỉ upload, chưa có kết quả duyệt |
| `identity.violation.recorded.v1` | Nội bộ, account.locked đã cover |

> [!TIP]
> **Mở rộng trong tương lai:** Bất kỳ event nào ở trên đều có thể "bật" thành Smart Consumer bằng cách thêm config vào `templates.yaml` + subscribe topic mới. Hoặc source service dùng kênh **Passive Subscriber** để gửi notification bất cứ lúc nào mà không cần Notification Service cập nhật code.

---

## 5. CHÍNH SÁCH PHÂN PHỐI MẶC ĐỊNH (Default Delivery Policy)

Khi source service **KHÔNG** truyền `policy_overrides`, Notification sẽ áp dụng chính sách mặc định:

| Classification | Priority | SSE | FCM Push | Retry | Lưu DB |
|:---|:---|:---|:---|:---|:---|
| **TRANSACTIONAL** | HIGH | Bắt buộc | Bắt buộc (nếu Offline) | 3 lần | Có |
| **INTERACTION** | MEDIUM | Bắt buộc | Có (gộp batch) | 1 lần | Không |
| **PROMOTIONAL** | LOW | Không cần | Không | Không | Có (Inbox) |

---

## 6. CHECKLIST CHO TEAM TÍCH HỢP

Khi service của bạn muốn trigger notification, hãy kiểm tra:

- [ ] Event payload có chứa **đủ** required fields theo bảng ở mục 2 không?
- [ ] Recipient field (`user_id`, `client_id`, `companion_id`...) có đúng UUID không?
- [ ] Event đã được publish lên đúng topic trên Kafka không?
- [ ] `id` (CloudEvents `id` field, được map làm `idempotency_key`) có unique không? (Dùng UUID)
- [ ] Nếu dùng Passive Channel: `classification_type` và `priority` có đúng enum không?
