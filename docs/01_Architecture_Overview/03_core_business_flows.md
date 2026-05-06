# CÁC LUỒNG NGHIỆP VỤ CỐT LÕI (CORE BUSINESS FLOWS)

Tài liệu này mô tả các luồng hành trình người dùng (User Journey) chính của hệ thống Rent-a-Girlfriend, dựa trên kết quả phân tích sự kiện (Event Storming). Những luồng này định hình cách các đối tượng (Actors) tương tác với hệ thống và cách các sự kiện (Events) kích hoạt các chính sách (Policies) nghiệp vụ xuyên suốt các Microservices.

---

## 1. CÁC LUỒNG USER JOURNEY CHÍNH

### Flow 1: Onboarding & Cập nhật Profile
Luồng này xử lý quá trình người dùng mới tham gia và nâng cấp thành Companion.
- **Xác thực:** Người dùng đăng nhập lần đầu qua **Google OAuth** (Identity Context).
- **Onboarding:** Người dùng gửi yêu cầu nâng cấp thành Companion. Hệ thống áp dụng chính sách yêu cầu Admin kiểm duyệt thủ công.
- **Duyệt/Từ chối:** Admin xét duyệt yêu cầu.
- **Cập nhật Profile:** Sau khi được duyệt, Companion cập nhật Profile, tạo danh mục dịch vụ (Scenario) và upload hình ảnh/Voice Intro.
  - *Policy (Voice Intro):* Hệ thống tự động kiểm tra định dạng file (MP3, ≤30s, ≤5MB). Nếu không hợp lệ sẽ tự động từ chối. File được lưu trữ ở External Storage (S3/Cloudinary).

### Flow 2: Nạp tiền vào Ví
Luồng này đảm bảo giao dịch tài chính nội bộ.
- **Nạp tiền:** Client thực hiện giao dịch nạp tiền thông qua cổng thanh toán **VNPay**.
- **Đối soát:** Hệ thống sử dụng IPN callback từ VNPay làm *Source of Truth* duy nhất (nguồn sự thật).
  - ✅ IPN xác nhận thành công: Cộng Kano-Coin cho người dùng (Finance Context).
  - ❌ IPN thất bại hoặc Timeout: Giao dịch được đánh dấu "chờ đối soát" để Admin kiểm tra thủ công, tránh mất tiền của Client.

### Flow 3: Đánh giá & Xử lý Khiếu nại (Dispute Resolution)
Luồng này xử lý các tương tác sau khi lịch hẹn kết thúc và việc giải quyết xung đột nếu có.
- **Đánh giá:** Sau khi lịch hẹn kết thúc, Client có thể gửi Đánh giá (Review). Mỗi booking chỉ được đánh giá 1 lần duy nhất và không cho phép sửa/xóa.
- **Khiếu nại (Report):** Nếu có sự cố (No-show, thái độ tệ), Client hoặc Companion có thể tạo Report.
  - *Policy:* Hệ thống lập tức đóng băng quá trình tự động thanh toán (Freeze Escrow) cho đến khi Admin giải quyết xong.
- **Giải quyết Khiếu nại:** Admin can thiệp.
  - Lỗi do Companion: Refund tiền về cho Client. Review (nếu có) sẽ bị ẩn và phòng chat bị khóa.
  - Không có lỗi: Thực hiện Payout thanh toán tiền cho Companion, giữ nguyên trạng thái hiển thị của Review.

### Flow 4: Quản lý Tài khoản & Vi phạm
Luồng này duy trì trật tự và sự an toàn của nền tảng.
- **Tích lũy vi phạm:** Khi Companion vi phạm (vd: hủy lịch sát giờ, no-show), hệ thống sẽ cộng dồn số lần vi phạm.
- **Đạt ngưỡng vi phạm:** Nếu vi phạm đạt ngưỡng (mặc định 3 lần), hệ thống tự động khóa tài khoản tạm thời để Admin xem xét.
- **Mở khóa/Khóa vĩnh viễn:** Admin sẽ quyết định mở khóa lại hay khóa vĩnh viễn tài khoản sau khi xem xét chi tiết.

---

## 2. CHI TIẾT VÒNG ĐỜI ĐẶT LỊCH (BOOKING CORE LOOP)

Booking Core Loop là luồng nghiệp vụ phức tạp nhất và quan trọng nhất hệ thống, liên quan đến sự tương tác giữa Booking, Finance, và Interaction Context.

### Sơ đồ luồng xử lý (Process Flow Diagram)

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

### Các Chính sách (Policies) Đáng chú ý trong Vòng đời Booking:
1. **Pending Cap Policy:** Mỗi Companion chỉ được nhận tối đa 10 Booking Request đang chờ xử lý cùng lúc để tránh spam (Admin có thể cấu hình).
2. **Timeout Policy:** Nếu Companion không phản hồi trong vòng 12 tiếng hoặc trước `start_time` 1 tiếng, hệ thống tự động đánh dấu Timeout và hoàn trả (Unfreeze) tiền cho Client.
3. **Cancellation Refund Policy:** 
   - Client hủy sớm (>24h): Hoàn 100% tiền.
   - Client hủy muộn (<24h): Phạt 100% tiền, chuyển vào cho Companion bồi thường.
   - Companion hủy: Luôn hoàn 100% tiền cho Client. Nếu Companion hủy muộn (<24h) thì bị ghi nhận 1 lỗi vi phạm.
4. **Chat Lock Policy:** Phòng chat sẽ bị khóa **ngay lập tức** nếu Booking bị hủy/refund, hoặc tự động khóa **sau 24h** kể từ thời điểm Booking tự động Completed.
