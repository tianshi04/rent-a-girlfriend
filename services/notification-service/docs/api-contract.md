# 🌐 GIAO TIẾP VỚI CLIENT (API CONTRACT)

Tài liệu này định nghĩa các RESTful APIs mà Notification Service cung cấp cho phía Client (Frontend Web / Mobile App) để tương tác với hệ thống. 

> [!IMPORTANT]
> **Quy tắc Xác thực (Authentication)**
> Toàn bộ các APIs dưới đây đều yêu cầu Client truyền HTTP Header:
> `Authorization: Bearer <JWT_TOKEN>`
> 
> Hệ thống áp dụng Auth Offloading. Backend sẽ nội suy ID của User từ JWT thông qua Istio Mesh. Do đó, trong URL không bao giờ có tham số `userId` (để tránh lỗi bảo mật BOLA/IDOR).

---

## 1. KẾT NỐI REALTIME (SSE)

Đây là endpoint dùng để mở luồng kết nối một chiều nhận dữ liệu realtime.

- **Endpoint**: `GET /v1/notifications/stream`
- **Mô tả chi tiết**: Vui lòng tham khảo tài liệu [Realtime Delivery (SSE)](./realtime-delivery.md).

---

## 2. QUẢN LÝ INBOX (LỊCH SỬ THÔNG BÁO)

### 2.1. Lấy danh sách thông báo (Fetch Inbox)
Sử dụng **Cursor-based Pagination** (Theo ADR-0004) để hỗ trợ tính năng Infinite Scroll (Cuộn vô cực) một cách mượt mà và không bị lỗi lặp data.

- **Endpoint**: `GET /v1/notifications`
- **Query Parameters**:
  - `limit` (int, optional): Số lượng bản ghi trả về. Mặc định `20`, tối đa `50`.
  - `cursor` (string, optional): Chuỗi Cursor lấy từ lần gọi trước để lấy trang tiếp theo. Để trống nếu tải lần đầu.
  - `unreadOnly` (boolean, optional): Nếu `true`, chỉ trả về các tin nhắn chưa đọc. Mặc định `false`.

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "11111111-2222-3333-4444-555555555555",
      "userId": "22222222-3333-4444-5555-666666666666",
      "eventId": "evt_123",
      "type": "TRANSACTIONAL",
      "priority": "HIGH",
      "payload": {
        "title": "Booking thành công!",
        "body": "Buổi hẹn của bạn đã được xác nhận.",
        "actionUrl": "rentagf://booking/detail/123",
        "imageUrl": "https://storage.rentagf.com/img.png"
      },
      "status": "PENDING",
      "readAt": null,
      "createdAt": "2026-05-10T14:00:00Z",
      "updatedAt": "2026-05-10T14:00:00Z"
    },
    {
      "id": "66666666-7777-8888-9999-000000000000",
      "userId": "22222222-3333-4444-5555-666666666666",
      "eventId": "evt_456",
      "type": "INTERACTION",
      "priority": "MEDIUM",
      "payload": {
        "title": "Tin nhắn mới",
        "body": "Client Nam vừa gửi tin nhắn cho bạn.",
        "actionUrl": "rentagf://chat/room/456"
      },
      "status": "PENDING",
      "readAt": "2026-05-09T09:35:00Z",
      "createdAt": "2026-05-09T09:30:00Z",
      "updatedAt": "2026-05-09T09:35:00Z"
    }
  ],
  "paging": {
    "nextCursor": "YmFzZTY0LWVuY29kZWQtY3Vyc29yLXN0cmluZw==",
    "hasMore": true
  }
}
```

---

### 2.2. Đánh dấu một thông báo Đã đọc
Khi user bấm vào một Notification để xem chi tiết.

- **Endpoint**: `PATCH /v1/notifications/{id}/read`
- **Path Parameters**:
  - `id` (UUID): ID của thông báo cần đánh dấu.
- **Request Body**: (Không có)

**Response (204 No Content):** (Không có body)

---

### 2.3. Đánh dấu tất cả Đã đọc
Dùng cho nút "Mark all as read" trên UI.

- **Endpoint**: `PATCH /v1/notifications/read-all`
- **Request Body**: (Không có)

**Response (200 OK):**
```json
{
  "affectedRows": 5
}
```

---

## 3. CHUẨN HÓA LỖI (ERROR RESPONSES)

Tất cả các API nếu gặp lỗi sẽ trả về cấu trúc thống nhất sau (HTTP Status 4xx, 5xx):

```json
{
  "error": {
    "code": "ERROR_CODE_STRING",
    "message": "Human readable error message for developers",
    "details": [] 
  }
}
```

**Các mã lỗi thường gặp:**
- `401 Unauthorized`: Lỗi do Istio chặn (Không có Token hoặc Token hết hạn).
- `400 Bad Request`: Thiếu tham số, sai format `cursor` (Code: `INVALID_CURSOR`).
- `404 Not Found`: Truy cập ID của thông báo không tồn tại hoặc không thuộc về User (Code: `NOTIFICATION_NOT_FOUND`).
- `500 Internal Server Error`: Lỗi DB hoặc xử lý logic (Code: `INTERNAL_ERROR`).
