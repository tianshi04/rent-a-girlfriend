# DESIGN-LEVEL EVENT STORMING

## 1. THIẾT KẾ AGGREGATES & INVARIANTS

Aggregate là ranh giới nhất quán của dữ liệu. Mọi Command đi vào hệ thống đều phải đi qua Aggregate Root để kiểm tra các Invariants (Quy tắc nghiệp vụ) trước khi thay đổi trạng thái (State) và phát ra Event.

### 1.1. Bounded Context: BOOKING
**Aggregate Root:** `Booking`
*   **Trách nhiệm:** Quản lý State Machine của cuộc hẹn và đảm bảo tính hợp lệ về mặt thời gian, trạng thái.
*   **State (Trạng thái lưu trữ):** `BookingId`, `ClientId`, `CompanionId`, `ScenarioSnapshot` (Giá, Thời lượng), `StartTime`, `EndTime`, `Status` (PENDING, ACCEPTED, REJECTED, COMPLETED, CANCELLED, DISPUTED, RESOLVED).
*   **Invariants (Quy tắc bất biến):**
    *   [INV-B01] `StartTime` phải lớn hơn thời gian hiện tại ít nhất 2 giờ.
    *   [INV-B02] `EndTime` phải bằng `StartTime` + `Scenario.Duration`.
    *   [INV-B03] Chỉ có thể `Accept` hoặc `Reject` khi Status đang là `PENDING`.
    *   [INV-B04] Companion không thể nhận quá 10 yêu cầu ở trạng thái `PENDING` từ tất cả các Client để tránh quá tải (Pending Cap Policy).
    *   [INV-B05] Không thể `Cancel` nếu Status đã là `COMPLETED` hoặc `CANCELLED` hoặc `RESOLVED`.
    *   [INV-B06] Client không được phép có hai booking chồng lặp (overlap) thời gian hoạt động (ở trạng thái `PENDING` hoặc `ACCEPTED`).
    *   [INV-B07] Companion không được phép chấp nhận (`Accept`) hai booking chồng lặp (overlap) thời gian hoạt động (ở trạng thái `ACCEPTED`).

**Thiết kế Command & Event:**
| Lệnh (Command) | Dữ liệu đầu vào (Payload Payload) | Sự kiện phát ra (Event Payload) |
| :--- | :--- | :--- |
| `RequestBooking` | `clientId`, `companionId`, `scenarioId`, `startTime`, `price` | `BookingRequested` { bookingId, clientId, companionId, price, startTime } |
| `AcceptBooking` | `bookingId`, `companionId` | `BookingAccepted` { bookingId, companionId, price } |
| `CancelBooking` | `bookingId`, `actorId`, `actorRole` | `BookingCancelledEarly` / `BookingCancelledLate` { bookingId, actorRole, isLate } |

---

### 1.2. Bounded Context: FINANCE
**Aggregate Root 1:** `Wallet`
*   **Trách nhiệm:** Quản lý số dư, đảm bảo không bao giờ âm tiền.
*   **State:** `WalletId`, `UserId`, `AvailableBalance`, `FrozenBalance`.
*   **Invariants:**
    *   [INV-F01] `AvailableBalance` không bao giờ được `< 0`.
    *   [INV-F02] Số tiền yêu cầu Freeze không được vượt quá `AvailableBalance`.
    *   [INV-F03] Khi Unfreeze, số tiền trả về không được lớn hơn `FrozenBalance`.

**Aggregate Root 2:** `Escrow`
*   **Trách nhiệm:** Quản lý quỹ đảm bảo cho từng Booking cụ thể.
*   **State:** `EscrowId`, `BookingId`, `Amount`, `Status` (HELD, REFUNDED, PAID_OUT).
*   **Invariants:**
    *   [INV-F04] Mỗi `BookingId` chỉ được có duy nhất 1 `Escrow` ở trạng thái `HELD`.
    *   [INV-F05] Chỉ được Payout hoặc Refund khi Status đang là `HELD`.

