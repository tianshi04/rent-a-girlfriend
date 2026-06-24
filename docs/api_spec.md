# Rent-a-Girlfriend Platform API Specification

Tài liệu này định nghĩa chi tiết các REST API endpoints của hệ thống **Rent-a-Girlfriend** dành cho các bên tích hợp hoặc đội ngũ Frontend (Web/Mobile) sử dụng để gọi vào cụm Microservices.

---

## 1. QUY CHUẨN CHUNG (GENERAL STANDARDS)

### 1.1. Base URL & Routing
* **Điểm truy cập chung (API Gateway):** `http://localhost:8080/api/v1`
* **Realtime Notification (SSE) Stream:** `http://localhost:8080/v1/notifications/stream`
* *Lưu ý:* Việc định tuyến giao thức HTTP được quản lý bởi **Istio Ambient Mode** thông qua các tài nguyên `Gateway` và `HTTPRoute` trong cụm Kubernetes.

### 1.2. JSON Casing
* Toàn bộ các trường dữ liệu trong JSON payload (Request Body & Response Body) **bắt buộc sử dụng `camelCase`** (ví dụ: `bookingId`, `availableBalance`, `reason`).

### 1.3. Cơ chế xác thực (Authentication & Header Injection)
* Việc xác thực chữ ký và tính hợp lệ của JWT token được **offload hoàn toàn cho Istio Waypoint Proxy**. Frontend chỉ cần gửi token qua Header chuẩn:
  ```http
  Authorization: Bearer <JWT_TOKEN>
  ```
* Sau khi xác thực thành công, Istio tự động trích xuất các claims và tiêm (inject) các header sau vào các request gửi tới các service nội bộ:
  * `user-id`: UUID của người dùng (từ claim `sub`)
  * `user-email`: Email của người dùng (từ claim `email`)
  * `user-role`: Vai trò của người dùng (`ACCOUNT_ROLE_CLIENT`, `ACCOUNT_ROLE_COMPANION`, `ACCOUNT_ROLE_ADMIN`)
  * `user-status`: Trạng thái tài khoản (`ACCOUNT_STATUS_ACTIVE`, `ACCOUNT_STATUS_LOCKED`)

### 1.4. Định dạng phản hồi thành công (Success Response - Naked JSON)
* Hệ thống áp dụng chuẩn **Naked JSON** cho phản hồi thành công. Dữ liệu tài nguyên được trả về trực tiếp ở root level, không có vỏ bọc (envelope) như `data` hay `success`.
* Ví dụ:
  ```json
  {
    "bookingId": "bk_cinema_888",
    "status": "ACCEPTED"
  }
  ```

### 1.5. Định dạng phản hồi lỗi (Error Response - gRPC Status Model)
* Khi gặp lỗi nghiệp vụ hoặc validate dữ liệu, hệ thống trả về mã trạng thái HTTP phù hợp (4xx/5xx) kèm body lỗi chuẩn hóa theo cấu trúc của gRPC-Gateway ở Root Level:
  ```json
  {
    "code": 3,
    "message": "Thời gian bắt đầu cuộc hẹn phải lớn hơn thời gian hiện tại ít nhất 2 giờ.",
    "details": [
      {
        "field": "startTime",
        "description": "Thời gian không hợp lệ [INV-B01]"
      }
    ]
  }
  ```
* **Bảng ánh xạ mã lỗi từ gRPC sang HTTP Status Code:**
  * `3 (INVALID_ARGUMENT)` &rarr; `400 Bad Request`
  * `5 (NOT_FOUND)` &rarr; `404 Not Found`
  * `6 (ALREADY_EXISTS)` &rarr; `409 Conflict`
  * `7 (PERMISSION_DENIED)` &rarr; `403 Forbidden`
  * `16 (UNAUTHENTICATED)` &rarr; `401 Unauthorized`
  * `13 (INTERNAL)` &rarr; `500 Internal Server Error`

---

## 2. CHI TIẾT CÁC ENDPOINTS THEO SERVICE

### 2.1. IDENTITY & AUTHENTICATION SERVICE
Dịch vụ quản lý xác thực tài khoản Google OAuth2 và các luồng quản trị tài khoản của Admin.

