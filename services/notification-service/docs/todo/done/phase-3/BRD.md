# 🎯 BUSINESS REQUIREMENT DOCUMENT (BRD) - PHASE 3

## 1. Mục tiêu Kinh doanh
Tự động hóa luồng tương tác khách hàng của hệ thống **Rent-a-Girlfriend** bằng cách dịch chuyển từ cơ chế trigger thủ công sang cơ chế **Event-Driven Notification**. Đảm bảo thông tin chạm tới người dùng đúng lúc, đúng kênh, cá nhân hóa ngôn ngữ với chi phí vận hành tối ưu.

## 2. Đối tượng Phục vụ & Giá trị Mang lại
*   **Companion**: Nhận ngay yêu cầu đặt lịch (Booking), thu nhập mới (Payout), hoặc đánh giá từ khách hàng để phản hồi tức thì.
*   **Client**: Nhận thông tin xác nhận đặt lịch, cảnh báo giao dịch (Nạp tiền/Thất bại), tin nhắn chat và khiếu nại (Dispute).
*   **Hệ thống**: Tối ưu hóa chi phí bằng cách ưu tiên kênh SSE (miễn phí), tự động fallback FCM Push (chỉ khi offline), và Email cho các giao dịch cần lưu trữ (Hóa đơn).

## 3. Các Yêu cầu Nghiệp vụ Cốt lõi
1.  **Tự động hóa toàn trình (Zero-touch)**: Tự lắng nghe sự kiện từ các dịch vụ khác (Booking, Finance, Interaction, Profile, Identity, Dispute) để dịch thành thông báo tương ứng.
2.  **Thông minh & Tiết kiệm (Smart Routing)**: Quyết định kênh gửi (SSE/FCM/Email) dựa trên trạng thái kết nối realtime của User.
3.  **Cá nhân hóa Đa ngôn ngữ (i18n)**: Tự động chọn ngôn ngữ (vi/en) hiển thị dựa trên tùy chọn của User.
4.  **Bảo vệ Trải nghiệm (Idempotency Guard)**: Tuyệt đối không gửi trùng lặp một thông báo do retry ở tầng mạng.
5.  **Hỗ trợ Kênh Bị động (Passive Channel)**: Cho phép Admin hoặc các dịch vụ legacy gửi tin trực tiếp dạng đã format sẵn.

## 4. Tiêu chí Thành công Nghiệp vụ
*   **100%** sự kiện nghiệp vụ đăng ký được xử lý và lưu DB trong < 200ms.
*   **0%** thông báo bị trùng lặp hiển thị tới người dùng.
*   Tỷ lệ tiếp cận thông báo quan trọng (Transactional HIGH) đạt **> 99%**.