**Thiết kế Command & Event:**
| Lệnh (Command) | Dữ liệu đầu vào (Payload) | Sự kiện phát ra (Event Payload) |
| :--- | :--- | :--- |
| `FreezeCoin` | `walletId`, `amount`, `referenceId` (bookingId) | `CoinFrozen` { walletId, amount, refId } |
| `TransferToEscrow` | `walletId`, `bookingId`, `amount` | `CoinEscrowed` { bookingId, amount } |
| `ProcessPayout` | `escrowId`, `companionWalletId`, `commissionRate` | `PayoutProcessed` { bookingId, amount, commission } |

### 1.3. Bounded Context: INTERACTION
**Aggregate Root 1:** `ChatRoom`
*   **Trách nhiệm:** Quản lý quyền truy cập và trạng thái đóng/mở của phòng chat.
*   **State:** `RoomId`, `BookingId`, `ParticipantIds` [Client, Companion], `Status` (ACTIVE, LOCKED).
*   **Invariants:**
    *   [INV-I01] Chỉ những User có trong `ParticipantIds` mới được gửi/đọc tin nhắn.
    *   [INV-I02] Không thể gửi tin nhắn nếu Status là `LOCKED`.
    *   [INV-I03] Khi đã `LOCKED`, không thể chuyển ngược lại thành `ACTIVE`.

**Aggregate Root 2:** `Review`
*   **Trách nhiệm:** Quản lý tính duy nhất và tính hiển thị của đánh giá.
*   **State:** `ReviewId`, `BookingId`, `ClientId`, `CompanionId`, `Rating`, `Comment`, `IsVisible`.
*   **Invariants:**
    *   [INV-I04] Mỗi `BookingId` chỉ được tồn tại đúng 1 `Review`.
    *   [INV-I05] Rating phải nằm trong khoảng từ 1 đến 5.

### 1.4. Bounded Context: PROFILE & CATALOGUE
**Aggregate Root 1:** `Scenario` (Kịch bản dịch vụ)
*   **Trách nhiệm:** Quản lý thông tin gói dịch vụ mà Companion cung cấp, đảm bảo tính hợp lệ của giá cả và thời lượng.
*   **State:** `ScenarioId`, `CompanionId`, `Title`, `Description`, `Price`, `Duration` (phút), `Status` (ACTIVE, INACTIVE).
*   **Invariants (Quy tắc bất biến):**
    *   [INV-P01] `Price` (Giá) luôn phải lớn hơn `0`.
    *   [INV-P02] `Duration` phải nằm trong các mốc quy định của nền tảng (VD: 60, 120, 180 phút).
    *   [INV-P03] Một Companion chỉ được tạo tối đa `N` Scenarios (VD: 5 kịch bản) để tránh spam.

**Aggregate Root 2:** `MediaAsset` (Tài sản truyền thông)
*   **Trách nhiệm:** Quản lý file upload (đặc biệt là Voice Intro), kiểm soát dung lượng và định dạng.
*   **State:** `AssetId`, `CompanionId`, `AssetType` (IMAGE, VOICE), `Url`, `DurationSec`, `SizeBytes`, `Status` (PENDING, APPROVED, REJECTED).
*   **Invariants:**
    *   [INV-P04] Nếu `AssetType` là VOICE, `DurationSec` không được vượt quá 30 giây (BR-11).
    *   [INV-P05] Nếu `AssetType` là VOICE, `SizeBytes` không được vượt quá 5MB.

**Thiết kế Command & Event:**
| Lệnh (Command) | Dữ liệu đầu vào (Payload) | Sự kiện phát ra (Event Payload) |
| :--- | :--- | :--- |
| `CreateScenario` | `companionId`, `title`, `price`, `duration` | `ScenarioCreated` { scenarioId, companionId, price } |
| `UploadVoiceIntro` | `companionId`, `fileUrl`, `duration`, `size` | `VoiceIntroUploaded` / `VoiceIntroRejected` { reason } |

### 1.5. Bounded Context: IDENTITY
**Aggregate Root:** `UserAccount`
*   **Trách nhiệm:** Quản lý định danh, quyền hạn và chế tài xử phạt (khóa tài khoản).
*   **State:** `UserId`, `Email`, `Role` (CLIENT, COMPANION, ADMIN), `AccountStatus` (ACTIVE, LOCKED), `ViolationCount`.
*   **Invariants:**
    *   [INV-ID01] Không thể thực hiện hành động đăng nhập (`Login`) nếu `AccountStatus` đang là `LOCKED`.
    *   *(Ghi chú: Việc khóa tài khoản tự động được xử lý bởi AccountLockPolicy Service).*