#### `GET /auth/google/init`
* **Authentication:** None (Public)
* **Mô tả:** Lấy URL đăng nhập Google OAuth2 để chuyển hướng người dùng.
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "authUrl": "https://accounts.google.com/o/oauth2/v2/auth?client_id=...",
    "state": "state123",
    "codeChallenge": "challenge123"
  }
  ```

#### `GET /auth/google/callback`
* **Authentication:** None (Public)
* **Mô tả:** Tiếp nhận Authorization Code và xử lý đăng nhập, cấp Access/Refresh Token.
* **Query Parameters:**
  * `code` (Bắt buộc): Mã xác thực từ Google.
  * `state` (Tùy chọn): Chuỗi bảo mật chống tấn công CSRF.
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "accessToken": "eyJhbGciOiJSUzI1NiIs...",
    "refreshToken": "ref_rot_abc123xyz...",
    "expiresIn": 3600
  }
  ```

#### `POST /auth/refresh`
* **Authentication:** None (Public)
* **Mô tả:** Đổi Refresh Token cũ lấy cặp token mới (Refresh Token Rotation).
* **Request Body:**
  ```json
  {
    "refreshToken": "ref_rot_abc123xyz..."
  }
  ```
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "accessToken": "eyJhbGciOiJSUzI1NiIs...",
    "refreshToken": "ref_rot_new_key_456...",
    "expiresIn": 3600
  }
  ```

#### `POST /auth/logout`
* **Authentication:** Bearer JWT
* **Mô tả:** Đăng xuất người dùng bằng cách thu hồi Refresh Token hiện tại.
* **Request Body:**
  ```json
  {
    "refreshToken": "ref_rot_new_key_456..."
  }
  ```
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "message": "logged out"
  }
  ```

#### `POST /upgrade-requests`
* **Authentication:** Bearer JWT (Yêu cầu vai trò `CLIENT`)
* **Mô tả:** Client gửi yêu cầu nâng cấp tài khoản lên Companion.
* **Request Body:**
  ```json
  {
    "reason": "Hãy để tôi đóng vai một người bạn gái hoàn hảo..."
  }
  ```
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "message": "upgrade request submitted"
  }
  ```

#### `GET /admin/upgrade-requests`
* **Authentication:** Bearer JWT (Yêu cầu vai trò `ADMIN`)
* **Mô tả:** Admin lấy danh sách phân trang các yêu cầu nâng cấp đang ở trạng thái chờ duyệt.
* **Query Parameters:**
  * `status` (Tùy chọn): Lọc theo trạng thái yêu cầu (`UPGRADE_STATUS_PENDING`, `UPGRADE_STATUS_APPROVED`, `UPGRADE_STATUS_REJECTED`).
  * `page` (Mặc định 1): Trang hiện tại.
  * `pageSize` (Mặc định 10): Số lượng bản ghi mỗi trang.
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "data": [
      {
        "id": "req_up_123",
        "userId": "usr_kazuya_001",
        "status": "UPGRADE_STATUS_PENDING",
        "reason": "Hãy để tôi đóng vai một người bạn gái hoàn hảo...",
        "rejectReason": "",
        "reviewedBy": "",
        "reviewedAt": null,
        "createdAt": "2026-06-20T10:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 10
  }
  ```

#### `POST /admin/upgrade-requests/{id}/approve`
* **Authentication:** Bearer JWT (Yêu cầu vai trò `ADMIN`)
* **Mô tả:** Admin duyệt yêu cầu nâng cấp lên Companion cho tài khoản.
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "message": "upgrade approved"
  }
  ```

#### `POST /admin/upgrade-requests/{id}/reject`
* **Authentication:** Bearer JWT (Yêu cầu vai trò `ADMIN`)
* **Mô tả:** Admin từ chối yêu cầu nâng cấp.
* **Request Body:**
  ```json
  {
    "reason": "Mô tả lý do từ chối (ảnh chưa phù hợp, thiếu thông tin...)"
  }
  ```
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "message": "upgrade rejected"
  }
  ```

#### `GET /admin/accounts/{id}`
* **Authentication:** Bearer JWT (Yêu cầu vai trò `ADMIN`)
* **Mô tả:** Admin lấy chi tiết trạng thái tài khoản người dùng bất kỳ.
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "role": "ACCOUNT_ROLE_COMPANION",
    "status": "ACCOUNT_STATUS_ACTIVE",
    "violationCount": 0
  }
  ```

#### `POST /admin/accounts/{id}/lock`
* **Authentication:** Bearer JWT (Yêu cầu vai trò `ADMIN`)
* **Mô tả:** Admin thực hiện khóa tài khoản vi phạm.
* **Request Body:**
  ```json
  {
    "reason": "Hành vi gian lận thanh toán hoặc vi phạm quy định hủy hẹn."
  }
  ```
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "message": "account locked"
  }
  ```

