# IDENTITY CONTEXT

**Phân loại Subdomain:** Generic Subdomain

## 1. TRÁCH NHIỆM CHÍNH (RESPONSIBILITY)
Xử lý xác thực người dùng, quản lý định danh, phân quyền, kiểm soát quy trình xét duyệt Onboarding, và quản lý chế tài xử phạt (khóa tài khoản).

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
