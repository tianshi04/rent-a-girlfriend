# MỨC XỬ LÝ (PROCESSING LEVEL EVENT STORMING)

## Rent-a-Girlfriend — Nền tảng Kết nối Dịch vụ Đồng hành Theo Kịch bản

---

## 1. GIỚI THIỆU
Ở mức **Processing Level**, chúng ta chuyển từ góc nhìn "bức tranh tổng thể" (Big Picture) sang việc phân tích chi tiết "làm thế nào hệ thống hoạt động". 
Trọng tâm của bước này là xác định chuỗi nhân quả: **[Actor]** gọi **(Command)** tác động lên **<Aggregate>** sinh ra **\*Event\***, từ đó kích hoạt các **{Policy}** để dẫn đến các **(Command)** tiếp theo.

**Quy ước ký hiệu (Legend):**
*   `[Actor]`: Người dùng hoặc Hệ thống bên ngoài.
*   `(Command)`: Hành động/Lệnh yêu cầu hệ thống thay đổi trạng thái (Động từ nguyên thể).
*   `<Aggregate>`: Thực thể chịu trách nhiệm xử lý logic và bảo vệ toàn vẹn dữ liệu.
*   `*Event*`: Sự kiện đã xảy ra (Động từ thì quá khứ).
*   `{Policy}`: Chính sách/Quy tắc phản ứng lại Sự kiện.
*   `[[External System]]`: Hệ thống bên ngoài.

---

## 2. PROCESS FLOW DIAGRAM (ASCII DIAGRAM)

Dưới đây là sơ đồ luồng xử lý cho Core Flow quan trọng nhất: **Vòng đời Đặt lịch (Booking Core Loop)** kết hợp với **Finance** và **Interaction**.

```text
[Client]
   |
   | 1. (RequestBooking)
   v
<Booking Aggregate> ===(Sync gRPC)===> <Wallet Aggregate> (Kiểm tra số dư & Freeze)
   |                                          |
   | 2. *BookingRequested*                    | 2a. *CoinFrozen*
   v                                          v
{Notification Policy}                  {Pending Cap Policy}
   |                                          |
   | 3. (SendSSEToCompanion)                  | (Kiểm tra < 10 pending requests)
   v
[Companion]
   |
   | 4. (AcceptBooking)
   v
<Booking Aggregate>
   |
   | 5. *BookingAccepted*
   +-------------------------------------------------------------+
   |                                                             |
   v                                                             v
{Escrow Policy}                                           {Chat Creation Policy}
   |                                                             |
   | 6. (TransferToEscrow)                                       | 7. (CreateChatRoom)
   v                                                             v
<Escrow Aggregate> [Finance Context]                      <ChatRoom Aggregate> [Interaction Context]
   |                                                             |
   | 8. *CoinEscrowed*                                           | 9. *ChatRoomCreated*
   v                                                             v
{Notification Policy}                                            |
   |                                                             |
   | 10. (SendSSEToClient)                                       |
   v                                                             |
[Client] <====================(Chatting)=========================> [Companion]
   |
   | 11. Thời gian đạt (end_time + 12h) & Không có khiếu nại
   v
{Auto-Complete Policy}
   |
   | 12. (CompleteBooking)
   v
<Booking Aggregate>
   |
   | 13. *BookingCompleted*
   +-------------------------------------------------------------+
   |                                                             |
   v                                                             v
{Payout Policy} [Finance Context]                         {Chat Lock Policy} [Interaction Context]
   |                                                             |
   | 14. (ProcessPayout)                                         | 15. (LockChatRoom) sau 24h
   v                                                             v
<Wallet Aggregate> (Trừ hoa hồng -> Cộng ví Companion)    <ChatRoom Aggregate>
   |                                                             |
   | 16. *PayoutProcessed* & *CommissionCollected*               | 17. *ChatRoomLocked*
```

---

## 3. DANH SÁCH COMMAND (THEO BOUNDED CONTEXT)

Danh sách các Command làm thay đổi trạng thái của hệ thống, được nhóm theo Bounded Context.

### 3.1. Identity Context
| Command | Actor | Aggregate Target | Event sinh ra |
| :--- | :--- | :--- | :--- |
| `LoginWithGoogle` | User | `<UserAccount>` | *UserLoggedIn* |
| `RequestCompanionUpgrade`| User | `<UserAccount>` | *UpgradeRequested* |
| `ApproveUpgrade` / `RejectUpgrade` | Admin | `<UserAccount>` | *UpgradeApproved* / *UpgradeRejected* |
| `LockAccount` / `UnlockAccount` | Admin / System | `<UserAccount>` | *AccountLocked* / *AccountUnlocked* |

