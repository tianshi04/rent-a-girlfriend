# MÔ HÌNH HÓA CHIẾN THUẬT (TACTICAL MODELING)

Dựa trên kết quả của quá trình Event Storming (từ Big Picture, Processing Level đến Design Level), tài liệu này đi sâu vào việc thiết kế kiến trúc bên trong của các Bounded Context sử dụng các mẫu thiết kế của Tactical Domain-Driven Design (DDD): **Aggregate Roots**, **Entities**, **Value Objects**, **Domain Services**, và **Repositories**.

Việc xác định rõ ràng các thành phần này giúp chuẩn bị trực tiếp cho quá trình implement (viết code) ở các microservices tương ứng.

---

## 1. Bounded Context: BOOKING

**Trách nhiệm:** Quản lý vòng đời, trạng thái của cuộc hẹn và đảm bảo tính hợp lệ về mặt thời gian, quy tắc nghiệp vụ.

### 1.1. Aggregates & Entities
*   **Aggregate Root: `Booking`**
    *   Đại diện cho một yêu cầu/cuộc hẹn. Mọi thao tác thay đổi trạng thái phải đi qua Aggregate này.
    *   *Properties:* `BookingId`, `Status` (PENDING, ACCEPTED, REJECTED, COMPLETED, CANCELLED).

### 1.2. Value Objects
*   `BookingId`: Định danh duy nhất (UUID).
*   `ClientId`, `CompanionId`: Tham chiếu đến Identity Context (UUID).
*   `TimeRange`: Đóng gói `StartTime` và `EndTime`, có logic kiểm tra thời gian bắt đầu phải trước thời gian kết thúc, và khoảng cách phải bằng đúng thời lượng kịch bản.
*   `ScenarioSnapshot`: Đóng gói `Price` (Money) và `Duration` tại thời điểm đặt lịch để đảm bảo tính bất biến dù Companion có đổi giá sau đó.

### 1.3. Domain Services
*   `BookingExpirationService`: Service chứa logic kiểm tra xem một Booking có vượt quá thời gian chờ (Timeout) phản hồi hay không để tự động chuyển sang `CANCELLED`.
*   `BookingCompletionService`: Service chứa logic nghiệp vụ tự động hoàn thành Booking khi đã qua thời gian `EndTime` + thời gian quy định (VD: 12h) mà không có khiếu nại.

### 1.4. Repositories
*   `IBookingRepository`: Lưu trữ và phục hồi trạng thái của `Booking` Aggregate.

---

## 2. Bounded Context: FINANCE

**Trách nhiệm:** Quản lý ví ảo (Kano-Coin), xử lý giao dịch, giữ tiền cọc (Escrow) và tính toán hoa hồng.

### 2.1. Aggregates & Entities
*   **Aggregate Root: `Wallet`**
    *   Quản lý số dư của User.
    *   *Entities con:* `TransactionLine` (Lưu lịch sử dòng tiền vào/ra trực tiếp trong ví để đảm bảo truy xuất nhanh và toàn vẹn khi cộng/trừ số dư).
*   **Aggregate Root: `Escrow`**
    *   Quản lý khoản tiền bị khóa tạm thời cho một Booking cụ thể.
    *   *Properties:* `Status` (HELD, REFUNDED, PAID_OUT).

### 2.2. Value Objects
*   `Money`: Đóng gói giá trị số tiền (Amount) và đơn vị tiền tệ (Currency - mặc định là Kano-Coin). Có các hàm cộng, trừ, và ném lỗi nếu số dư âm.
*   `WalletId`, `EscrowId`, `TransactionId`: Định danh.

### 2.3. Domain Services
*   `CommissionCalculatorService`: Chứa thuật toán tính toán hoa hồng của nền tảng khi một Booking hoàn thành (ví dụ: nền tảng thu 10% - 20% tùy thuộc vào hạng của Companion).
*   `CurrencyExchangeService`: (Nếu có) Chuyển đổi từ tiền VNĐ thực tế sang Kano-Coin khi Nạp tiền (Deposit) thông qua cổng thanh toán VNPay.

### 2.4. Repositories
*   `IWalletRepository`
*   `IEscrowRepository`

---

## 3. Bounded Context: PROFILE & CATALOGUE

**Trách nhiệm:** Quản lý hồ sơ Companion, danh mục dịch vụ (Scenario) và các tài sản Media.

### 3.1. Aggregates & Entities
*   **Aggregate Root: `CompanionProfile`**
    *   Chứa thông tin hiển thị của Companion.
*   **Aggregate Root: `Scenario`**
    *   Mô tả dịch vụ cung cấp.
*   **Aggregate Root: `MediaAsset`**
    *   Quản lý các file upload (Voice, Image).

