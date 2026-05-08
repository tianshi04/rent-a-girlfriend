# IDENTITY CONTEXT

**Phân loại Subdomain:** Generic Subdomain

## 1. TRÁCH NHIỆM CHÍNH (RESPONSIBILITY)
Xử lý xác thực người dùng (Google OAuth), cấp phát và quản lý vòng đời Token (Access/Refresh Token), phân quyền (RBAC), kiểm soát quy trình Onboarding, và quản lý chế tài xử phạt (khóa tài khoản).

## 2. AGGREGATES & ENTITIES
### Aggregate Root: `UserAccount`
Lưu trữ thông tin xác thực và trạng thái người dùng.
*   **State:** `UserId`, `Email`, `Role` (CLIENT, COMPANION, ADMIN), `AccountStatus` (ACTIVE, LOCKED), `ViolationCount` (Số lần vi phạm).

## 3. VALUE OBJECTS
*   `Email`: Đóng gói chuỗi email, có logic regex kiểm tra tính hợp lệ.
*   `PasswordHash`: Chứa mã băm của mật khẩu, tuyệt đối không lưu plaintext.
*   `Role`: Enum quyền người dùng (CLIENT, COMPANION, ADMIN).

## 4. INVARIANTS (QUY TẮC BẤT BIẾN)
*   `[INV-ID01]` Không thể thực hiện hành động đăng nhập (`Login`) nếu `AccountStatus` đang là `LOCKED`.
*   *(Ghi chú: Việc tự động khóa tài khoản khi đạt ngưỡng vi phạm được xử lý thông qua AccountLockPolicyService để tăng tính linh hoạt).*

## 5. THIẾT KẾ COMMAND & EVENT
| Lệnh (Command) | Dữ liệu đầu vào (Payload) | Sự kiện phát ra (Event Payload) |
| :--- | :--- | :--- |
| `RecordViolation` | `userId`, `reason`, `bookingId` | `ViolationRecorded` { userId, currentCount } |
| `LockAccount` | `userId`, `reason`, `lockedBy` | `AccountLocked` { userId, reason } |

## 6. DOMAIN SERVICES
*   `AccountLockPolicyService`: Chứa logic đánh giá xem tài khoản có nên bị khóa hay không dựa vào số lần vi phạm (ViolationCount) hoặc tính nghiêm trọng của lỗi, từ đó tự động LockAccount.

## 7. REPOSITORIES
*   `IUserAccountRepository`: Quản lý lưu trữ trạng thái người dùng.

## 8. CƠ CHẾ XÁC THỰC (AUTHENTICATION MECHANISM)

Để đảm bảo tính độc lập và bảo mật, Identity Service áp dụng các cơ chế sau:

*   **Token Pair:** Sau khi login qua Google, hệ thống cấp cặp Access Token (ngắn hạn) và Refresh Token (dài hạn).
*   **Refresh Token Rotation:** Mỗi lần refresh token sẽ sinh ra một refresh token mới, vô hiệu hóa token cũ để chống tấn công replay.
*   **Xác thực Phi tập trung (JWKS) qua Service Mesh:** Identity Service cung cấp Public Keys qua endpoint `GET /.well-known/jwks.json`.
    *   Hệ thống không yêu cầu từng Microservice tự verify Token. Thay vào đó, **Istio Service Mesh (Waypoint Proxy)** sẽ sử dụng Public Keys này để tự động xác thực JWT (Token-based Auth) ngay tại tầng hạ tầng.
    *   Request chỉ được chuyển đến service nếu Token hợp lệ, giúp giảm tải logic cho các service nghiệp vụ và tăng tính bảo mật đồng nhất.
    *   Hỗ trợ Key Rotation (xoay khóa) định kỳ với Key ID (`kid`).

*Chi tiết về thuật toán, cấu trúc payload và logic xoay vòng xem tại: [Auth Design Details](../reference/auth_implementation_details.md).*
