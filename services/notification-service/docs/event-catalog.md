# 📨 DANH MỤC SỰ KIỆN (EVENT CATALOG)

Tài liệu này định nghĩa cấu trúc giao tiếp (Event Contracts) giữa Notification Service và phần còn lại của hệ thống. Tuân thủ kiến trúc Event-Driven, các service giao tiếp thông qua Message Broker và sử dụng chuẩn **CloudEvents v1.0**.

> [!TIP]
> **Triết lý thiết kế (Hybrid Strategy)**: Notification Service kết hợp giữa sự **Chủ động** (Smart Consumer - lắng nghe Domain Events trực tiếp từ các service core) và sự **Thụ động** (lắng nghe sự kiện `NotificationRequested` cho các trường hợp generic). Cách tiếp cận này giúp các Core Service hoàn toàn sạch bóng logic hiển thị thông báo, trong khi Notification Service vẫn duy trì được tính linh hoạt cao nhờ hệ thống [Template YAML](../config/templates.yaml).

---

## 1. INBOUND EVENTS (SỰ KIỆN LẮNG NGHE)

Hệ thống lắng nghe hai nhóm sự kiện:

### 1.1. Nhóm sự kiện Domain (Chủ động)
Notification Service lắng nghe trực tiếp các sự kiện nghiệp vụ để tự tạo thông báo. Chi tiết ánh xạ xem tại: [Domain Event Mapping](./domain-event-mapping.md).

### 1.2. Nhóm sự kiện Yêu cầu (Thụ động)
Dành cho các trường hợp đặc biệt hoặc service bên thứ 3.

#### `rentagf.notification.requested.v1`
- **Mô tả:** Được phát ra bởi các Core Services (Booking, Finance, v.v.) khi họ muốn Notification Service giao một thông báo tới người dùng.
- **Routing Key:** `notification.requested`

**Cấu trúc Payload (JSON):**
```json
{
  "specversion": "1.0",
  "type": "rentagf.notification.requested.v1",
  "source": "/services/booking",
  "id": "e3b0c442-989b-464c-8650-123456789abc",
  "time": "2026-05-10T19:00:00Z",
  "datacontenttype": "application/json",
  "data": {
    "userId": "user-uuid-1234",
    "classificationType": "TRANSACTIONAL",
    "priority": "HIGH",
    "content": {
      "title": "Bạn có 1 yêu cầu đặt lịch mới!",
      "body": "Client Nam đã đặt lịch lúc 20:00 tối nay.",
      "actionUrl": "rentagf://booking/detail/booking-uuid-5678",
      "imageUrl": "https://storage.rentagf.com/images/xyz.png"
    },
    "policyOverrides": {
      "requirePush": true,
      "requireEmail": false
    }
  }
}
```

**Chi tiết trường dữ liệu (Data):**
- `classificationType`, `priority`: Phải khớp với các Enum đã định nghĩa trong file `notification-taxonomy.md`.
- `content.actionUrl` (Tùy chọn): Deep link để app điều hướng khi User bấm vào thông báo.
- `policyOverrides` (Tùy chọn): Dùng để ép buộc Notification Service phá vỡ rule mặc định (VD: Bắt buộc gửi Email cho Event quên mật khẩu).

---

## 2. OUTBOUND EVENTS (SỰ KIỆN PHÁT RA)

Mặc dù có định nghĩa Contract, nhưng cho mục tiêu MVP, hệ thống **TẠM THỜI CHƯA IMPLEMENT** logic Publish các sự kiện này lên Broker (Theo quyết định tại `ADR-0001`), nhằm giữ cho service đơn giản nhất có thể.

Khi hệ thống mở rộng (cần tracking/audit nâng cao), chúng ta sẽ sử dụng các hợp đồng sau:

### `rentagf.notification.delivery.completed.v1`
- **Mô tả:** Phát ra khi thông báo đã được gửi thành công đến người dùng (qua SSE hoặc FCM/Email).

**Cấu trúc Data:**
```json
"data": {
  "notificationId": "uuid-cua-thong-bao",
  "userId": "user-uuid-1234",
  "idempotencyKey": "e3b0c442-...", // Link lại với idempotency key gốc
  "channelUsed": "FCM",
  "completedAt": "2026-05-10T19:00:05Z"
}
```

### `rentagf.notification.delivery.failed.v1`
- **Mô tả:** Phát ra khi hệ thống đã cố gắng Retry hết số lần cho phép (VD: 3 lần) mà vẫn không thể gửi thông báo.

**Cấu trúc Data:**
```json
"data": {
  "notificationId": "uuid-cua-thong-bao",
  "userId": "user-uuid-1234",
  "idempotencyKey": "e3b0c442-...",
  "failureReason": "FCM Token is invalid or expired. Tried 3 times.",
  "failedAt": "2026-05-10T19:05:00Z"
}
```

---

## 3. NGUYÊN TẮC GIAO TIẾP (BEST PRACTICES)

1. **Idempotency (Tính luỹ đẳng)**: Notification Service dựa vào trường `id` ở Header của CloudEvent (được mapping thành `idempotencyKey`) và `userId` của người nhận để xác định xem thông báo này đã được xử lý chưa. Nếu xử lý rồi, nó sẽ bỏ qua để tránh gửi tin rác cho người dùng và cho phép gửi tin đa đối tượng (multi-recipient) an toàn từ cùng một sự kiện gốc.
2. **Fire-and-Forget**: Service nguồn (Booking, Finance) sau khi đẩy `NotificationRequested` lên Queue thì coi như xong việc, không cần đợi phản hồi từ Notification Service. Việc này đảm bảo độ trễ thấp nhất cho luồng nghiệp vụ chính.
