# 📝 BUSINESS REQUIREMENTS DOCUMENT (BRD) - PHASE 5: OUTBOUND PROVIDERS & RETRY EXECUTION

## 1. Tổng quan & Mục tiêu Nghiệp vụ (Business Goal)

Mục tiêu cốt lõi của **Phase 5** là hoàn thiện khả năng gửi thông báo thực tế tới các kênh ngoại vi bên ngoài và đảm bảo tính tin cậy tuyệt đối (Reliability) của hệ thống thông báo **Rent-A-Girlfriend**. 

Hệ thống cần giải quyết các bài toán nghiệp vụ sau:
1. **Gửi tin đa kênh thực tế**: Hiện thực hóa việc gửi thông báo qua **FCM Push Notification** ( Firebase Cloud Messaging cho thiết bị di động của Client/Companion) và **SMTP Email** (cho các thông tin giao dịch, sao kê, điều khoản dịch vụ).
2. **Đảm bảo giao hàng tin cậy (Delivery Guarantee - Retry)**: Khi các nhà cung cấp dịch vụ bên ngoài gặp lỗi tạm thời (ví dụ: mất kết nối SMTP, Firebase quá tải), hệ thống phải tự động thử lại (Retry) thông minh mà không làm ảnh hưởng đến hiệu năng của luồng chính.
3. **Phòng vệ sự cố thời gian thực (SSE Fallback)**: Trong trường hợp hệ thống phát hiện người dùng online và định tuyến gửi qua **SSE (Server-Sent Events)**, nhưng quá trình gửi thực tế bị thất bại (do mạng chập chờn, client tắt app đột ngột), hệ thống phải tự động chuyển kênh **Fallback sang FCM Push** ngay lập tức để thông tin quan trọng không bị thất lạc.

---

## 2. Các Ràng buộc & Quy tắc Nghiệp vụ (Business Invariants & Rules)

### [INV-N01]: Giới hạn Retry tối đa 3 lần thất bại (Recoverable)
- Đối với các lỗi có tính chất tạm thời và có thể khôi phục được (**Recoverable**), hệ thống chỉ được phép thử lại tối đa **3 lần** (tổng cộng tối đa 4 lần gửi bao gồm cả lần đầu).
- Nếu sau 3 lần thử lại vẫn thất bại, trạng thái của thông báo (`Notification`) bắt buộc phải chuyển sang **`FAILED`** để bộ phận vận hành có thể tra cứu và xử lý thủ công.

### [INV-N02]: Cấm tạo attempt mới sau khi Notification đã hoàn thành
- Khi một thông báo đã được gửi thành công qua bất kỳ kênh nào và trạng thái chuyển sang **`COMPLETED`**, nghiêm cấm mọi hành vi tạo thêm nỗ lực gửi (`DeliveryAttempt`) mới cho thông báo đó.

### [INV-N09]: Tự động chuyển kênh phòng vệ (SSE -> FCM Fallback)
- Khi nỗ lực gửi thông báo thời gian thực qua kênh SSE thất bại (trả về kết quả lỗi hoặc không thể phân phối xuống client), và thông báo đó có cấu hình kênh FCM trong `policyOverrides`, hệ thống phải **tự động và ngay lập tức** kích hoạt luồng gửi thay thế qua kênh **FCM Push**.

### [INV-N10]: Chính sách Thử lại Trì hoãn Không Block Luồng (Non-blocking Exponential Backoff)
- Cơ chế retry bắt buộc phải thực hiện bất đồng bộ và có khoảng thời gian chờ (delay) tăng dần theo chính sách **Exponential Backoff** để tránh gây quá tải (DDoS ngược) cho các Provider bên ngoài.
- Khoảng thời gian backoff được định nghĩa cụ thể cho từng kênh:
  - **FCM Push**: Thử lại sau **2s** (lần 1) -> **4s** (lần 2) -> **8s** (lần 3).
  - **SMTP Email**: Thử lại sau **5s** (lần 1) -> **15s** (lần 2) -> **45s** (lần 3).
- Tuyệt đối **không được block thread xử lý chính** (không sử dụng Thread.sleep trên các luồng xử lý sự kiện).

---

## 3. Phân loại Kết quả & Tính chất Lỗi (Failure Classification)

Nghiệp vụ yêu cầu phân loại lỗi cực kỳ rõ ràng để tối ưu hóa tài nguyên hệ thống, tránh thử lại vô ích đối với các lỗi vĩnh viễn:

1. **Lỗi khôi phục được (Recoverable Error)**:
   - Là các lỗi do môi trường, hạ tầng tạm thời bị gián đoạn (ví dụ: Timeout kết nối SMTP, Gateway Timeout từ Firebase API, Rate Limit tạm thời).
   - *Hành động*: Ghi nhận trạng thái attempt là `FAILED_RECOVERABLE`, giữ trạng thái Notification là `PROCESSING` và lên lịch retry trì hoãn.
2. **Lỗi KHÔNG khôi phục được (Unrecoverable Error)**:
   - Là các lỗi do cấu hình sai, sai thông tin nghiệp vụ vĩnh viễn (ví dụ: Token thiết bị FCM không hợp lệ/hết hạn, Địa chỉ Email sai định dạng, Sai tài khoản gửi SMTP).
   - *Hành động*: Ghi nhận trạng thái attempt là `FAILED_UNRECOVERABLE`, chuyển trạng thái Notification sang `FAILED` ngay lập tức và **dừng toàn bộ quá trình retry**.

---

## 4. Danh sách Use Cases Nghiệp vụ

### Use Case 1: Gửi thông báo qua FCM Push (Mock & Real)
### Use Case 1: Gửi thông báo qua FCM Push (Mock & Real)
- **Actor**: System (Async Worker).
- **Luồng nghiệp vụ**: Hệ thống bốc thông tin token FCM của người nhận, đóng gói tiêu đề và nội dung tin nhắn, gửi đến Firebase Messaging API thực tế sử dụng Service Account Credentials sẵn có (`notification-service-d166c-firebase-adminsdk-fbsvc-4b6bc988c1.json`). Ghi nhận kết quả gửi thành công/thất bại chính xác.

### Use Case 2: Gửi thông báo qua SMTP Email (Mailtrap Sandbox & Real)
- **Actor**: System (Async Worker).
- **Luồng nghiệp vụ**: Hệ thống bốc địa chỉ Email của người nhận từ payload, tạo cấu trúc thư điện tử chuẩn, kết nối tới Mailtrap SMTP Sandbox (gói miễn phí cực kỳ tối ưu cho testing & demo) để gửi tin thật trực quan.

### Use Case 3: SSE Failover (Tự động chuyển kênh phòng vệ)
- **Actor**: System (Async Worker).
- **Luồng nghiệp vụ**: Khi gửi tin SSE thất bại, hệ thống tự động kiểm tra cấu hình, nếu có kênh FCM thì tạo ngay một sự kiện gửi Push FCM bất đồng bộ thay thế cho phiên SSE bị hỏng.

### Use Case 4: Lên lịch Retry trì hoãn bất đồng bộ
- **Actor**: System (Scheduler).
- **Luồng nghiệp vụ**: Khi ghi nhận lỗi gửi Recoverable, hệ thống tự động tính toán thời gian trễ dựa theo số lần thất bại trước đó, lên lịch chạy lại bất đồng bộ. Khi đến giờ hẹn, hệ thống tự động thực thi lại luồng gửi tin.
