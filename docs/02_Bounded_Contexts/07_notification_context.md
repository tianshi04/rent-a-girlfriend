# NOTIFICATION CONTEXT

**Phân loại Subdomain:** Generic Subdomain

## 1. TRÁCH NHIỆM CHÍNH (RESPONSIBILITY)
Đóng gói toàn bộ hạ tầng gửi tin, quản lý tập trung việc phân phối thông báo đa kênh (SSE, FCM, Email). Tách rời mối bận tâm về thông báo ra khỏi các service nghiệp vụ khác.

## 2. AGGREGATES & ENTITIES
### Aggregate Root: `Notification`
Đảm bảo thông điệp được gửi đi, theo dõi trạng thái gửi và quản lý chiến lược Retry (thử lại) nếu gửi lỗi.
*   **State:** `NotificationId`, `UserId`, `Channel` (SSE, FCM, EMAIL), `Content`, `Status` (PENDING, SENT, FAILED), `RetryCount`.

## 3. VALUE OBJECTS
*   `NotificationChannel`: Enumeration/Value Object (SSE, FCM, EMAIL).
*   `NotificationContent`: Đóng gói tiêu đề (Title) và Nội dung (Body).

## 4. INVARIANTS (QUY TẮC BẤT BIẾN)
*   `[INV-N01]` `RetryCount` không được vượt quá số lần cấu hình tối đa (VD: 3 lần). Nếu vượt quá, chuyển Status sang `FAILED`.
*   `[INV-N02]` Không gửi lại thông báo nếu Status đã là `SENT`.

## 5. THIẾT KẾ COMMAND & EVENT
| Lệnh (Command) | Dữ liệu đầu vào (Payload) | Sự kiện phát ra (Event Payload) |
| :--- | :--- | :--- |
| `SendNotification` | `userId`, `channel`, `content` | `NotificationSent` / `NotificationFailed` |

## 6. DOMAIN SERVICES
*   `NotificationFallbackService`: Logic quyết định chiến lược gửi thông báo (VD: Thử gửi qua kênh realtime là SSE trước, nếu User offline hoặc quá thời gian timeout thì chuyển sang gửi qua FCM push notification).

## 7. REPOSITORIES
*   `INotificationRepository`: Lưu trữ lịch sử thông báo đã được điều phối và gửi.