### 3.2. Value Objects
*   `Location`: Tỉnh/Thành phố hoạt động.
*   `Money`: Giá của kịch bản.
*   `MediaUrl`: Chuỗi URL hợp lệ đến S3/Cloudinary.
*   `Duration`: Thời lượng kịch bản (được chuẩn hóa theo các block cố định: 60, 120, 180 phút).

### 3.3. Domain Services
*   `MediaValidationService`: Logic kiểm tra tính hợp lệ của File (VD: Voice intro không vượt quá 30s, dung lượng < 5MB).

### 3.4. Repositories
*   `ICompanionProfileRepository`
*   `IScenarioRepository`
*   `IMediaAssetRepository`

---

## 4. Bounded Context: INTERACTION

**Trách nhiệm:** Giao tiếp an toàn (Chat) và Đánh giá (Review).

### 4.1. Aggregates & Entities
*   **Aggregate Root: `ChatRoom`**
    *   Quản lý vòng đời và quyền truy cập vào phòng chat.
    *   *Entities con:* `ChatMessage` (Có thể lưu dạng Entity con nếu dùng Document DB như MongoDB, hoặc tách riêng nếu dùng RDBMS).
*   **Aggregate Root: `Review`**
    *   Quản lý đánh giá độc lập.

### 4.2. Value Objects
*   `Rating`: Chứa giá trị Integer từ 1 đến 5. Ném lỗi nếu giá trị nằm ngoài khoảng này.
*   `ChatContent`: Đóng gói text tin nhắn, có thể chứa logic filter từ ngữ nhạy cảm (profanity filter).

### 4.3. Domain Services
*   `ProfanityFilterService`: Chứa logic kiểm tra và làm mờ các từ ngữ vi phạm tiêu chuẩn cộng đồng hoặc chia sẻ thông tin liên lạc bên ngoài (Zalo, SĐT) trong tin nhắn chat.

### 4.4. Repositories
*   `IChatRoomRepository`
*   `IReviewRepository`

---

## 5. Bounded Context: IDENTITY

**Trách nhiệm:** Định danh, phân quyền và quản lý tài khoản.

### 5.1. Aggregates & Entities
*   **Aggregate Root: `UserAccount`**
    *   Lưu trữ thông tin xác thực.
    *   *Properties:* `AccountStatus` (ACTIVE, LOCKED), `ViolationCount` (Số lần vi phạm).

### 5.2. Value Objects
*   `Email`: Đóng gói chuỗi email, có logic regex kiểm tra tính hợp lệ.
*   `PasswordHash`: Chứa mã băm của mật khẩu, tuyệt đối không lưu plaintext.
*   `Role`: Enum (CLIENT, COMPANION, ADMIN).

### 5.3. Domain Services
*   `AccountLockPolicyService`: Chứa logic đánh giá xem tài khoản có nên bị khóa hay không dựa vào số lần vi phạm (ViolationCount) hoặc tính nghiêm trọng của lỗi.

### 5.4. Repositories
*   `IUserAccountRepository`

---

## 6. Bounded Context: DISPUTE RESOLUTION

**Trách nhiệm:** Hệ thống khiếu nại và phán quyết.

### 6.1. Aggregates & Entities
*   **Aggregate Root: `Dispute`**
    *   Đại diện cho một khiếu nại.
    *   *Entities con:* `DisputeEvidence` (Bằng chứng đính kèm như hình ảnh, text giải trình).

### 6.2. Value Objects
*   `DisputeReason`: Lý do khiếu nại (Enum hoặc Value Object định nghĩa sẵn).
*   `Resolution`: Quyết định cuối cùng (REFUND_CLIENT, PAYOUT_COMPANION, REJECT).

### 6.3. Domain Services
*   `DisputeRoutingService`: Chứa logic tự động phân bổ Dispute cho Admin đang online hoặc có ít task xử lý nhất.

### 6.4. Repositories
*   `IDisputeRepository`

---

## 7. Bounded Context: NOTIFICATION

**Trách nhiệm:** Phân phối thông điệp (Notification) tới người dùng.

### 7.1. Aggregates & Entities
*   **Aggregate Root: `Notification`**
    *   Quản lý một thông báo gửi đi và trạng thái Retry.

### 7.2. Value Objects
*   `NotificationChannel`: Enumeration/Value Object (SSE, FCM, EMAIL).
*   `NotificationContent`: Đóng gói tiêu đề (Title) và Nội dung (Body).

### 7.3. Domain Services
*   `NotificationFallbackService`: Logic quyết định chiến lược gửi thông báo (VD: Thử gửi qua SSE, nếu User offline (timeout) thì chuyển sang gửi qua FCM push notification).

### 7.4. Repositories
*   `INotificationRepository`