#### `POST /admin/accounts/{id}/unlock`
* **Authentication:** Bearer JWT (Yêu cầu vai trò `ADMIN`)
* **Mô tả:** Admin mở khóa cho tài khoản bị phạt.
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "message": "account unlocked"
  }
  ```

---

### 2.2. PROFILE & CATALOGUE SERVICE
Dịch vụ quản lý thông tin Companion, danh mục kịch bản (Scenario) và tài nguyên media.

#### `GET /companions`
* **Authentication:** None (Public)
* **Mô tả:** Tìm kiếm và lọc danh sách Companion đã được duyệt hoạt động.
* **Query Parameters:**
  * `name`: Lọc theo tên (partial match).
  * `city`: Lọc theo thành phố hoạt động (ví dụ: `Hanoi`, `HCM`, `Danang`).
  * `minPrice` / `maxPrice`: Khoảng giá kịch bản dịch vụ.
  * `page` (mặc định 1): Trang hiện tại.
  * `pageSize` (mặc định 10): Số lượng bản ghi mỗi trang.
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "data": [
      {
        "companionId": "cmp_chizuru_123",
        "displayName": "Chizuru Ichinose",
        "avatarUrl": "https://storage.rent-a-gf.com/avatars/chizuru.png",
        "averageRating": 4.9,
        "city": "HCM",
        "startingPrice": 300
      }
    ],
    "total": 45,
    "page": 1,
    "pageSize": 10
  }
  ```

#### `GET /companions/{companion_id}`
* **Authentication:** None (Public)
* **Mô tả:** Lấy chi tiết hồ sơ Companion dạng Magazine View (bao gồm danh sách Scenario).
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "companionId": "cmp_chizuru_123",
    "displayName": "Chizuru Ichinose",
    "introText": "Hãy để tôi đóng vai một người bạn gái hoàn hảo trong buổi hẹn hò của bạn.",
    "avatarUrl": "https://storage.rent-a-gf.com/avatars/chizuru.png",
    "availableCities": ["HCM", "Danang"],
    "voiceIntroUrl": "https://storage.rent-a-gf.com/voice/chizuru.mp3?X-Amz-Signature=...",
    "albumUrls": [
      "https://storage.rent-a-gf.com/albums/chizuru_1.png",
      "https://storage.rent-a-gf.com/albums/chizuru_2.png"
    ],
    "averageRating": 4.9,
    "totalReviews": 150,
    "status": "APPROVED",
    "scenarios": [
      {
        "id": "scn_cinema_01",
        "title": "Hẹn hò xem phim lãng mạn",
        "description": "Cùng đi xem phim, chia sẻ bắp rang bơ...",
        "price": 300,
        "duration": 120,
        "status": "ACTIVE"
      }
    ]
  }
  ```

#### `GET /profile/me`
* **Authentication:** Bearer JWT (Yêu cầu vai trò `COMPANION`)
* **Mô tả:** Companion xem thông tin hồ sơ của chính mình.
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "companionId": "cmp_chizuru_123",
    "displayName": "Chizuru Ichinose",
    "introText": "Hãy để tôi đóng vai một người bạn gái hoàn hảo trong buổi hẹn hò của bạn.",
    "avatarUrl": "https://storage.rent-a-gf.com/avatars/chizuru.png",
    "availableCities": ["HCM", "Danang"],
    "voiceIntroUrl": "https://storage.rent-a-gf.com/voice/chizuru.mp3?X-Amz-Signature=...",
    "albumUrls": [],
    "averageRating": 4.9,
    "totalReviews": 150,
    "status": "APPROVED",
    "scenarios": [
      {
        "id": "scn_cinema_01",
        "title": "Hẹn hò xem phim lãng mạn",
        "description": "Cùng đi xem phim, chia sẻ bắp rang bơ...",
        "price": 300,
        "duration": 120,
        "status": "ACTIVE"
      }
    ]
  }
  ```

#### `POST /profile/me/media/presigned-urls`
* **Authentication:** Bearer JWT (Yêu cầu vai trò `COMPANION`)
* **Mô tả:** Xin link Presigned URL để upload file (Ảnh chân dung/Album hoặc Voice Intro) trực tiếp lên Cloud Storage.
* **Ràng buộc nghiệp vụ:**
  * Ảnh (IMAGE): Kích thước tối đa **2MB** `[INV-P05]`.
  * Âm thanh (VOICE): Kích thước tối đa **5MB** và độ dài không quá **30 giây** `[INV-P04]`.
