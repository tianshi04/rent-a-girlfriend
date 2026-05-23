# 📄 BUSINESS REQUIREMENTS DOCUMENT (BRD) - NOTIFICATION SERVICE

## 🛠️ TÀI LIỆU YÊU CẦU NGHIỆP VỤ (VERSION 1.0)
* **Dự án**: Hệ sinh thái dịch vụ hẹn hò Rent-a-Girlfriend
* **Thành phần**: Notification Service (Dịch vụ Thông báo)
* **Mục tiêu**: Xây dựng Delivery Hub tối ưu hóa trải nghiệm người dùng, đảm bảo tin cậy, bảo mật và khả năng phân phối thông tin đa kênh realtime.

---

## 1. GIỚI THIỆU & BỐI CẢNH DỰ ÁN (INTRODUCTION)

Hệ thống **Rent-a-Girlfriend** là nền tảng kết nối trực tuyến giữa **Client** (Khách hàng có nhu cầu thuê) và **Companion** (Người cung cấp dịch vụ hẹn hò/bạn gái thuê) thông qua các **Scenario** (Kịch bản hẹn hò) cụ thể. Toàn bộ giao dịch tài chính, thanh toán, ký quỹ dịch vụ được thực hiện bằng đồng coin ảo nội bộ **Kano-Coin** thông qua hệ thống **Escrow** (Ký quỹ bảo mật).

Trong một nền tảng tương tác thời gian thực đòi hỏi độ tin cậy tuyệt đối, **Notification Service** đóng vai trò là "Delivery Hub" trung tâm. Nó chịu trách nhiệm định tuyến, chuyển phát và lưu vết toàn bộ các thông điệp giao dịch, tương tác và khuyến mãi đến đúng đối tượng sử dụng trên các kênh liên lạc phù hợp nhất (SSE, Firebase Push, Email).

---

## 2. THUẬT NGỮ NGHIỆP VỤ THỐNG NHẤT (UBIQUITOUS LANGUAGE)

Để đảm bảo sự nhất quán từ tài liệu nghiệp vụ đến mã nguồn thực tế, toàn bộ hệ thống phải sử dụng thống nhất các thuật ngữ sau:

*   **Client (Khách hàng)**: Người dùng đăng ký tài khoản trên hệ thống, nạp **Kano-Coin** để thuê Companion.
*   **Companion (Bạn gái thuê)**: Người dùng cung cấp dịch vụ hẹn hò, thực hiện các kịch bản hẹn hò để nhận Kano-Coin.
*   **Kano-Coin**: Đồng tiền ảo chính thức dùng cho mọi giao dịch thanh toán trong hệ thống.
*   **Scenario (Kịch bản)**: Điều khoản, thỏa thuận thời gian, địa điểm và nội dung hẹn hò được hai bên thống nhất.
*   **Escrow (Ký quỹ)**: Cơ chế trung gian giữ **Kano-Coin** của Client khi đặt lịch, và chỉ giải ngân cho Companion sau khi cuộc hẹn hoàn tất thành công hoặc xử lý tranh chấp (Dispute) xong.
*   **Notification (Thông báo)**: Một yêu cầu truyền đạt thông tin cụ thể gửi tới một người dùng cụ thể.
*   **Delivery Attempt (Nỗ lực chuyển phát)**: Một lần tương tác thực tế với các nhà cung cấp dịch vụ viễn thông/mạng để truyền tải nội dung thông báo.

---

## 3. PHÂN LOẠI THÔNG BÁO (NOTIFICATION TAXONOMY)

Thông báo được phân thành 3 nhóm nghiệp vụ chính với chính sách lưu trữ và độ ưu tiên rõ rệt nhằm tối ưu hóa trải nghiệm người dùng (UX) và hiệu năng hệ thống:

| Nhóm thông báo | Mô tả & Ví dụ | Độ ưu tiên | Kênh Realtime (SSE) | Kênh Push (FCM) | Kênh Email | Cơ chế Lưu DB (Persistence) |
| :--- | :--- | :---: | :---: | :---: | :---: | :---: |
| **Transactional** (Giao dịch/Hệ thống) | **OTP, Đổi mật khẩu**, thay đổi bảo mật.<br>**Giao dịch Kano-Coin**: Biến động số dư tài khoản.<br>**Escrow & Booking**: Yêu cầu đặt lịch, Companion chấp nhận/hủy lịch, Escrow giải ngân hoặc xử lý Dispute. | **HIGH** | **Bắt buộc** | **Bắt buộc** (nếu Offline) | **Có** (khi có cờ yêu cầu) | **Có** (Lưu Inbox app & Nhật ký Audit bảo mật) |
| **Interaction** (Tương tác xã hội) | **Tin nhắn Chat mới** giữa Client và Companion.<br>**Nhận được Review/Đánh giá mới** sau khi kết thúc cuộc hẹn. | **MEDIUM** | **Bắt buộc** | **Có** (Gửi gom dạng **Batch** để tránh làm phiền) | **Không** | **Không** (Dữ liệu đã được lưu trữ tại Interaction/Chat Service) |
| **Promotional** (Khuyến mãi/Thông tin) | **Voucher/Khuyến mãi mới** được tặng.<br>**Thông báo bảo trì hệ thống**.<br>Nhắc nhở cập nhật thông tin Profile. | **LOW** | **Không** | **Không** (Trừ khi quản trị viên ép buộc) | **Không** | **Có** (Chỉ lưu Inbox) |