**Thiết kế Command & Event:**
| Lệnh (Command) | Dữ liệu đầu vào (Payload) | Sự kiện phát ra (Event Payload) |
| :--- | :--- | :--- |
| `RecordViolation` | `userId`, `reason`, `bookingId` | `ViolationRecorded` { userId, currentCount } |
| `LockAccount` | `userId`, `reason`, `lockedBy` | `AccountLocked` { userId, reason } |

### 1.6. Bounded Context: DISPUTE RESOLUTION
**Aggregate Root:** `Dispute` (Khiếu nại / Tranh chấp)
*   **Trách nhiệm:** Quản lý vòng đời của một khiếu nại, đảm bảo Admin chỉ ra quyết định 1 lần duy nhất cho mỗi sự cố.
*   **State:** `DisputeId`, `BookingId`, `ReporterId`, `AccusedId`, `Reason`, `AdminId`, `Status` (OPEN, RESOLVING, REFUNDED, PAID_OUT, REJECTED).
*   **Invariants:**
    *   [INV-D01] Mỗi `BookingId` chỉ được phép tồn tại duy nhất 1 `Dispute` ở trạng thái `OPEN` hoặc `RESOLVING`.
    *   [INV-D02] Khi Status đã chuyển sang trạng thái quyết định cuối cùng (`REFUNDED`, `PAID_OUT`, `REJECTED`), tuyệt đối không được phép thay đổi lại (để tránh SAGA chạy lại luồng tiền tệ).
    *   [INV-D03] Chỉ Admin mới có quyền thực thi các lệnh Resolve.

**Thiết kế Command & Event:**
| Lệnh (Command) | Dữ liệu đầu vào (Payload) | Sự kiện phát ra (Event Payload) |
| :--- | :--- | :--- |
| `CreateReport` | `bookingId`, `reporterId`, `reason` | `ReportCreated` { disputeId, bookingId } |
| `ResolveDisputeRefund`| `disputeId`, `adminId`, `notes` | `DisputeResolvedRefund` { disputeId, bookingId } |

### 1.7. Bounded Context: NOTIFICATION
**Aggregate Root:** `Notification`
*   **Trách nhiệm:** Đảm bảo tin nhắn được gửi đi, quản lý chiến lược Retry (thử lại) nếu gửi lỗi.
*   **State:** `NotificationId`, `UserId`, `Channel` (SSE, FCM, EMAIL), `Content`, `Status` (PENDING, SENT, FAILED), `RetryCount`.
*   **Invariants:**
    *   [INV-N01] `RetryCount` không được vượt quá số lần cấu hình tối đa (VD: 3 lần). Nếu vượt quá, chuyển Status sang `FAILED`.
    *   [INV-N02] Không gửi lại thông báo nếu Status đã là `SENT`.

**Thiết kế Command & Event:**
| Lệnh (Command) | Dữ liệu đầu vào (Payload) | Sự kiện phát ra (Event Payload) |
| :--- | :--- | :--- |
| `SendNotification` | `userId`, `channel`, `content` | `NotificationSent` / `NotificationFailed` |

## 2. THIẾT KẾ SAGA STATE (PROCESS MANAGERS)

Dựa trên Bước 4.1, chúng ta có các Saga Orchestrator. Tại bước này, thiết kế cấu trúc dữ liệu để lưu trữ trạng thái của Saga (Saga State) nhằm phục vụ việc phục hồi khi hệ thống sập (Crash Recovery) và Idempotency.

### 2.1. Booking Accept Saga (Quản lý bởi Booking Context)
*   **Saga Entity:** `BookingAcceptSaga`
*   **Trường dữ liệu:**
    *   `SagaId` (UUID)
    *   `BookingId` (UUID)
    *   `CurrentState`: `WAITING_FOR_ESCROW` | `WAITING_FOR_CHAT` | `ACCEPTED` | `REVERTING_ESCROW` | `FAILED_TECHNICAL`
    *   `CreatedAt`, `UpdatedAt`