* **Request Body:**
  ```json
  {
    "assetType": "VOICE",
    "sizeBytes": 3200000,
    "durationSeconds": 24,
    "contentType": "audio/mp3"
  }
  ```
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "uploadUrl": "https://storage.rent-a-gf.com/voice/chizuru.mp3?X-Amz-Signature=...",
    "fileUrl": "https://storage.rent-a-gf.com/voice/chizuru.mp3"
  }
  ```

#### `POST /profile/me/scenarios`
* **Authentication:** Bearer JWT (Yêu cầu vai trò `COMPANION`)
* **Mô tả:** Companion tạo kịch bản dịch vụ mới.
* **Ràng buộc nghiệp vụ:**
  * Giá tiền (`price`) phải lớn hơn 0 Kano-Coin `[INV-P01]`.
  * Thời lượng (`durationMinutes`) phải thuộc mốc quy chuẩn: `60`, `120`, hoặc `180` phút `[INV-P02]`.
  * Số lượng kịch bản hoạt động tối đa là **5** kịch bản `[INV-P03]`.
* **Request Body:**
  ```json
  {
    "title": "Hẹn hò rạp chiếu phim",
    "description": "Cùng đi xem phim và đi dạo trò chuyện...",
    "price": 300,
    "durationMinutes": 120
  }
  ```
* **Phản hồi thành công (201 Created):**
  ```json
  {
    "scenarioId": "scn_cinema_01"
  }
  ```

#### `PUT /profile/me/scenarios/{scenario_id}`
* **Authentication:** Bearer JWT (Yêu cầu vai trò `COMPANION`)
* **Mô tả:** Cập nhật thông tin chi tiết hoặc trạng thái kịch bản (ACTIVE / INACTIVE).
* **Request Body:**
  ```json
  {
    "title": "Hẹn hò rạp chiếu phim v2",
    "description": "Cập nhật mô tả kịch bản...",
    "price": 350,
    "durationMinutes": 120,
    "status": "ACTIVE"
  }
  ```
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "success": true
  }
  ```

#### `DELETE /profile/me/scenarios/{scenario_id}`
* **Authentication:** Bearer JWT (Yêu cầu vai trò `COMPANION`)
* **Mô tả:** Xóa kịch bản của Companion.
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "success": true
  }
  ```

---

### 2.3. BOOKING SERVICE
Dịch vụ lõi xử lý chu kỳ sống của cuộc hẹn. Toàn bộ các thao tác cập nhật trạng thái (accept, reject, cancel, complete) đều sử dụng phương thức **`POST`** thay vì `PUT` do sự tương thích ánh xạ gRPC-Gateway.

#### `POST /bookings`
* **Authentication:** Bearer JWT (Yêu cầu vai trò `CLIENT`)
* **Mô tả:** Đặt lịch hẹn mới. Hệ thống sẽ gọi gRPC nội bộ sang Finance để đóng băng tiền (`FreezeCoin`).
* **Ràng buộc nghiệp vụ:**
  * Giờ hẹn bắt đầu (`startTime`) phải cách thời điểm đặt tối thiểu **2 giờ** (MinAdvanceBookingHours = 2) `[INV-B01]`.
  * Client không được có 2 cuộc hẹn bị chồng chéo thời gian (overlap) `[INV-B06]`.
* **Request Body:**
  ```json
  {
    "companionId": "cmp_chizuru_123",
    "scenarioId": "scn_cinema_01",
    "startTime": "2026-06-20T19:00:00Z"
  }
  ```
* **Phản hồi thành công (200 OK - Trả về thông tin tóm tắt khởi tạo thành công):**
  ```json
  {
    "bookingId": "bk_cinema_888",
    "status": "BOOKING_STATUS_PENDING_RESERVING",
    "message": "Booking request created successfully"
  }
  ```

#### `GET /bookings`
* **Authentication:** Bearer JWT
* **Mô tả:** Lấy danh sách lịch sử lịch hẹn của bản thân (cả Client/Companion). Trả về danh sách phẳng kèm con trỏ phân trang `nextPageToken`.
* **Query Parameters:**
  * `status`: Lọc theo trạng thái cuộc hẹn (`BOOKING_STATUS_PENDING_RESERVING`, `BOOKING_STATUS_PENDING`, `BOOKING_STATUS_ACCEPTED`, `BOOKING_STATUS_COMPLETED`, `BOOKING_STATUS_CANCELLED`, `BOOKING_STATUS_DISPUTED`, `BOOKING_STATUS_RESOLVED`).
  * `page` (mặc định 1) / `pageSize` (mặc định 10).
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "bookings": [
      {
        "bookingId": "bk_cinema_888",
        "clientId": "cli_kazuya_001",
        "companionId": "cmp_chizuru_123",
        "price": 300,
        "durationMinutes": 120,
        "startTime": "2026-06-20T19:00:00Z",
        "endTime": "2026-06-20T21:00:00Z",
        "status": "BOOKING_STATUS_ACCEPTED",
        "createdAt": "2026-06-20T17:00:00Z"
      }
    ],
    "nextPageToken": "eyJwYWdlIjoyfQ=="
  }
  ```

