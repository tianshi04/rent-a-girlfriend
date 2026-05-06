# BOOKING CONTEXT

**Phân loại Subdomain:** Core Subdomain

## 1. TRÁCH NHIỆM CHÍNH (RESPONSIBILITY)
Quản lý vòng đời (State Machine), trạng thái của cuộc hẹn và đảm bảo tính hợp lệ về mặt thời gian, quy tắc nghiệp vụ.

## 2. AGGREGATES & ENTITIES
### Aggregate Root: `Booking`
Đại diện cho một yêu cầu/cuộc hẹn. Mọi thao tác thay đổi trạng thái phải đi qua Aggregate này.
*   **State (Trạng thái lưu trữ):** `BookingId`, `ClientId`, `CompanionId`, `ScenarioSnapshot` (Giá, Thời lượng), `StartTime`, `EndTime`, `Status` (PENDING, ACCEPTED, REJECTED, COMPLETED, CANCELLED).

## 3. VALUE OBJECTS
*   `BookingId`: Định danh duy nhất (UUID).
*   `ClientId`, `CompanionId`: Tham chiếu đến Identity Context (UUID).
*   `TimeRange`: Đóng gói `StartTime` và `EndTime`, có logic kiểm tra thời gian bắt đầu phải trước thời gian kết thúc, và khoảng cách phải bằng đúng thời lượng kịch bản.
*   `ScenarioSnapshot`: Đóng gói `Price` (Money) và `Duration` tại thời điểm đặt lịch để đảm bảo tính bất biến dù Companion có đổi giá sau đó.

## 4. INVARIANTS (QUY TẮC BẤT BIẾN)
*   `[INV-B01]` `StartTime` phải lớn hơn thời gian hiện tại ít nhất 2 giờ.
*   `[INV-B02]` `EndTime` phải bằng `StartTime` + `Scenario.Duration`.
*   `[INV-B03]` Chỉ có thể `Accept` hoặc `Reject` khi Status đang là `PENDING`.
*   `[INV-B04]` Client không thể gửi quá 10 Request `PENDING` cho cùng 1 Companion (Pending Cap Policy).
*   `[INV-B05]` Không thể `Cancel` nếu Status đã là `COMPLETED` hoặc `CANCELLED`.

## 5. THIẾT KẾ COMMAND & EVENT
| Lệnh (Command) | Dữ liệu đầu vào (Payload) | Sự kiện phát ra (Event Payload) |
| :--- | :--- | :--- |
| `RequestBooking` | `clientId`, `companionId`, `scenarioId`, `startTime`, `price` | `BookingRequested` { bookingId, clientId, companionId, price, startTime } |
| `AcceptBooking` | `bookingId`, `companionId` | `BookingAccepted` { bookingId, companionId, price } |
| `CancelBooking` | `bookingId`, `actorId`, `actorRole` | `BookingCancelledEarly` / `BookingCancelledLate` { bookingId, actorRole, isLate } |

## 6. SAGA STATE (PROCESS MANAGERS)
*   **Saga Entity:** `BookingAcceptSaga`
*   **Trường dữ liệu:** `SagaId`, `BookingId`, `CurrentState` (`WAITING_FOR_ESCROW`, `WAITING_FOR_CHAT`, `ACCEPTED`, `REVERTING_ESCROW`, `FAILED_TECHNICAL`).
*   **Logic bù trừ (Compensation):** Nếu nhận `ChatRoomFailed` khi đang ở `WAITING_FOR_CHAT` -> Chuyển state sang `REVERTING_ESCROW` -> Gửi Command `RefundEscrowToFrozen` tới Finance Context.

## 7. READ MODELS (CQRS PROJECTIONS)
**Read Model: Booking History (Lịch sử cuộc hẹn của Client)**
*   **Nguồn cập nhật:** Lắng nghe `BookingRequested`, `BookingAccepted`, `ChatRoomCreated`, `PayoutProcessed`.
*   **Cấu trúc dữ liệu (JSON Document):**
    ```json
    {
      "bookingId": "bk_999",
      "status": "COMPLETED",
      "companionName": "Chizuru Mizuhara",
      "companionAvatar": "https://...",
      "scenarioTitle": "Hẹn hò rạp chiếu phim",
      "totalPrice": 500,
      "startTime": "2023-12-01T19:00:00Z",
      "chatRoomId": "chat_888",
      "hasReviewed": true
    }
    ```
*   **Mục đích:** Cung cấp API `GET /api/clients/me/bookings` cho UI hiển thị danh sách lịch sử mượt mà.

## 8. DOMAIN SERVICES
*   `BookingExpirationService`: Chứa logic kiểm tra xem một Booking có vượt quá thời gian chờ (Timeout) phản hồi hay không để tự động chuyển sang `CANCELLED`.
*   `BookingCompletionService`: Chứa logic nghiệp vụ tự động hoàn thành Booking khi đã qua thời gian `EndTime` + 12h mà không có khiếu nại.

## 9. REPOSITORIES
*   `IBookingRepository`: Lưu trữ và phục hồi trạng thái của `Booking` Aggregate.
