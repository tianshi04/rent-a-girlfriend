# DISPUTE RESOLUTION CONTEXT

**Phân loại Subdomain:** Supporting Subdomain

## 1. TRÁCH NHIỆM CHÍNH (RESPONSIBILITY)
Hỗ trợ Admin tiếp nhận khiếu nại (Report/Dispute) từ người dùng, quản lý vòng đời khiếu nại và cung cấp công cụ phân xử để quyết định dòng tiền (Refund hoặc Payout).

## 2. AGGREGATES & ENTITIES
### Aggregate Root: `Dispute` (Khiếu nại / Tranh chấp)
Đại diện cho một khiếu nại, đảm bảo Admin chỉ ra quyết định 1 lần duy nhất cho mỗi sự cố.
*   **State:** `DisputeId`, `BookingId`, `ReporterId`, `AccusedId`, `Reason`, `AdminId`, `Status` (OPEN, RESOLVING, REFUNDED, PAID_OUT, REJECTED).
*   **Entities con:** `DisputeEvidence` (Bằng chứng đính kèm như hình ảnh, text giải trình).

## 3. VALUE OBJECTS
*   `DisputeReason`: Lý do khiếu nại (Enum hoặc Value Object định nghĩa sẵn).
*   `Resolution`: Quyết định cuối cùng (REFUND_CLIENT, PAYOUT_COMPANION, REJECT).

## 4. INVARIANTS (QUY TẮC BẤT BIẾN)
*   `[INV-D01]` Mỗi `BookingId` chỉ được phép tồn tại duy nhất 1 `Dispute` ở trạng thái `OPEN` hoặc `RESOLVING`.
*   `[INV-D02]` Khi Status đã chuyển sang trạng thái quyết định cuối cùng (`REFUNDED`, `PAID_OUT`, `REJECTED`), tuyệt đối không được phép thay đổi lại (để tránh SAGA chạy lại luồng tiền tệ).
*   `[INV-D03]` Chỉ Admin mới có quyền thực thi các lệnh Resolve (phán quyết).

## 5. THIẾT KẾ COMMAND & EVENT
| Lệnh (Command) | Dữ liệu đầu vào (Payload) | Sự kiện phát ra (Event Payload) |
| :--- | :--- | :--- |
| `CreateReport` | `bookingId`, `reporterId`, `accusedId`, `reason` | `ReportCreated` { disputeId, bookingId, reporterId, accusedId, reason, occurredAt } |
| `ResolveDispute` | `disputeId`, `adminId`, `resolution`, `notes` | `DisputeResolved` { disputeId, bookingId, resolution, occurredAt, resolvedBy, reporterId, accusedId } |

## 6. SAGA STATE (PROCESS MANAGERS)
**Dispute Refund Saga**
*   **Saga Entity:** `DisputeRefundSaga`
*   **Trường dữ liệu:** `SagaId`, `DisputeId`, `BookingId`, `CurrentState` (`REFUNDING`, `HIDING_REVIEW`, `DISPUTE_RESOLVED_REFUNDED`, `DISPUTE_FAILED`).
*   **Logic xử lý:**
    *   Nếu nhận `RefundSuccess` -> Chuyển state sang `HIDING_REVIEW` -> Gửi Command `HideReviewAndLockChat` tới Interaction Context.
    *   Nếu lỗi Interaction khi đang `HIDING_REVIEW`, hệ thống sẽ sử dụng cơ chế Retry vô hạn thay vì bù trừ để đảm bảo tính toàn vẹn quyết định. Không có logic Rollback hoàn tiền.

**Dispute Payout Saga**
*   **Saga Entity:** `DisputePayoutSaga`
*   **Trường dữ liệu:** `SagaId`, `DisputeId`, `BookingId`, `CurrentState` (`PAYING_OUT`, `LOCKING_CHAT`, `DISPUTE_RESOLVED_PAID_OUT`, `DISPUTE_FAILED`).
*   **Logic xử lý:**
    *   Nếu nhận `PayoutSuccess` -> Chuyển state sang `LOCKING_CHAT` -> Gửi Command `LockChatRoom` tới Interaction Context.
    *   Tương tự Refund Saga, áp dụng Retry vô hạn cho bước Interaction thay vì Rollback tiền của Companion.

## 7. READ MODELS (CQRS PROJECTIONS)
**Read Model: Admin Dashboard**
Để giao diện Admin có thể hiển thị danh sách các vấn đề cần xử lý trên toàn hệ thống từ 1 API duy nhất.
*   **Nguồn cập nhật:** Lắng nghe `ReportCreated`, `AccountLocked`, `DepositFailed`.
*   **Cấu trúc dữ liệu (JSON Document):**
    ```json
    {
      "actionRequiredId": "act_111",
      "type": "DISPUTE", 
      "priority": "HIGH",
      "referenceId": "bk_999", 
      "summary": "Client khiếu nại Companion không đến",
      "createdAt": "2023-10-27T10:00:00Z",
      "status": "UNRESOLVED",
      "involvedParties": {
         "clientId": "cli_12",
         "companionId": "cmp_34"
      }
    }
    ```

## 8. DOMAIN SERVICES
*   `DisputeRoutingService`: Chứa logic tự động phân bổ Dispute cho Admin đang online hoặc có ít task xử lý nhất.

## 9. REPOSITORIES
*   `IDisputeRepository`