#### `GET /bookings/{booking_id}`
* **Authentication:** Bearer JWT
* **Mô tả:** Lấy thông tin chi tiết phẳng của một cuộc hẹn cụ thể. (Lưu ý: ID phòng chat tương ứng chính là `booking_id`).
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "bookingId": "bk_cinema_888",
    "clientId": "cli_kazuya_001",
    "companionId": "cmp_chizuru_123",
    "price": 300,
    "durationMinutes": 120,
    "startTime": "2026-06-20T19:00:00Z",
    "endTime": "2026-06-20T21:00:00Z",
    "status": "BOOKING_STATUS_ACCEPTED",
    "createdAt": "2026-06-20T17:00:00Z"
  }
  ```

#### `POST /bookings/{booking_id}/accept`
* **Authentication:** Bearer JWT (Yêu cầu vai trò `COMPANION`)
* **Mô tả:** Companion đồng ý nhận cuộc hẹn. Tiền đặt cọc được chuyển sang dạng ký quỹ (Escrow) và kích hoạt phòng chat.
* **Ràng buộc nghiệp vụ:**
  * Hành động duyệt phải diễn ra trong vòng 12h kể từ lúc tạo và cách `startTime` ít nhất 1 giờ.
  * Companion không được accept các booking bị trùng thời gian hoạt động `[INV-B07]`.
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "bookingId": "bk_cinema_888",
    "status": "BOOKING_STATUS_ACCEPTED",
    "message": "Booking accepted and escrow holds processed"
  }
  ```

#### `POST /bookings/{booking_id}/reject`
* **Authentication:** Bearer JWT (Yêu cầu vai trò `COMPANION`)
* **Mô tả:** Companion từ chối cuộc hẹn. Hệ thống tự động giải phóng cọc (`Unfreeze`) trả lại ví Client.
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "bookingId": "bk_cinema_888",
    "status": "BOOKING_STATUS_CANCELLED",
    "message": "Booking rejected successfully"
  }
  ```

#### `POST /bookings/{booking_id}/cancel`
* **Authentication:** Bearer JWT
* **Mô tả:** Hủy cuộc hẹn đã chấp nhận (ACCEPTED).
* **Quy tắc hoàn tiền và phạt vi phạm:**
  * **Hủy sớm (> 24 giờ so với giờ hẹn):** Hoàn cọc 100% về ví Client.
  * **Hủy muộn (<= 24 giờ):**
    * Nếu Client chủ động hủy muộn: Client mất 100% tiền cọc chuyển sang ví đền bù cho Companion.
    * Nếu Companion chủ động hủy muộn: Client được hoàn 100% tiền cọc. Companion bị ghi nhận **1 lần vi phạm**.
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "bookingId": "bk_cinema_888",
    "status": "BOOKING_STATUS_CANCELLED",
    "message": "Booking cancelled successfully"
  }
  ```

#### `POST /bookings/{booking_id}/complete`
* **Authentication:** None (Thường chạy tự động bởi hệ thống)
* **Mô tả:** Đánh dấu hoàn thành cuộc hẹn.
* *Lưu ý:* API này chủ yếu phục vụ Worker System Scheduler chạy ngầm để đóng cuộc hẹn sau `endTime` + 12 giờ nếu không có khiếu nại phát sinh để tự động giải ngân (payout) tiền cọc.
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "bookingId": "bk_cinema_888",
    "status": "BOOKING_STATUS_COMPLETED",
    "message": "Booking completed successfully"
  }
  ```

---

### 2.4. FINANCE & WALLET SERVICE
Dịch vụ quản lý số dư ví, nạp tiền quy đổi kano-coin và VNPay integration.

#### `GET /finance/wallet`
* **Authentication:** Bearer JWT
* **Mô tả:** Lấy thông tin số dư ví hiện tại của người dùng.
* **Query Parameters:**
  * `userId` (Bắt buộc): ID người dùng cần kiểm tra số dư.
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "walletId": "wall_kazuya_001",
    "userId": "550e8400-e29b-41d4-a716-446655440000",
    "availableBalance": 520,
    "frozenBalance": 300
  }
  ```