*   **Logic bù trừ (Compensation):**
    *   Nếu nhận `ChatRoomFailed` khi đang ở `WAITING_FOR_CHAT` -> Chuyển state sang `REVERTING_ESCROW` -> Gửi Command `RefundEscrowToFrozen` tới Finance Context.

### 2.2. Dispute Refund Saga (Quản lý bởi Dispute Context)
*   **Saga Entity:** `DisputeRefundSaga`
*   **Trường dữ liệu:**
    *   `SagaId` (UUID)
    *   `DisputeId` (UUID)
    *   `BookingId` (UUID)
    *   `CurrentState`: `REFUNDING` | `HIDING_REVIEW` | `DISPUTE_RESOLVED_REFUNDED` | `DISPUTE_FAILED`
    *   `CreatedAt`, `UpdatedAt`
*   **Logic xử lý / Bù trừ (Compensation):**
    *   Nếu nhận `RefundSuccess` -> Chuyển state sang `HIDING_REVIEW` -> Gửi Command `HideReviewAndLockChat` tới Interaction Context.
    *   Nếu nhận `RefundFailed` -> Chuyển state sang `DISPUTE_FAILED` (chờ Admin xử lý thủ công).
    *   *Đặc thù:* Không có logic Rollback hoàn tiền. Nếu Interaction lỗi khi đang `HIDING_REVIEW`, hệ thống sẽ sử dụng cơ chế Retry vô hạn thay vì bù trừ để đảm bảo tính toàn vẹn quyết định.

### 2.3. Dispute Payout Saga (Quản lý bởi Dispute Context)
*   **Saga Entity:** `DisputePayoutSaga`
*   **Trường dữ liệu:**
    *   `SagaId` (UUID)
    *   `DisputeId` (UUID)
    *   `BookingId` (UUID)
    *   `CurrentState`: `PAYING_OUT` | `LOCKING_CHAT` | `DISPUTE_RESOLVED_PAID_OUT` | `DISPUTE_FAILED`
    *   `CreatedAt`, `UpdatedAt`
*   **Logic xử lý / Bù trừ (Compensation):**
    *   Nếu nhận `PayoutSuccess` -> Chuyển state sang `LOCKING_CHAT` -> Gửi Command `LockChatRoom` tới Interaction Context.
    *   Nếu nhận `PayoutFailed` -> Chuyển state sang `DISPUTE_FAILED`.
    *   *Đặc thù:* Tương tự Refund Saga, hệ thống áp dụng Retry vô hạn cho bước Interaction thay vì Rollback tiền của Companion.

### 2.4. Quản lý trạng thái cho Saga Choreography
Các luồng như Hủy Booking (Client/Companion) hay Hoàn tất Booking được triển khai theo mô hình Choreography. Do không có Process Manager trung tâm, trạng thái và độ tin cậy được bảo đảm thông qua:
*   **Idempotency Store (Bảng theo dõi sự kiện):** Mỗi Context tham gia (Finance, Interaction, Profile) duy trì bảng `ProcessedEvents` (lưu `eventId` hoặc `sagaId`). Khi nhận được sự kiện (VD: `BookingCancelled`), Context sẽ kiểm tra ID; nếu đã xử lý, hệ thống bỏ qua logic nghiệp vụ và chỉ trả về ACK.
*   **Transactional Outbox:** Context phát sinh sự kiện (VD: Booking Context) cập nhật trạng thái Aggregate và ghi nội dung sự kiện vào bảng `Outbox` trong cùng một Database Transaction. Tiến trình ngầm sẽ đọc bảng Outbox và phát (publish) sang Broker, đảm bảo Eventual Consistency.

## 3. THIẾT KẾ READ MODELS (CQRS PROJECTIONS)

Trong thiết kế Microservices/DDD, các Aggregates ở trên (Write Model) được tối ưu cho việc ghi và bảo vệ logic. Tuy nhiên, UI cần hiển thị dữ liệu phức tạp (Join nhiều bảng). Do đó, chúng ta thiết kế các **Read Models** (được cập nhật thông qua việc lắng nghe Events).