---

## 4. QUY TẮC ĐỊNH TUYẾN THÔNG BÁO (ROUTING POLICY)

Hệ thống hoạt động như một **Delivery Engine thông minh**, không tự suy diễn nghiệp vụ mà định tuyến hoàn toàn dựa trên trạng thái kết nối thời gian thực của người dùng và các tham số ép buộc (`policyOverrides`) từ Core Services gửi kèm:

### 4.1. Kênh Realtime (SSE - Server-Sent Events) First
*   Khi có thông báo mới, hệ thống luôn kiểm tra trạng thái kết nối SSE của người nhận (`userId`).
*   Nếu người dùng **Online** (đang mở App và giữ kết nối socket):
    *   Đẩy thông báo qua kết nối SSE.
    *   **Hủy luồng gửi Push Notification (FCM)** nhằm tiết kiệm chi phí băng thông Firebase và tránh tạo ra các tiếng chuông rung làm phiền người dùng khi họ đang hoạt động trên ứng dụng.
    *   *Ngoại lệ*: Nếu Core Service gửi kèm cờ `requirePush: true`, hệ thống sẽ đồng thời gửi qua cả hai kênh SSE và FCM Push.

### 4.2. Kênh Đánh thức (FCM Push Fallback)
*   Hệ thống gọi API của Google Firebase (FCM) để gửi thông báo đẩy đến thiết bị di động trong các trường hợp:
    *   Người dùng hoàn toàn **Offline** (không có kết nối SSE hoạt động).
    *   Kết nối SSE bị đứt hoặc lỗi kết nối mạng xảy ra đột ngột trong quá trình truyền dữ liệu (Broken pipe/Timeout).
    *   Có yêu cầu bắt buộc `requirePush: true` hoặc thông báo có độ ưu tiên là `HIGH` (như OTP, biến động tài khoản Kano-Coin).

### 4.3. Kênh Email (Độc lập)
*   Khi Core Service truyền cờ `requireEmail: true`, hệ thống sẽ gửi một bản sao thông báo qua kênh Email (sử dụng SMTP/SendGrid).
*   Luồng gửi Email chạy độc lập và song song, không ảnh hưởng đến quyết định định tuyến của SSE hay FCM Push.

---

## 5. VÒNG ĐỜI TRẠNG THÁI & LỖI (STATE MACHINE & RETRY)

Để đảm bảo tính nhất quán và khả năng phục hồi dữ liệu khi mạng chập chờn, mọi nỗ lực gửi thông báo phải được lưu vết rõ ràng.

### 5.1. Sơ đồ trạng thái của Yêu cầu gửi (`Notification`)
*   **`PENDING`**: Trạng thái mới nhận từ Message Queue, đã ghi nhận thông tin xuống DB nhưng chưa xử lý định tuyến.
*   **`PROCESSING`**: Đang thực hiện định tuyến và đợi phản hồi từ hạ tầng mạng hoặc bên thứ 3 (Firebase, Mail Service).
*   **`COMPLETED`**: (Trạng thái cuối cùng - Thành công) Ít nhất một kênh truyền thông hợp lệ đã phân phối thông tin tới người dùng thành công.
*   **`FAILED`**: (Trạng thái cuối cùng - Thất bại) Quá giới hạn thử lại cho phép hoặc gặp lỗi hệ thống nghiêm trọng không thể phục hồi.

### 5.2. Sơ đồ trạng thái của Nỗ lực chuyển phát (`DeliveryAttempt`)
*   **`PENDING`**: Nỗ lực đang được thực thi (đang ghi dữ liệu vào socket SSE hoặc đang gọi HTTP POST tới Firebase API).
*   **`SUCCESS`**: Phân phối thành công tới hạ tầng (Nhận HTTP 200 từ FCM, ghi thành công TCP socket SSE).
*   **`FAILED_RECOVERABLE`**: Thất bại do lỗi kỹ thuật tạm thời (Timeout mạng, HTTP 503 từ Firebase, v.v.). **Cho phép thử lại (Retry)**.
*   **`FAILED_UNRECOVERABLE`**: Thất bại do lỗi logic/dữ liệu không hợp lệ (Sai Device Token, tài khoản bị unsubscribe, Payload quá kích thước cho phép). **CẤM thử lại và chốt thất bại lập tức**.