#### `POST /finance/topup`
* **Authentication:** Bearer JWT (Yêu cầu vai trò `CLIENT`)
* **Mô tả:** Tạo đơn nạp tiền quy đổi (Tỷ lệ: `1 Kano-Coin = 1,000 VNĐ`). Trả về link để redirect người dùng sang cổng VNPay Sandbox.
* **Request Body:**
  ```json
  {
    "userId": "550e8400-e29b-41d4-a716-446655440000",
    "amount": 500
  }
  ```
* **Phản hồi thành công (201 Created):**
  ```json
  {
    "paymentUrl": "https://sandbox.vnpayment.vn/paymentv2/vpcpay.html?vnp_Amount=50000000&..."
  }
  ```

#### `GET /finance/vnpay-ipn`
* **Authentication:** None (Webhook từ VNPay)
* **Mô tả:** Endpoint Webhook nhận kết quả từ VNPay để cập nhật số dư ví thực tế an toàn. Đảm bảo tính Idempotency tuyệt đối.
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "RspCode": "00",
    "Message": "Confirm success"
  }
  ```

#### `GET /finance/vnpay-return`
* **Authentication:** None (Redirect từ Trình duyệt)
* **Mô tả:** Trang kết quả giao diện thanh toán trả về HTML hiển thị kết quả cho người dùng sau khi thanh toán xong trên VNPay.
* **Phản hồi thành công (200 OK):**
  * *Trả về trực tiếp mã nguồn HTML cao cấp.*

---

### 2.5. INTERACTION & CHAT ROOM SERVICE
Dịch vụ quản lý chat room và hệ thống đánh giá (Review).

#### `POST /interaction/rooms/{room_id}/messages`
* **Authentication:** Bearer JWT
* **Mô tả:** Gửi tin nhắn mới vào phòng chat của cuộc hẹn (Chỉ gửi được khi trạng thái phòng chat là `ACTIVE`). ID của phòng chat (`room_id`) chính là ID cuộc hẹn (`booking_id`).
* **Request Body:**
  ```json
  {
    "text": "Ok bạn, mình sẽ đến đúng giờ."
  }
  ```
* **Phản hồi thành công (201 Created):**
  ```json
  {
    "messageId": "msg_002",
    "roomId": "chat_room_888",
    "senderId": "cmp_chizuru_123",
    "content": "Ok bạn, mình sẽ đến đúng giờ.",
    "createdAt": "2026-06-18T11:02:00Z"
  }
  ```

#### `GET /interaction/rooms/{room_id}/messages`
* **Authentication:** Bearer JWT
* **Mô tả:** Lấy lịch sử chat của phòng.
* **Query Parameters:**
  * `limit` (mặc định 50): Giới hạn tin nhắn.
  * `offset` (mặc định 0): Bỏ qua số bản ghi để phân trang.
* **Phản hồi thành công (200 OK):**
  ```json
  [
    {
      "messageId": "msg_001",
      "roomId": "chat_room_888",
      "senderId": "cli_kazuya_001",
      "content": "Chào bạn, mình hẹn gặp nhau đúng giờ nhé!",
      "createdAt": "2026-06-18T11:00:00Z"
    }
  ]
  ```

#### `POST /interaction/reviews`
* **Authentication:** Bearer JWT (Yêu cầu vai trò `CLIENT`)
* **Mô tả:** Gửi đánh giá và bình luận sau cuộc hẹn.
* **Ràng buộc nghiệp vụ:**
  * Điểm đánh giá (`rating`) nằm trong khoảng `[1, 5]` `[INV-I05]`.
  * Mỗi cuộc hẹn chỉ được đánh giá **duy nhất 1 lần**, không được sửa đổi hay xóa `[INV-I04]`.
  * Chỉ được gửi review từ sau mốc thời gian kết thúc cuộc hẹn (`endTime`).
* **Request Body:**
  ```json
  {
    "bookingId": "bk_cinema_888",
    "clientId": "cli_kazuya_001",
    "companionId": "cmp_chizuru_123",
    "rating": 5,
    "comment": "Dịch vụ rất tuyệt vời!"
  }
  ```
* **Phản hồi thành công (201 Created):**
  ```json
  {
    "reviewId": "rev_777",
    "status": "SUBMITTED",
    "message": "Review submitted successfully."
  }
  ```

#### `GET /interaction/reviews/companion/{companion_id}`
* **Authentication:** None (Public)
* **Mô tả:** Public API lấy danh sách toàn bộ các đánh giá của Companion.
* **Phản hồi thành công (200 OK):**
  ```json
  [
    {
      "reviewId": "rev_777",
      "bookingId": "bk_cinema_888",
      "clientId": "cli_kazuya_001",
      "companionId": "cmp_chizuru_123",
      "rating": 5,
      "comment": "Dịch vụ rất tuyệt vời!",
      "createdAt": "2026-06-20T21:30:00Z",
      "updatedAt": "2026-06-20T21:30:00Z"
    }
  ]
  ```

#### `GET /interaction/rooms`
* **Authentication:** Bearer JWT
* **Mô tả:** Lấy danh sách toàn bộ phòng chat (cả đang hoạt động và đã khóa) liên quan tới người dùng hiện tại (Client hoặc Companion).
* **Phản hồi thành công (200 OK):**
  ```json
  [
    {
      "roomId": "chat_room_888",
      "bookingId": "bk_cinema_888",
      "clientId": "cli_kazuya_001",
      "companionId": "cmp_chizuru_123",
      "status": "ACTIVE",
      "lockAt": "2026-06-18T12:00:00Z",
      "createdAt": "2026-06-18T10:00:00Z",
      "updatedAt": "2026-06-18T10:00:00Z"
    }
  ]
  ```

---

### 2.6. DISPUTE & REPORT SERVICE
Dịch vụ tiếp nhận khiếu nại và hỗ trợ Admin phân xử tiền ký quỹ.

#### `POST /disputes`
* **Authentication:** Bearer JWT (Đối tác tham gia cuộc hẹn)
* **Mô tả:** Khởi tố khiếu nại (báo cáo vi phạm, companion no-show). Hệ thống sẽ lập tức đóng băng Escrow để chờ Admin giải quyết.
* **Request Body:**
  ```json
  {
    "bookingId": "bk_cinema_888",
    "accusedId": "cmp_chizuru_123",
    "reason": "NO_SHOW",
    "evidences": [
      {
        "evidenceType": "IMAGE",
        "content": "https://storage.rent-a-gf.com/evidence/no_show_proof.png"
      }
    ]
  }
  ```
* **Phản hồi thành công (201 Created):**
  ```json
  {
    "disputeId": "dis_abc999"
  }
  ```

#### `GET /disputes`
* **Authentication:** Bearer JWT (Yêu cầu vai trò `ADMIN`)
* **Mô tả:** Admin lấy danh sách các tranh chấp trong hệ thống.
* **Query Parameters:**
  * `status`: Trạng thái khiếu nại (`OPEN`, `RESOLVING`, `RESOLVED`).
  * `page` (mặc định 1) / `pageSize` (mặc định 10).
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "disputes": [
      {
        "disputeId": "dis_abc999",
        "bookingId": "bk_cinema_888",
        "reporterId": "cli_kazuya_001",
        "accusedId": "cmp_chizuru_123",
        "reason": "NO_SHOW",
        "status": "OPEN",
        "adminId": null,
        "resolution": null,
        "notes": null,
        "version": 1,
        "evidences": [
          {
            "evidenceId": "ev_001",
            "evidenceType": "IMAGE",
            "content": "https://storage.rent-a-gf.com/evidence/no_show_proof.png"
          }
        ]
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 10
  }
  ```