### 3.1. Read Model: Companion Catalogue (Trang chủ tìm kiếm)
*   **Vị trí:** Profile Context (sử dụng Elasticsearch hoặc Redis/MongoDB).
*   **Nguồn cập nhật:** Lắng nghe các event `ProfileUpdated`, `ScenarioCreated`, `ReviewSubmitted` (từ Interaction).
*   **Cấu trúc dữ liệu (JSON Document):**
    ```json
    {
      "companionId": "cmp_123",
      "displayName": "Chizuru Mizuhara",
      "avatarUrl": "https://...",
      "averageRating": 4.9,
      "totalReviews": 150,
      "availableCities": ["Hanoi", "HCM"],
      "scenarios": [
         { "scenarioId": "scn_01", "title": "Hẹn hò rạp chiếu phim", "price": 500 }
      ],
      "voiceIntroUrl": "https://..."
    }
    ```
*   **Mục đích:** Cho phép Client tìm kiếm, lọc theo giá, thành phố, rating cực kỳ nhanh chóng mà không cần Join dữ liệu giữa Profile, Interaction và Booking.

### 3.2. Read Model: Booking History (Lịch sử cuộc hẹn của Client)
*   **Vị trí:** Booking Context.
*   **Nguồn cập nhật:** Lắng nghe `BookingRequested`, `BookingAccepted`, `ChatRoomCreated`, `PayoutProcessed`.
*   **Cấu trúc dữ liệu:**
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
*   **Mục đích:** Cung cấp API `GET /api/clients/me/bookings` cho Mobile/Web App hiển thị danh sách lịch sử mượt mà, gom toàn bộ thông tin cần thiết vào 1 object duy nhất.

### 3.3. Read Model: Admin Dashboard (Mô hình đọc cho Admin)
Để giao diện Admin có thể hiển thị danh sách các vấn đề cần xử lý mà không cần gọi API đến 4-5 service khác nhau, ta thiết kế một Read Model chuyên dụng cho Admin.

*   **Vị trí:** Dispute Resolution Context (hoặc một BFF - Backend for Frontend riêng cho Admin).
*   **Nguồn cập nhật:** Lắng nghe `ReportCreated`, `AccountLocked`, `DepositFailed` (từ Finance).
*   **Cấu trúc dữ liệu (JSON Document):**
    ```json
    {
      "actionRequiredId": "act_111",
      "type": "DISPUTE", // Có thể là DISPUTE, COMPANION_UPGRADE, DEPOSIT_CHECK
      "priority": "HIGH",
      "referenceId": "bk_999", // ID của booking hoặc user liên quan
      "summary": "Client khiếu nại Companion không đến",
      "createdAt": "2023-10-27T10:00:00Z",
      "status": "UNRESOLVED",
      "involvedParties": {
         "clientId": "cli_12",
         "companionId": "cmp_34"
      }
    }
    ```
*   **Mục đích:** Admin chỉ cần gọi 1 API duy nhất để lấy danh sách "To-do list" (Những việc cần xử lý gấp) trên toàn hệ thống, thay vì phải vào từng module tìm kiếm.

## 4. THIẾT KẾ CONTRACT CHO EVENT BUS (MESSAGE BROKER)

Để các Bounded Context giao tiếp an toàn, mọi Event đẩy lên Kafka/RabbitMQ phải tuân thủ một "Lớp vỏ" (Envelope) chuẩn.

**Cấu trúc CloudEvents Standard (Ví dụ cho `BookingAccepted`):**
```json
{
  "specversion": "1.0",
  "id": "evt_abc123", 
  "source": "/rent-a-gf/booking-context/booking/bk_999",
  "type": "booking.booking-accepted.v1",
  "datacontenttype": "application/json",
  "time": "2023-10-27T10:00:00Z",
  "correlationid": "req_xyz789", // Thuộc tính mở rộng ở root (lowercase) dùng để trace log toàn hệ thống
  "data": {
    "bookingId": "bk_999",
    "sagaId": "saga_555",
    "companionId": "cmp_123",
    "price": 500
  }
}
```
*   **`type`**: Có đánh version (`.v1`) để hỗ trợ nâng cấp cấu trúc payload sau này mà không làm gãy các Consumer cũ.
*   **`correlationid`**: Truyền ở cấp độ gốc (root) xuyên suốt từ API Gateway đến tất cả các Context để Debug trên Kibana/Datadog.