### 3.2. Profile & Catalogue Context
| Command | Actor | Aggregate Target | Event sinh ra |
| :--- | :--- | :--- | :--- |
| `UpdateProfile` | Companion | `<CompanionProfile>`| *ProfileUpdated* |
| `UploadVoiceIntro` | Companion | `<MediaAsset>` | *VoiceIntroUploaded* / *VoiceIntroRejected* |
| `CreateScenario` / `UpdateScenario`| Companion | `<Scenario>` | *ScenarioCreated* / *ScenarioUpdated* |

### 3.3. Booking Context
| Command | Actor | Aggregate Target | Event sinh ra |
| :--- | :--- | :--- | :--- |
| `RequestBooking` | Client | `<Booking>` | *BookingRequested* |
| `AcceptBooking` | Companion | `<Booking>` | *BookingAccepted* |
| `RejectBooking` | Companion | `<Booking>` | *BookingRejected* |
| `CancelBooking` | Client/Companion | `<Booking>` | *BookingCancelledEarly* / *BookingCancelledLate* |
| `CompleteBooking` | System (Cron) | `<Booking>` | *BookingCompleted* |
| `ExpireBooking` | System (Cron) | `<Booking>` | *BookingTimedOut* |

### 3.4. Finance Context
| Command | Actor | Aggregate Target | Event sinh ra |
| :--- | :--- | :--- | :--- |
| `InitiateDeposit` | Client | `<Transaction>` | *DepositInitiated* |
| `ProcessVNPayIPN` | VNPay | `<Transaction>` | *KanoCoinDeposited* / *DepositFailed* |
| `FreezeCoin` | Booking Context | `<Wallet>` | *CoinFrozen* / *CoinFreezeFailed* |
| `UnfreezeCoin` | Booking/Dispute | `<Wallet>` | *CoinUnfrozen* |
| `TransferToEscrow` | Booking Context | `<Escrow>` | *CoinEscrowed* |
| `ProcessPayout` | Booking/Dispute | `<Wallet>` | *PayoutProcessed*, *CommissionCollected* |

### 3.5. Interaction Context
| Command | Actor | Aggregate Target | Event sinh ra |
| :--- | :--- | :--- | :--- |
| `CreateChatRoom` | Booking Context | `<ChatRoom>` | *ChatRoomCreated* |
| `LockChatRoom` | System/Dispute | `<ChatRoom>` | *ChatRoomLocked* |
| `SubmitReview` | Client | `<Review>` | *ReviewSubmitted* |
| `HideReview` | Dispute Context | `<Review>` | *ReviewHidden* |

### 3.6. Dispute Resolution Context
| Command | Actor | Aggregate Target | Event sinh ra |
| :--- | :--- | :--- | :--- |
| `CreateReport` | Client/Companion | `<Dispute>` | *ReportCreated* |
| `ResolveDisputeRefund` | Admin | `<Dispute>` | *DisputeResolvedRefund* |
| `ResolveDisputePayout` | Admin | `<Dispute>` | *DisputeResolvedPayout* |
| `RecordViolation` | System/Admin | `<Dispute>` | *ViolationRecorded* |

---

## 4. POLICY MAP (BẢN ĐỒ CHÍNH SÁCH)

Policy Map thể hiện cách hệ thống phản ứng (Reactive) khi một Event xảy ra. Đây là cơ sở để thiết kế Event Listeners / Message Brokers (Kafka/RabbitMQ).

| Trigger Event (Sự kiện xảy ra) | Policy (Chính sách áp dụng) | Target Command (Lệnh được kích hoạt) | Target Context |
| :--- | :--- | :--- | :--- |
| *BookingRequested* | **Timeout Policy**: Nếu sau 12h hoặc sát giờ hẹn 1h không ai phản hồi. | `ExpireBooking` | Booking |
| *BookingRequested* | **Notification Policy**: Gửi thông báo cho Companion. | `SendNotification` | Notification |
| *BookingAccepted* | **Escrow Policy**: Chuyển tiền đã Freeze sang Escrow. | `TransferToEscrow` | Finance |
| *BookingAccepted* | **Chat Creation Policy**: Tạo phòng chat cho 2 bên. | `CreateChatRoom` | Interaction |
| *BookingRejected* / *BookingTimedOut* | **Refund Policy**: Trả lại tiền cọc cho Client. | `UnfreezeCoin` | Finance |
| *BookingCancelledEarly* (Client) | **Refund Policy**: Trả lại 100% tiền cho Client. | `UnfreezeCoin` | Finance |
| *BookingCancelledLate* (Client) | **Penalty Policy**: Chuyển tiền cho Companion. | `ProcessPayout` | Finance |
| *BookingCancelledLate* (Companion) | **Violation Policy**: Ghi nhận vi phạm cho Companion. | `RecordViolation` | Dispute |
| *BookingCompleted* | **Payout Policy**: Trừ hoa hồng và thanh toán cho Companion. | `ProcessPayout` | Finance |
| *ReportCreated* | **Freeze Escrow Policy**: Chặn tiến trình Auto-complete. | `HaltBookingCompletion` | Booking |
| *DisputeResolvedRefund* | **Review Visibility Policy**: Ẩn đánh giá nếu lỗi do Companion. | `HideReview` | Interaction |
| *ViolationRecorded* | **Threshold Policy**: Nếu vi phạm >= 3 lần. | `LockAccount` | Identity |
| *(Any Chat/Booking Event)* | **Chat Lock Policy**: Khóa chat khi Cancel/Refund hoặc sau Complete 24h. | `LockChatRoom` | Interaction |