#### `GET /disputes/{dispute_id}`
* **Authentication:** Bearer JWT (Yêu cầu vai trò `ADMIN`)
* **Mô tả:** Xem chi tiết một vụ khiếu nại tranh chấp.
* **Phản hồi thành công (200 OK):**
  * *Body chi tiết tương tự bản ghi thuộc danh sách trên.*

#### `GET /disputes/{dispute_id}/saga`
* **Authentication:** Bearer JWT (Yêu cầu vai trò `ADMIN`)
* **Mô tả:** Lấy trạng thái hiện tại của Saga phân xử tài chính (hoàn tiền hoặc giải ngân) cho tranh chấp này.
* **Phản hồi thành công (200 OK):**
  * Trả về chi tiết SAGA hoặc `null` nếu chưa bắt đầu SAGA.
  ```json
  {
    "sagaId": "saga_rf_abc999",
    "disputeId": "dis_abc999",
    "bookingId": "bk_cinema_888",
    "sagaType": "REFUND",
    "currentState": "COMPENSATING",
    "retryCount": 0,
    "lastError": null,
    "version": 1
  }
  ```

#### `POST /disputes/{dispute_id}/resolve`
* **Authentication:** Bearer JWT (Yêu cầu vai trò `ADMIN`)
* **Mô tả:** Admin giải quyết tranh chấp và đưa ra quyết định xử lý dòng tiền.
* **Lựa chọn phán quyết (`resolution`):**
  * `REFUND_CLIENT`: Hoàn lại 100% tiền cọc cho khách hàng.
  * `PAYOUT_COMPANION`: Giải ngân 100% tiền cho Companion.
  * `REJECT_DISPUTE`: Từ chối khiếu nại (hoàn tất cuộc hẹn bình thường).