---

## 6. CÁC QUY TẮC BẤT BIẾN NGHIỆP VỤ (BUSINESS INVARIANTS)

Lớp lõi nghiệp vụ (Domain Layer) của Notification Service có trách nhiệm bảo vệ tuyệt đối các ràng buộc cốt lõi sau:

*   **`[INV-N01] - Giới hạn Thử lại tối đa (Max Retries Limit)`**:
    *   Hệ thống chỉ được phép thử lại (Retry) tối đa **3 lần** đối với các lỗi kỹ thuật tạm thời (`FAILED_RECOVERABLE`).
    *   Khi số lần thử thất bại đạt đến 3, hệ thống buộc phải dừng mọi hoạt động và chuyển trạng thái tổng thể của thông báo sang `FAILED`.
    *   Cơ chế thử lại phải áp dụng giải thuật **Exponential Backoff** (chờ 2s, 4s, 8s...) để tránh làm quá tải các hệ thống đối tác.
*   **`[INV-N02] - Bảo vệ trạng thái Terminal (No Modifications After Completed)`**:
    *   Một khi thông báo đã đạt trạng thái thành công (`COMPLETED`), hệ thống **TUYỆT ĐỐI CẤM** tạo thêm bất kỳ nỗ lực gửi (`DeliveryAttempt`) nào mới hoặc thay đổi trạng thái của thông báo đó. Quy tắc này ngăn chặn các lỗi bất đồng bộ gây gửi lặp tin nhắn tới người dùng.
*   **`[INV-N03] - Tính Idempotency`**:
    *   Hệ thống cấm tạo ra hai thông báo khác nhau cho cùng một `userId` xuất phát từ cùng một `idempotencyKey` của hệ thống.
    *   Ràng buộc này được bảo vệ nghiêm ngặt ở tầng Persistence bằng một chỉ mục độc bản (Unique Constraint) ghép cặp `(user_id, idempotency_key)` trong cơ sở dữ liệu.
*   **`[INV-N04] - Đảm bảo dọn dẹp kết nối chết (Dead Connection Cleanup)`**:
    *   Hệ thống bắt buộc phải tự động phát hiện kết nối đứt ở tầng mạng TCP (Broken pipe, Client offline không ngắt kết nối sạch) và dọn dẹp đối tượng kết nối SSE (`SseEmitter`) khỏi `SseConnectionRegistry` trong vòng tối đa **5 giây** sau khi phát hiện lỗi I/O để tránh rò rỉ bộ nhớ (Memory Leak).
*   **`[INV-N05] - Giới hạn tần suất Heartbeat (Heartbeat Frequency)`**:
    *   Để ngăn chặn việc Load Balancer (AWS ALB, Nginx, Istio Waypoint Proxy) ngắt kết nối HTTP streaming do trạng thái rảnh rỗi (Idle Connection), hệ thống bắt buộc phải duy trì việc gửi gói tin Heartbeat (`: ping\n\n`) định kỳ với tần suất cố định là **15 giây một lần** cho mỗi kết nối active.


---

## 7. YÊU CẦU PHI CHỨC NĂNG (NON-FUNCTIONAL REQUIREMENTS)

*   **Khả năng chịu tải & Bất đồng bộ**: Luồng xử lý gửi tin của Notification Service phải tận dụng tối đa kiến trúc bất đồng bộ chạy trên **Java 21 Virtual Threads** nhằm giải phóng tài nguyên I/O blocking khi gọi API Firebase/Email, đảm bảo tốc độ phản hồi nhanh ngay cả dưới tải cực lớn.
*   **Tính an toàn & Bảo mật**:
    *   Thông tin xác thực cơ sở dữ liệu (Neon Cloud Postgres) tuyệt đối không được ghi cứng (hardcode) trong mã nguồn, bắt buộc phải nạp qua biến môi trường (Environment Variables) khi ứng dụng khởi chạy.
    *   Đảm bảo mTLS và xác thực danh tính dịch vụ (SPIFFE) thông qua Service Mesh Istio Ambient Mode (ztunnel/Waypoint), không tự xử lý xác thực JWT thủ công bên trong code của Service.
*   **Khả năng giám sát (Observability)**:
    *   Tách biệt lịch sử nỗ lực gửi (`DeliveryAttempt`) ra bảng riêng giúp quản trị viên dễ dàng truy vết nguyên nhân lỗi gửi tin (ví dụ: truy vấn tỉ lệ lỗi Timeout của Firebase hoặc tỉ lệ token chết của Client).
