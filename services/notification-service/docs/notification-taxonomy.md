# 🏷️ PHÂN LOẠI VÀ CHÍNH SÁCH THÔNG BÁO (NOTIFICATION TAXONOMY)

Tài liệu này định nghĩa cách hệ thống phân loại các luồng thông báo và áp dụng chính sách phân phối (Delivery Policy) tương ứng cho từng loại. Việc phân loại chuẩn xác giúp Notification Service tối ưu hóa tài nguyên mạng, quản lý độ trễ (latency) và đảm bảo trải nghiệm người dùng (UX).

---

## 1. PHÂN LOẠI THÔNG BÁO (CLASSIFICATION)

Thông báo trong hệ thống Rent-a-Girlfriend được chia thành 3 nhóm chính, dựa trên bản chất nghiệp vụ và mức độ cấp thiết của thông tin.

### 1.1. Transactional/System (Giao dịch/Hệ thống)
Là các thông báo liên quan trực tiếp đến luồng hoạt động chính (Core Business Flows) hoặc tính bảo mật.
- **Ví dụ:** Mã OTP, Kết quả thanh toán (Kano-Coin/VNPay), Yêu cầu/Chấp nhận Booking, Hủy Booking, Thông báo Dispute.

### 1.2. Interaction (Tương tác)
Là các thông báo sinh ra từ hành vi tương tác trực tiếp giữa các người dùng với nhau.
- **Ví dụ:** Tin nhắn Chat mới, Nhận được Đánh giá/Review mới.

### 1.3. Promotional/Alerts (Khuyến mãi/Cảnh báo)
Là các thông báo từ hệ thống nhằm mục đích thông tin chung hoặc marketing.
- **Ví dụ:** Hệ thống chuẩn bị bảo trì, Tặng Voucher khuyến mãi, Nhắc nhở cập nhật Profile.

---

## 2. CHÍNH SÁCH PHÂN PHỐI (DELIVERY POLICY)

Dựa trên phân loại ở trên, mỗi loại thông báo sẽ tuân thủ nghiêm ngặt các chính sách chuyển phát sau:

| Loại thông báo | Độ ưu tiên (Priority) | Realtime (SSE) | FCM Push | Thử lại (Retry) | Lưu DB (Persistence) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Transactional** | `HIGH` | Bắt buộc | Bắt buộc (nếu Offline) | Có (Tối đa 3 lần) | Có (Inbox & Audit) |
| **Interaction** | `MEDIUM` | Bắt buộc | Có (nhưng gộp/Batch) | Có (1 lần) | Không (Lưu ở Interaction Service) |
| **Promotional** | `LOW` | Không cần | Không (Trừ khi Admin ép) | Không | Có (Chỉ lưu Inbox) |

### Chi tiết các thuộc tính chính sách:

- **Độ ưu tiên (Priority)**: Xác định thứ tự lấy message ra khỏi Message Queue để xử lý.
- **Realtime (SSE)**: Xác định xem có bắt buộc phải đẩy thông báo ngay lập tức qua kết nối SSE đang mở hay không.
- **FCM Push**: Quy định việc gọi API Firebase để "đánh thức" thiết bị di động.
  - *Lưu ý*: Với nhóm Interaction (Chat), thay vì gửi Push cho mỗi tin nhắn, hệ thống sẽ gom (Batch) gửi 1 Push "Bạn có tin nhắn mới" sau 1 khoảng thời gian delay.
- **Thử lại (Retry)**: Áp dụng khi gọi FCM thất bại hoặc kết nối SSE bị đứt ngang. (Tuân thủ `[INV-N01]`).
- **Lưu DB (Persistence)**: Quyết định việc lưu bản ghi vào Database của Notification Service.
  - Các thông báo quan trọng sẽ được lưu lại để hiển thị trong mục **Notification Center (Inbox)** của ứng dụng.
  - Các thông báo dạng chat đã được lưu ở Interaction Service nên không cần lưu lại ở đây để tránh trùng lặp dữ liệu.

> [!IMPORTANT]
> **Quy tắc Vàng**: Notification Service **TUYỆT ĐỐI KHÔNG** được tự quyết định việc có nên gửi Push hay không. Nó chỉ thực thi dựa trên cấu hình (Policy) đã được truyền kèm theo Event từ các Core Services gửi sang. Bảng trên đóng vai trò là "Cấu hình mặc định" nếu Core Service không chỉ định cụ thể.

---

## 3. CẤU TRÚC PAYLOAD (ĐỀ XUẤT)

Khi một service khác muốn gửi thông báo, nó sẽ phát ra một Event (ví dụ: `NotificationRequested`) tuân thủ cấu hình trên:

```json
{
  "idempotencyKey": "uuid",
  "userId": "user-123",
  "type": "TRANSACTIONAL",
  "content": {
    "title": "Đặt lịch thành công!",
    "body": "Companion Aki đã chấp nhận cuộc hẹn của bạn."
  },
  "policyOverrides": {
    "requirePush": true
  }
}
```
