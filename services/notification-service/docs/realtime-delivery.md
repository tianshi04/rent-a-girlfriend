# ⚡ HỢP ĐỒNG GIAO TIẾP THỜI GIAN THỰC (REALTIME DELIVERY - SSE)

Tài liệu này định nghĩa "Hợp đồng" (Contract) giữa Client (Mobile/Web) và Notification Service để duy trì kết nối thời gian thực thông qua công nghệ **Server-Sent Events (SSE)**.

---

## 1. TẠI SAO LẠI DÙNG SSE MÀ KHÔNG PHẢI WEBSOCKET?

- **Tính một chiều (One-way)**: Notification chỉ cần luồng dữ liệu từ Server đẩy xuống Client. Việc dùng WebSocket (hai chiều) là thừa thãi và tốn kém tài nguyên duy trì hơn.
- **Tích hợp dễ dàng**: SSE bản chất là HTTP Streaming, đi qua Load Balancer, Istio Mesh và các lớp Firewall dễ dàng hơn rất nhiều so với WebSocket.
- **Tự động kết nối lại**: Giao thức SSE có sẵn cơ chế tự động reconnect ở trình duyệt.

---

## 2. BÀI TOÁN XÁC THỰC (AUTHENTICATION & ISTIO MESH)

Đây là điểm sống còn của hệ thống. Vì Notification Service áp dụng triết lý **Auth Offloading** (Đẩy việc xác thực cho Istio Service Mesh), code Golang bên trong sẽ **KHÔNG** tự parse JWT.

1. **Client gửi Request**: Bắt buộc phải đính kèm Header `Authorization: Bearer <JWT_TOKEN>`.
2. **Istio Waypoint Proxy**: Đứng chặn trước Notification Service. Nó sẽ verify JWT (bằng JWKS của Identity Service).
   - Nếu lỗi: Istio trả thẳng `401 Unauthorized`. (Bảo vệ Notification Service khỏi request rác).
   - Nếu hợp lệ: Istio bóc tách Payload JWT và gắn `userId` vào Header (Ví dụ: `x-user-id: 1234`) rồi forward vào Notification Service.
3. **Notification Service (Golang)**: Chỉ cần đọc header `x-user-id` là tin tưởng 100% đó là user hợp lệ.

> [!WARNING]
> **Giới hạn của Web Frontend**: Trình duyệt mặc định cung cấp class `EventSource` để kết nối SSE, nhưng class native này **CẤM** truyền custom headers (như `Authorization`). 
> **=> Cách giải quyết (Tối ưu nhất)**: Team Web Frontend BẮT BUỘC phải sử dụng thư viện ngoài (Ví dụ: `@microsoft/fetch-event-source` cho React/JS) để có thể truyền Header JWT thay vì dùng `EventSource` nguyên bản. Tuyệt đối không truyền JWT qua Query URL (`?token=...`) vì rủi ro lộ lọt qua Access Logs của Nginx/Istio.

---

## 3. VÒNG ĐỜI KẾT NỐI (CONNECTION LIFECYCLE)

### 3.1. Handshake (Bắt tay kết nối)
- **Endpoint**: `GET /v1/notifications/stream`
- **Headers yêu cầu**:
  - `Accept: text/event-stream`
  - `Authorization: Bearer <token>`
- **Response**: Server giữ connection mở, trả về HTTP 200 kèm Headers:
  - `Content-Type: text/event-stream`
  - `Cache-Control: no-cache`
  - `Connection: keep-alive`

### 3.2. Heartbeat (Giữ nhiệt)
Để ngăn chặn Load Balancer (AWS ALB, Nginx) ngắt kết nối do "Idle Timeout" (thường là 60s), Server sẽ chủ động gửi một **Ping** mỗi 15 giây.
```text
: ping\n\n
```
*Lưu ý: Dấu `:` ở đầu biểu thị đây là một Comment trong chuẩn SSE. Trình duyệt sẽ tự động bỏ qua nó (không trigger event cho code JS), nhưng ở tầng TCP thì nó chứng minh kết nối vẫn còn sống.*

### 3.3. Dữ liệu thực (Data Event)
Khi có thông báo, Server trả về cấu trúc:
```text
id: e3b0c442-989b...
event: notification
data: {"title": "Hello", "body": "Bạn có tin nhắn mới", "type": "INTERACTION"}

```
*(Luôn kết thúc bằng 2 dấu xuống dòng `\n\n`)*

---

## 4. CHIẾN LƯỢC KẾT NỐI LẠI (RECONNECT STRATEGY)

Do đường truyền mạng (Mobile/Wifi) luôn có rủi ro chập chờn, hợp đồng quy định như sau:

1. **Auto Reconnect**: Khi mất kết nối, thư viện SSE của Client sẽ tự động nối lại sau khoảng 3s-5s.
2. **Missing Data Recovery (Khôi phục dữ liệu rơi rớt)**: 
   - Giả sử mạng đứt lúc 10:00, nối lại được lúc 10:05. Trong 5 phút đó, Notification Service đã đánh dấu User là **Offline** và luân chuyển tin nhắn sang nhánh **Push FCM** (hoặc lưu DB Inbox).
   - **Nhiệm vụ của Client**: Ngay khi nhận được tín hiệu kết nối SSE thành công trở lại, Client BẮT BUỘC phải gọi một API REST (Ví dụ: `GET /v1/notifications/inbox?unread=true`) để kéo những tin nhắn đã bị "miss" trong 5 phút đứt cáp đó về máy.

---

## 5. TÍCH HỢP REDIS PUB/SUB (NỘI BỘ SERVER)

Trong môi trường Cluster (nhiều Pods Notification Service), quy trình như sau:
1. Client Connect vào **Pod A**. Pod A lưu `userId` và `Response Writer` vào bộ nhớ RAM (Local Map).
2. Khi có Event gửi thông báo, Hệ thống Publish message vào topic Redis: `channel:user_{userId}:sse`.
3. **Pod A** (và các Pod khác) lắng nghe Redis. Chỉ Pod A thấy `userId` có trong RAM của mình, nó sẽ bốc data ra và gọi hàm `w.Write()` để đẩy xuống mạng TCP cho Client. Các Pod khác sẽ âm thầm bỏ qua message đó.