---

## 5. PHÂN TÍCH SYNC/ASYNC & INTEGRATION POINTS (ĐIỂM TÍCH HỢP)

Để đảm bảo hiệu năng, tính nhất quán dữ liệu và giảm thiểu độ trễ (latency), giao tiếp giữa các Bounded Context được thiết kế kết hợp giữa **Synchronous (Đồng bộ)** và **Asynchronous (Bất đồng bộ)**.

### 5.1. Synchronous Integration (Đồng bộ - REST API / gRPC)
Sử dụng khi một Context cần dữ liệu hoặc cần đảm bảo tính nhất quán tức thời (Strong Consistency) trước khi thực hiện hành động tiếp theo.

1. **Booking Context → Profile Context (Get Scenario Snapshot)**
   * *Mục đích:* Khi Client bấm `RequestBooking`, Booking Context phải gọi đồng bộ sang Profile Context để lấy đúng giá tiền và thông tin Scenario tại thời điểm đó (tránh việc Companion vừa đổi giá xong).
   * *Giao thức đề xuất:* gRPC (nhanh, nội bộ) hoặc REST.
2. **Booking Context → Finance Context (Check Balance & Freeze)**
   * *Mục đích:* Không thể tạo Booking Request nếu ví không đủ tiền. Do đó, Booking Context gọi đồng bộ sang Finance Context để kiểm tra số dư và thực hiện lệnh `FreezeCoin`. Nếu thành công mới lưu Booking vào DB.
   * *Giao thức đề xuất:* gRPC.
3. **API Gateway → Identity Context (Token Validation)**
   * *Mục đích:* Xác thực mọi request từ Client/Companion trước khi route đến các Context nghiệp vụ.

### 5.2. Asynchronous Integration (Bất đồng bộ - Event-Driven / Pub-Sub)
Sử dụng Event Broker (Kafka, RabbitMQ, hoặc AWS SNS/SQS) cho các tác vụ không yêu cầu phản hồi ngay lập tức (Eventual Consistency), giúp hệ thống chịu tải tốt và lỏng lẻo (loosely coupled).

1. **Event: `BookingAccepted` (Nguồn: Booking Context)**
   * **Finance Context** (Subscriber): Lắng nghe để chạy lệnh `TransferToEscrow`.
   * **Interaction Context** (Subscriber): Lắng nghe để chạy lệnh `CreateChatRoom`.
   * **Notification Context** (Subscriber): Lắng nghe để push SSE cho Client.
2. **Event: `BookingCompleted` (Nguồn: Booking Context)**
   * **Finance Context** (Subscriber): Lắng nghe để tính toán hoa hồng và chạy `ProcessPayout`.
   * **Notification Context** (Subscriber): Lắng nghe để nhắc Client viết Review.
3. **Event: `DisputeResolvedRefund` (Nguồn: Dispute Context)**
   * **Finance Context** (Subscriber): Lắng nghe để trả tiền từ Escrow về lại Wallet cho Client.
   * **Interaction Context** (Subscriber): Lắng nghe để chạy `HideReview` và `LockChatRoom`.
4. **Event: `ViolationRecorded` (Nguồn: Dispute Context)**
   * **Identity Context** (Subscriber): Lắng nghe, cộng dồn biến đếm. Nếu đạt ngưỡng (>=3), tự động kích hoạt `LockAccount`.

### 5.3. External Integration (Tích hợp hệ thống ngoài)
1. **VNPay (Payment Gateway):**
   * *Mô hình:* Asynchronous Webhook (IPN).
   * *Luồng:* Client thanh toán trên web VNPay -> VNPay gọi IPN (Webhook) về **Finance Context** -> Finance Context chạy `ProcessVNPayIPN` -> Cộng Kano-Coin.
2. **Google OAuth (Identity Provider):**
   * *Mô hình:* Synchronous Redirect & Callback.
3. **Storage Service (AWS S3 / Cloudinary):**
   * *Mô hình:* Synchronous Upload từ Client/Companion (thường thông qua Presigned URL để giảm tải cho server), sau đó lưu URL vào **Profile Context**.