* **Request Body:**
  ```json
  {
    "resolution": "REFUND_CLIENT",
    "notes": "Companion xác nhận no-show, hoàn trả cọc 100% cho Client."
  }
  ```
* **Phản hồi thành công (200 OK):**
  ```json
  {
    "success": true
  }
  ```

---

### 2.7. REALTIME NOTIFICATIONS (SSE)
Dịch vụ cung cấp stream thông báo thời gian thực thông qua Server-Sent Events (SSE).

#### `GET /v1/notifications/stream`
* **Authentication:** Bearer JWT (Headers được kiểm tra bởi Istio)
* **Mô tả:** Đăng ký kênh truyền thông tin một chiều thời gian thực từ hệ thống tới thiết bị client.
* **Response Headers:**
  * `Content-Type: text/event-stream`
  * `Cache-Control: no-cache`
  * `Connection: keep-alive`
* **Cơ chế hoạt động:**
  * Hệ thống duy trì gói tin heartbeat `: ping\n\n` định kỳ **15 giây** `[INV-N05]`.
  * Tự động ngắt kết nối dọn dẹp bộ nhớ sau **30 phút** idle kết nối `[INV-N04]`.
* **Mẫu luồng tin nhắn (Event Output Chunks):**
  * *Khi có Yêu cầu đặt lịch mới (Đẩy tới Companion):*
    ```event
    event: notification
    data: {"id":"nt_1","type":"BOOKING_REQUESTED","title":"Yêu cầu đặt lịch mới","body":"Bạn có yêu cầu hẹn hò xem phim mới từ Kazuya.","bookingId":"bk_cinema_888"}
    ```
  * *Khi Companion đồng ý cuộc hẹn (Đẩy tới Client):*
    ```event
    event: notification
    data: {"id":"nt_2","type":"BOOKING_ACCEPTED","title":"Chấp nhận lịch hẹn","body":"Chizuru đã đồng ý cuộc hẹn của bạn! Phòng chat hiện đã mở.","bookingId":"bk_cinema_888"}
    ```

---

## 3. CHUẨN ĐỊNH DẠNG SỰ KIỆN BẤT ĐỒNG BỘ (EVENT BUS)

Đối với các hệ thống backend lắng nghe sự kiện bất đồng bộ qua **Kafka Message Broker**, mọi tin nhắn bắt buộc phải được đóng gói theo chuẩn **CloudEvents (Structured Content Mode)** dưới cấu trúc sau:

### 3.1. Phong bì CloudEvents chuẩn (CE Envelope Schema)
```json
{
  "specversion": "1.0",
  "id": "evt_abc123",
  "source": "/rent-a-gf/booking-context/booking/bk_999",
  "type": "booking.booking-accepted.v1",
  "datacontenttype": "application/json",
  "time": "2026-06-20T10:00:00Z",
  "correlationid": "req_xyz789",
  "data": {
    "bookingId": "bk_999",
    "sagaId": "saga_555",
    "companionId": "cmp_123",
    "price": 300
  }
}
```

### 3.2. Quy tắc đặt tên sự kiện (Event Naming & Keying)
* **Quy chuẩn đặt tên `type`:** `<domain>.<event-name>.v<version>` (kebab-case, ví dụ `booking.booking-accepted.v1`).
* **Quy chuẩn Casing:** Toàn bộ dữ liệu nghiệp vụ nằm trong trường `data` bắt buộc viết dưới dạng **`camelCase`**. Riêng các trường extension ngoài envelope (như `correlationid`) bắt buộc viết thường toàn bộ.
* **Khóa tin nhắn (Kafka Message Key):** Bắt buộc phải chọn ID của **Aggregate Root** chính liên quan đến sự kiện (như `bookingId`, `userId`) làm Kafka Message Key để đảm bảo tính tuần tự xử lý trên cùng một Partition (Partition Order Guarantee).
