# IMPLEMENTATION TIMELINE: RENT-A-GIRLFRIEND PLATFORM

Dựa trên tài liệu thiết kế hệ thống, lộ trình triển khai dự án được chia thành 6 giai đoạn (Milestones) chính, tập trung vào việc xây dựng nền tảng vững chắc trước khi đi sâu vào nghiệp vụ lõi.

---

## 📅 TỔNG QUAN LỘ TRÌNH

### Giai đoạn 1: Hạ tầng & Nền tảng
**Mục tiêu:** Thiết lập môi trường chạy Microservices và các thư viện dùng chung.
- [x] **Infrastructure Setup:** Cấu hình Kubernetes (Local: Minikube/K3s), Docker Registry.
- [x] **Service Mesh:** Cài đặt **Istio Ambient Mode**. Cấu hình `ztunnel` cho mTLS và `Waypoint Proxy` cho L7 traffic/Auth.
- [ ] **Shared Libraries:** Xây dựng package `common` (Standard Logging, Error Handling, Event Bus Wrapper).
- [ ] **Contract Definition:** Hoàn thiện Protobuf/OpenAPI schemas cho toàn bộ hệ thống tại `/contracts`.

### Giai đoạn 2: Identity & Notification
**Mục tiêu:** Xây dựng hệ thống định danh và thông báo làm nền tảng cho các service khác.
- [ ] **Identity Service (Generic):** 
    - Tích hợp Google OAuth.
    - Cung cấp JWKS endpoint cho Istio Waypoint xác thực tập trung.
    - Quản lý Role (Client/Companion/Admin) và Onboarding flow.
- [ ] **Notification Service (Generic):** 
    - Xây dựng hạ tầng **SSE (Server-Sent Events)** để đẩy thông báo real-time.
    - Implement consumer lắng nghe mọi Domain Events để gửi thông báo tương ứng.

### Giai đoạn 3: Profile & Catalogue
**Mục tiêu:** Cho phép Companion tạo hồ sơ và kịch bản (Scenario).
- [ ] **Profile Service (Core):** 
    - Implement Hexagonal Architecture cho Companion Profile.
    - Media Asset management: Tích hợp Presigned URL (S3/Cloudinary) cho Voice Intro và Ảnh.
    - Scenario CRUD: Cho phép thiết kế kịch bản và giá dịch vụ.
- [ ] **Search Engine:** Xây dựng Read-Model đơn giản để Client tìm kiếm Companion theo Thành phố/Giá.

### Giai đoạn 4: Core Loop - Booking & Finance
**Mục tiêu:** Triển khai luồng nghiệp vụ quan trọng nhất (Core Business Loop).
- [ ] **Finance Service (Core):** 
    - Quản lý Wallet (Kano-Coin), Ledger ghi nhận giao dịch.
    - Tích hợp cổng thanh toán **VNPay (IPN Webhook)** để nạp tiền.
    - Logic **Freeze/Escrow/Payout** cho Kano-Coin.
- [ ] **Booking Service (Core):** 
    - Xây dựng State Machine điều phối vòng đời Booking.
    - **SAGA Orchestrator:** Điều phối giao dịch liên service (ví dụ: `CreateBooking` -> `Finance:FreezeCoin`).
    - Logic Timeout: Tự động reject nếu Companion không phản hồi sau 12h.

### Giai đoạn 5: Interaction & Dispute Resolution
**Mục tiêu:** Hoàn thiện trải nghiệm sau khi kết nối và xử lý rủi ro.
- [ ] **Interaction Service (Supporting):** 
    - **Booking Chat:** Tạo phòng chat tự động sau khi Accept, khóa sau 24h.
    - **Review System:** Cho phép Client đánh giá 1 lần, không sửa/xóa.
- [ ] **Dispute Service (Supporting):** 
    - Tiếp nhận Report/No-show.
    - Admin Interface để phân xử (Payout/Refund).
    - **Complex SAGA:** Xử lý dòng tiền dựa trên phán quyết của Admin.

### Giai đoạn 6: Admin Dashboard & Tổng kết
**Mục tiêu:** Quản trị toàn diện và đảm bảo chất lượng.
- [ ] **Admin Dashboard (BFF/Frontend):**
    - Duyệt Companion Profile.
    - Giám sát dòng tiền Escrow và xử lý Dispute.
    - Quản lý User/Hệ thống.
- [ ] **End-to-End Testing:** Chạy kịch bản test toàn bộ Core Loop từ nạp tiền -> đặt lịch -> chat -> đánh giá.
- [ ] **Security Audit & Performance Tuning:** Kiểm tra gRPC/REST latency và Istio policies.
