# 📦 MÔ HÌNH LĨNH VỰC (DOMAIN MODEL)

Tài liệu này định nghĩa cấu trúc dữ liệu cốt lõi (Domain Model) của Notification Service theo chuẩn **Domain-Driven Design (DDD)**. Mô hình được thiết kế theo hướng **Production-minded**, đảm bảo khả năng truy vết (traceability), phục vụ cơ chế Retry và hỗ trợ tối đa cho việc debug lỗi mạng.

---

## 1. TẠI SAO PHẢI TÁCH BIỆT THỰC THỂ?

Trong thực tế, một thông báo có thể phải trải qua **nhiều lần gửi thất bại** (do lỗi kết nối mạng, Firebase sập, User mất mạng) trước khi tới đích thành công, hoặc có thể gửi qua **nhiều kênh** khác nhau. 

Nếu gộp chung tất cả vào một bảng duy nhất, chúng ta sẽ mất đi lịch sử (lỗi do đâu, lỗi lúc nào). Vì vậy, hệ thống bóc tách thành 2 thực thể có quan hệ **1-N**:
- **`Notification`**: Đại diện cho *Thông tin cần truyền đạt* (Gửi cái gì? Cho ai?).
- **`DeliveryAttempt`**: Đại diện cho *Nỗ lực chuyển phát* (Gửi qua đâu? Gửi lúc nào? Thành công hay Thất bại?).

---

## 2. AGGREGATE ROOT: `Notification`

`Notification` là điểm vào (Entry Point) của mọi thao tác. Không một service nào được phép sửa trực tiếp `DeliveryAttempt` mà phải đi qua `Notification`.

### 2.1. Cấu trúc Thuộc tính (Attributes)

| Thuộc tính | Kiểu dữ liệu | Mô tả |
| :--- | :--- | :--- |
| `id` | UUID | Định danh duy nhất của thông báo. |
| `userId` | UUID | Người nhận thông báo. |
| `idempotencyKey` | String | Khóa idempotency (từ Event gốc hoặc API Request) để đảm bảo Idempotency. |
| `type` | String/Enum | `TRANSACTIONAL`, `INTERACTION`, `PROMOTIONAL`. |
| `priority` | String/Enum | `HIGH`, `MEDIUM`, `LOW`. |
| `payload` | JSONB | Chứa `title`, `body` và các `metadata` động (VD: link ảnh, deep link). Lợi thế của PostgreSQL JSONB. |
| `status` | String/Enum | `PENDING`, `PROCESSING`, `COMPLETED`, `FAILED`. |
| `createdAt` | Timestamp | Thời điểm tạo thông báo. |
| `updatedAt` | Timestamp | Thời điểm cập nhật trạng thái cuối cùng. |
| `attempts` | Array | Mảng chứa các `DeliveryAttempt` (Trong code Application, không phải DB schema). |

### 2.2. Hành vi (Behaviors / Methods)

- `createAttempt(channel: Channel): DeliveryAttempt`: Khởi tạo một nỗ lực gửi mới.
- `markAttemptSuccess(attemptId: UUID, messageId: String)`: Đánh dấu một nỗ lực thành công. Cập nhật `status` của Notification thành `COMPLETED`.
- `markAttemptFailed(attemptId: UUID, errorCode: String, errorMessage: String, recoverable: boolean)`: Đánh dấu nỗ lực thất bại (Phân loại lỗi Recoverable vs Unrecoverable).
- `hasExceededRetryLimit(): boolean`: Kiểm tra xem đã vượt quá số lần Retry tối đa (3 lần) chưa. Nếu rồi, chuyển trạng thái Notification sang `FAILED`.

---

## 3. ENTITY: `DeliveryAttempt`

Lưu vết từng lần giao tiếp với bên thứ 3 (Firebase, SendGrid) hoặc mạng nội bộ (SSE).

### 3.1. Cấu trúc Thuộc tính (Attributes)

| Thuộc tính | Kiểu dữ liệu | Mô tả |
| :--- | :--- | :--- |
| `id` | UUID | Định danh của nỗ lực gửi. |
| `notificationId` | UUID | Khóa ngoại chỉ tới `Notification`. |
| `channel` | String/Enum | `SSE`, `FCM`, `EMAIL`. |
| `status` | String/Enum | `PENDING`, `SUCCESS`, `FAILED_RECOVERABLE`, `FAILED_UNRECOVERABLE`. |
| `errorCode` | String | (Tùy chọn) Mã lỗi kỹ thuật cụ thể (Ví dụ: "FCM_TOKEN_INVALID"). |
| `errorMessage` | Text | (Tùy chọn) Lý do lỗi chi tiết. Phục vụ debug. |
| `attemptedAt` | Timestamp | Thời điểm bắt đầu gửi. |
| `resolvedAt` | Timestamp | Thời điểm có kết quả (thành công hoặc lỗi). |

---

## 4. QUY TẮC BẤT BIẾN (INVARIANTS)

Đây là các quy tắc kinh doanh (Business Rules) bắt buộc lớp code Domain phải bảo vệ, không bao giờ được vi phạm:

- **`[INV-N01] - Giới hạn Retry`**: Tổng số lượng `DeliveryAttempt` có trạng thái `FAILED` không được vượt quá **3 lần**. Vượt quá số này, `Notification` buộc phải chốt status là `FAILED` và dừng mọi xử lý.
- **`[INV-N02] - Cấm ghi đè trạng thái`**: Nếu `Notification` đã chuyển sang trạng thái `COMPLETED` (đã có 1 `DeliveryAttempt` gửi thành công), hệ thống **CẤM** tạo thêm bất kỳ `DeliveryAttempt` nào mới cho nó.
- **`[INV-N03] - Tính Idempotency`**: Hệ thống cấm tạo ra 2 `Notification` có cùng `idempotencyKey` và `userId` (bảo vệ ở mức DB bằng Unique Constraint composite).

> [!TIP]
> Việc thiết kế trên PostgreSQL cho phép chúng ta dễ dàng thực hiện câu SQL: *`SELECT * FROM DeliveryAttempt WHERE status = 'FAILED' AND errorMessage LIKE '%Timeout%'`* để phát hiện xem hệ thống mạng của mình (hay của Firebase) đang gặp sự cố. Đây là giá trị cốt lõi của việc tách Entity!
