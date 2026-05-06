# PHÂN CHIA SUBDOMAIN & BOUNDED CONTEXT

Dựa trên quá trình phân tích **Big Picture Event Storming**, các thành phần của hệ thống Rent-a-Girlfriend được phân loại thành các Subdomain dựa trên mức độ cốt lõi và giá trị chiến lược mang lại cho nền tảng.

Việc phân định này giúp định hướng kiến trúc Microservices và chiến lược phát triển (Tự xây dựng hay tích hợp giải pháp có sẵn).

---

## 1. CORE SUBDOMAIN
Đây là khu vực tạo ra lợi thế cạnh tranh lớn nhất và mang lại giá trị cốt lõi cho người dùng. Đội ngũ phát triển cần đầu tư tối đa nguồn lực vào việc tối ưu hóa và xây dựng các tính năng tại đây.

### 1.1. Booking Context
*   **Mô tả:** Trái tim nghiệp vụ của nền tảng, chịu trách nhiệm điều phối toàn bộ State Machine của lịch hẹn (từ lúc tạo yêu cầu, khóa coin, phản hồi, xử lý hủy lịch/vi phạm, đến khi hoàn thành).
*   **Aggregates chính:** `Booking`
*   **Tại sao là Core?** Trải nghiệm kết nối nhanh chóng, mượt mà và logic xử lý trạng thái chặt chẽ chính là điểm mấu chốt quyết định sự thành công của dịch vụ.

### 1.2. Profile & Catalogue Context
*   **Mô tả:** Nơi Companion quản lý hồ sơ cá nhân, kho media, và đặc biệt là kịch bản dịch vụ (Scenario).
*   **Aggregates chính:** `CompanionProfile`, `MediaAsset` (bao gồm Ảnh, Voice Intro), `Scenario`
*   **Tại sao là Core?** Khả năng tùy biến hồ sơ linh hoạt, kết hợp với các tính năng độc đáo như *Voice Intro* và *Scenario* (đóng vai theo kịch bản) là selling point (điểm bán hàng) khác biệt của nền tảng so với các dịch vụ thông thường.

### 1.3. Finance Context
*   **Mô tả:** Quản lý toàn bộ vòng đời của dòng tiền ảo (Kano-Coin), bao gồm: số dư ví, lịch sử giao dịch nạp tiền, đóng băng tiền (Escrow), đối soát thanh toán (Payout) và thu hoa hồng. Context này là chủ sở hữu (owner) của khái niệm Escrow.
*   **Aggregates chính:** `Wallet`, `Transaction`, `Escrow`
*   **Tại sao là Core?** Kano-Coin là đơn vị tiền tệ nội bộ, yêu cầu các logic nghiệp vụ đặc thù (Escrow, quy tắc hoa hồng, cơ chế phạt vi phạm). Giải pháp off-the-shelf không thể đáp ứng được. Trong khi VNPay chỉ đóng vai trò cổng nạp, lõi tài chính (wallet, escrow, payout) bắt buộc phải tự xây dựng để kiểm soát toàn bộ dòng tiền của hệ thống.
*   **Tích hợp bên ngoài:** Kết nối trực tiếp với cổng thanh toán **VNPay** để nạp tiền.

---

## 2. SUPPORTING SUBDOMAIN
Khu vực này hỗ trợ trực tiếp cho Core Subdomain. Chúng rất cần thiết để nền tảng hoạt động trơn tru, nhưng không phải là lợi thế cạnh tranh khác biệt. 

### 2.1. Interaction Context
*   **Mô tả:** Chịu trách nhiệm xử lý các tương tác sau khi kết nối thành công, bao gồm giao tiếp (Phòng chat) và Đánh giá (Review) dịch vụ.
*   **Aggregates chính:** `ChatRoom`, `Review`
*   **Tại sao là Supporting?** Bất kỳ nền tảng kết nối nào cũng cần có tính năng chat và đánh giá, đây là chức năng tiêu chuẩn. Hệ thống có thể tự xây dựng đơn giản để phục vụ đúng nhu cầu.

### 2.2. Dispute Resolution Context
*   **Mô tả:** Hỗ trợ Admin tiếp nhận khiếu nại (Report/Dispute) từ người dùng và phân xử để quyết định dòng tiền (Refund cho Client hoặc Payout cho Companion).
*   **Aggregates chính:** `Dispute`
*   **Tại sao là Supporting?** Là chức năng vận hành thiết yếu (operations) để duy trì sự công bằng và an toàn của hệ thống, nhưng không phải là tính năng thu hút người dùng ban đầu.

---

## 3. GENERIC SUBDOMAIN
Đây là các bài toán kỹ thuật chung, phổ biến trên rất nhiều hệ thống thương mại điện tử hoặc nền tảng trực tuyến. Các thành phần này không mang yếu tố đặc thù của dự án và hoàn toàn có thể (hoặc nên) tích hợp hệ thống bên thứ ba.

### 3.1. Identity Context
*   **Mô tả:** Xử lý xác thực đăng nhập, quản lý thông tin tài khoản cơ bản, vai trò người dùng (Client, Companion, Admin), và trạng thái khóa/mở khóa tài khoản.
*   **Aggregates chính:** `UserAccount`
*   **Tích hợp bên ngoài:** Sử dụng **Google OAuth** cho việc đăng nhập.

### 3.2. Notification Context
*   **Mô tả:** Quản lý tập trung toàn bộ hạ tầng và logic gửi thông báo đa kênh (SSE, FCM, Email). Đây là một cross-cutting concern đảm bảo các context khác không bị biến thành "distributed big ball of mud" khi mỗi nơi tự gọi API gửi thông báo.
*   **Aggregates chính:** `NotificationTemplate`, `Notification`
*   **Tại sao là Generic?** Gửi thông báo là bài toán kỹ thuật chung, không mang yếu tố nghiệp vụ đặc thù của hệ thống. Hệ thống có thể tận dụng hoặc tích hợp các dịch vụ gửi tin nhắn sẵn có (FCM, AWS SNS) thay vì phát triển lại toàn bộ hạ tầng.
