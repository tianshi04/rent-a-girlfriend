# QUÁ TRÌNH EVENT STORMING

## Rent-a-Girlfriend — Nền tảng Kết nối Dịch vụ Đồng hành Theo Kịch bản

---

## 1. GIỚI THIỆU

### 1.1. Mục tiêu và Phạm vi
Tài liệu này ghi nhận kết quả của quá trình **Big Picture Event Storming**.
Mục tiêu là khám phá các sự kiện nghiệp vụ (Domain Events) cốt lõi và hình dung tổng thể luồng hoạt động của nền tảng, làm cơ sở để phân định các Bounded Context.

**Phạm vi áp dụng:**
- Tập trung vào **Big Picture Event Storming**, xác định các Domain Event mang ý nghĩa nghiệp vụ cốt lõi để cung cấp góc nhìn tổng thể về hệ thống.

### 1.2. Quy ước màu sắc (Sticky Notes)

| Màu | Ý nghĩa | Mô tả |
|:---:|:---|:---|
| 🟧 **Cam** | Domain Event | Sự kiện nghiệp vụ cốt lõi đã xảy ra (viết ở thì quá khứ) |
| 🟨 **Vàng** | Actor | Người dùng thực hiện tác động |
| 🟪 **Tím** | Policy | Quy tắc / Chính sách nghiệp vụ của hệ thống |
| 🩷 **Hồng** | External System | Hệ thống bên ngoài tương tác với luồng nghiệp vụ |
| 🔴 **Đỏ** | Hot Spot | Điểm chưa rõ, rủi ro hoặc vấn đề cần thảo luận thêm |

---

## 2. KHÁM PHÁ DOMAIN EVENTS (CHAOTIC EXPLORATION)

Danh sách các sự kiện nghiệp vụ cốt lõi, đã được chắt lọc để phục vụ góc nhìn tổng thể:

**Nhóm Xác thực & Onboarding:**
1. User đã đăng nhập lần đầu (Google OAuth)
2. Yêu cầu nâng cấp Companion đã được gửi
3. Yêu cầu nâng cấp Companion đã được duyệt
4. Yêu cầu nâng cấp Companion bị từ chối
5. Tài khoản đã bị khóa
6. Tài khoản đã được mở khóa

**Nhóm Quản lý Profile:**
7. Profile Companion đã được cập nhật
8. Voice Intro đã được tải lên
9. Voice Intro bị từ chối bởi System (không hợp lệ — sai format/vượt giới hạn)
10. Kịch bản dịch vụ (Scenario) đã được tạo/cập nhật

**Nhóm Đặt lịch (Booking Core):**
11. Yêu cầu đặt lịch (Booking Request) đã được gửi
12. Coin đã bị Freeze (khi gửi Booking Request)
13. Coin bị từ chối Freeze (số dư không đủ)
14. Yêu cầu đặt lịch đã được chấp nhận (Accepted)
15. Coin đã được chuyển vào Escrow (khi Companion Accept)
16. Yêu cầu đặt lịch bị từ chối (Rejected)
17. Coin đã được Unfreeze (khi Reject/Timeout)
18. Yêu cầu đặt lịch bị hết hạn (Timeout)
19. Lịch hẹn bị hủy sớm bởi Client (>24h — hoàn 100%)
20. Lịch hẹn bị hủy muộn bởi Client (<24h — mất 100%)
21. Lịch hẹn bị hủy sớm bởi Companion (>24h — không phạt)
22. Lịch hẹn bị hủy muộn bởi Companion (<24h — ghi vi phạm)
23. Cuộc hẹn đã được tự động hoàn thành

**Nhóm Tương tác (Chat & Review):**
24. Phòng chat đã được tạo
25. Phòng chat đã bị khóa
26. Đánh giá (Review) đã được gửi
27. Đánh giá đã bị ẩn (do xử lý khiếu nại)

**Nhóm Tài chính & Khiếu nại:**
28. Nạp tiền Kano-Coin thành công
29. Nạp tiền Kano-Coin thất bại
30. Hoa hồng nền tảng đã được thu
31. Thanh toán cho Companion (Payout) đã được thực hiện
32. Khiếu nại (Report/Dispute) đã được tạo
33. Khiếu nại đã được giải quyết (Refund cho Client)
34. Khiếu nại đã được giải quyết (Payout cho Companion)
35. Vi phạm của Companion đã được ghi nhận
36. Companion đã đạt ngưỡng vi phạm

**Nhóm Thông báo:**
37. Thông báo đã được gửi (cho các sự kiện booking)

---

## 3. LUỒNG THỜI GIAN (TIMELINE ORDERING) & POLICIES

Sắp xếp các Domain Event theo trình tự thời gian kết hợp với các Actor, External System và Policy.

### Flow 1: Onboarding & Cập nhật Profile
- **[Actor: User]** → `User đã đăng nhập lần đầu` qua **[External: Google OAuth]**
- **[Actor: User]** → `Yêu cầu nâng cấp Companion đã được gửi`
- **[Policy]** Hệ thống yêu cầu Admin kiểm duyệt thủ công.
- **[Actor: Admin]** → `Yêu cầu nâng cấp Companion đã được duyệt` (hoặc `bị từ chối`)
- **[Actor: Companion]** → `Profile Companion đã được cập nhật` (Cập nhật Scenario, upload ảnh/Voice Intro qua **[External: Storage/File Service]**)
  - **[Policy: Voice Intro Validation]** System tự động kiểm tra file upload (MP3, ≤30s, ≤5MB — BR-11). Nếu không hợp lệ → `Voice Intro bị từ chối bởi System`.
  - **[Policy: Voice Intro Access]** Bất kỳ người dùng nào (kể cả chưa đăng nhập) đều có quyền nghe Voice Intro; không cho phép tải xuống (BR-11).

### Flow 2: Nạp tiền vào Ví
- **[Actor: Client]** thực hiện giao dịch qua **[External: VNPay]**
- **[Policy: VNPay IPN]** Hệ thống lấy IPN callback từ VNPay làm nguồn sự thật duy nhất (source of truth). Chỉ cộng coin khi IPN xác nhận thành công.
  - ✅ IPN xác nhận thành công → `Nạp tiền Kano-Coin thành công`.
  - ❌ IPN xác nhận thất bại hoặc không nhận được IPN → `Nạp tiền Kano-Coin thất bại`. Giao dịch được đánh dấu trạng thái "chờ đối soát" để Admin tra cứu thủ công nếu Client khiếu nại.

### Flow 3: Vòng đời Đặt lịch (Booking Core Loop)
- **[Actor: Client]** → `Yêu cầu đặt lịch đã được gửi`
  - **[Policy: Pending Cap]** Mỗi Companion chỉ được tồn tại tối đa **10 Booking Request** ở trạng thái PENDING cùng lúc. Nếu đạt giới hạn, Client sẽ không thể gửi request mới cho Companion đó cho đến khi có slot trống. *(Giá trị 10 là mặc định, Admin có thể cấu hình lại.)*
  - **[Policy: Freeze coin]** Hệ thống kiểm tra số dư và freeze coin tương ứng phí Scenario ngay lập tức.
    - ✅ Đủ số dư → `Coin đã bị Freeze`.
    - ❌ Không đủ số dư → `Coin bị từ chối Freeze` → Booking Request bị hủy, kết thúc luồng.
- **[Policy: Timeout]** Booking chờ Companion phản hồi tối đa 12h, hoặc đến trước giờ `start_time` - 1 tiếng (tùy điều kiện nào đến trước). Nếu quá hạn: → `Yêu cầu đặt lịch bị hết hạn` → `Coin đã được Unfreeze`.
- **[Actor: Companion]** phản hồi:
  - `Yêu cầu đặt lịch bị từ chối` → `Coin đã được Unfreeze` → Kết thúc luồng.
  - `Yêu cầu đặt lịch đã được chấp nhận`
    - **[Policy: Escrow]** Khi Accept thành công → `Coin đã được chuyển vào Escrow`.
    - **[Policy: Chat Creation]** Khi Accept thành công → `Phòng chat đã được tạo`.
- Thay đổi lịch trình từ hai bên (nếu có):
  - **[Actor: Client]** → `Lịch hẹn bị hủy sớm bởi Client` hoặc `Lịch hẹn bị hủy muộn bởi Client`.
    - **[Policy: Refund — Client Cancel]** Hủy sớm (>24h) → Hoàn 100% coin cho Client. Hủy muộn (<24h) hoặc No-show → Client mất 100% coin, khoản này được chuyển cho Companion như bồi thường (BR-06).
    - **[Policy: Chat Lock on Cancel]** Khi booking bị hủy → `Phòng chat đã bị khóa` ngay lập tức (BR-15).
  - **[Actor: Companion]** → `Lịch hẹn bị hủy sớm bởi Companion` hoặc `Lịch hẹn bị hủy muộn bởi Companion`.
    - **[Policy: Refund — Companion Cancel]** Companion hủy (bất kể sớm hay muộn) → luôn hoàn 100% coin cho Client → `Coin đã được Unfreeze`.
    - **[Policy: Vi phạm]** Nếu Companion hủy muộn (<24h) hoặc No-show → `Vi phạm của Companion đã được ghi nhận` *(xảy ra song song với hoàn coin)*.
    - **[Policy: Chat Lock on Cancel]** Khi booking bị hủy → `Phòng chat đã bị khóa` ngay lập tức (BR-15).
- **[Policy: Auto-complete]** Khi thời gian đạt `end_time` + 12h và không có Report nào xảy ra: → `Cuộc hẹn đã được tự động hoàn thành`.
  - **[Policy: Hoa hồng]** Hệ thống khấu trừ % hoa hồng trước khi payout → `Hoa hồng nền tảng đã được thu` và `Thanh toán cho Companion đã được thực hiện`.
  - **[Policy: Chat Lock on Complete]** Sau 24h kể từ `end_time` → `Phòng chat đã bị khóa`.

### Flow 4: Đánh giá & Xử lý Khiếu nại
- **[Actor: Client]** sau khi thời gian hiện tại ≥ `end_time` → `Đánh giá đã được gửi`.
  - **[Policy: One-time Review]** Mỗi booking chỉ cho phép Client đánh giá 1 lần duy nhất (sao + comment). Không cho phép sửa/xóa sau khi đã gửi. Companion không có tính năng phản hồi (BR-09, BR-10).
- Nếu có sự cố (No-show, trải nghiệm tệ...), **[Actor: Client / Companion]** → `Khiếu nại đã được tạo`.
  - **[Policy: Freeze Escrow]** Khi có khiếu nại, tạm ngưng ngay việc hoàn thành và thanh toán tự động (đóng băng Escrow).
- **[Actor: Admin]** can thiệp xử lý:
  - Nếu lỗi do Companion: → `Khiếu nại đã được giải quyết (Refund cho Client)`.
    - **[Policy: Review Hidden]** Khi có phán quyết Refund → `Đánh giá đã bị ẩn` (nếu Client đã viết review — BR-08a) và `Phòng chat đã bị khóa`.
  - Nếu không có lỗi (hoặc lỗi từ Client): → `Khiếu nại đã được giải quyết (Payout cho Companion)`.
    - **[Policy: Review Visible]** Review giữ nguyên trạng thái VISIBLE (BR-08b).

### Flow 5: Quản lý Tài khoản (Admin)
- **[Policy: Ngưỡng vi phạm]** Khi số lần vi phạm tích lũy của Companion đạt ngưỡng cấu hình → `Companion đã đạt ngưỡng vi phạm`.
  - Mặc định: **3 lần vi phạm** → tự động tạm khóa tài khoản Companion (Companion không thể nhận booking mới). Admin xem xét và quyết định mở khóa hoặc khóa vĩnh viễn. *(Ngưỡng và chế tài do Admin cấu hình — BR-07a.)*
  - Hệ thống tự động đánh dấu (flag) tài khoản và gửi `Thông báo đã được gửi` tới Admin để xử lý.
- **[Actor: Admin]** → `Tài khoản đã bị khóa` (khi phát hiện vi phạm nghiêm trọng hoặc khi Companion đạt ngưỡng vi phạm).
- **[Actor: Admin]** → `Tài khoản đã được mở khóa` (khi xét duyệt lại).

### Flow 6: Thông báo (Notification)
- **[Policy: SSE Notification]** Hệ thống gửi thông báo real-time qua SSE cho các sự kiện sau (FR-12):
  - Booking mới được gửi → `Thông báo đã được gửi` (tới Companion).
  - Companion Accept/Reject → `Thông báo đã được gửi` (tới Client).
  - Nhắc nhở trước hẹn 1h → `Thông báo đã được gửi` (tới cả Client và Companion).
  - Booking kết thúc (tại `end_time`) → `Thông báo đã được gửi` (nhắc Client review).

---

## 4. DANH SÁCH EXTERNAL SYSTEMS

| # | External System | Sử dụng trong Flow | Mô tả |
|:---:|:---|:---|:---|
| 1 | **Google OAuth** | Flow 1 (Onboarding) | Xác thực đăng nhập |
| 2 | **VNPay** | Flow 2 (Nạp tiền) | Cổng thanh toán nạp Kano-Coin |
| 3 | **Storage/File Service** | Flow 1 (Profile) | Lưu trữ ảnh đại diện, album, Voice Intro |
| 4 | **Notification Service** | Flow 6 (Thông báo) | Kênh gửi thông báo — hiện tại: SSE; dự phòng scale: FCM, Email |

---

## 5. DANH SÁCH POLICIES

| # | Policy | Nguồn BRD | Áp dụng trong Flow |
|:---:|:---|:---|:---|
| P-01 | Admin duyệt thủ công yêu cầu nâng cấp Companion | FR-01 | Flow 1 |
| P-02 | Freeze coin khi gửi Booking Request | FR-06 | Flow 3 |
| P-03 | Unfreeze coin khi Reject/Timeout | FR-06 | Flow 3 |
| P-04 | Chuyển coin vào Escrow khi Accept | FR-06 | Flow 3 |
| P-05 | Payout coin cho Companion khi hoàn thành | FR-06 | Flow 3 |
| P-06 | Hoa hồng platform khấu trừ trước Payout | BR-03 | Flow 3 |
| P-07 | Timeout booking: 12h hoặc `start_time` - 1h | FR-07 | Flow 3 |
| P-08 | Auto-complete: `end_time` + 12h nếu không có Report | BR-16 | Flow 3 |
| P-09 | Chat tạo sau Accept, khóa sau 24h kể từ `end_time` | BR-14, BR-15 | Flow 3 |
| P-10 | Client hủy sớm (>24h) → hoàn 100% | BR-05 | Flow 3 |
| P-11 | Client hủy muộn (<24h) → mất 100% coin (chuyển cho Companion) | BR-06 | Flow 3 |
| P-12 | Companion hủy muộn → ghi vi phạm | BR-07a | Flow 3 |
| P-13 | Freeze Escrow khi có Report/Dispute | BR-18 | Flow 4 |
| P-14 | Review bị hidden khi Admin refund | BR-08a | Flow 4 |
| P-15 | Bất kỳ người dùng nào (kể cả chưa đăng nhập) đều có quyền nghe Voice Intro | BR-11 | Flow 1 |
| P-16 | Từ chối Freeze khi số dư không đủ → hủy Booking Request | FR-06 | Flow 3 |
| P-17 | Companion đạt ngưỡng vi phạm (mặc định 3 lần) → tạm khóa tài khoản + thông báo Admin | BR-07a | Flow 3, Flow 5 |
| P-18 | Gửi thông báo real-time (SSE) cho các sự kiện booking | FR-12 | Flow 6 |
| P-19 | Mỗi booking chỉ cho phép đánh giá 1 lần, không sửa/xóa | BR-09, BR-10 | Flow 4 |
| P-20 | Review giữ VISIBLE khi Admin payout cho Companion | BR-08b | Flow 4 |
| P-21 | Chat khóa ngay lập tức khi booking bị Cancel hoặc Admin Refund | BR-15 | Flow 3, Flow 4 |
| P-22 | Giới hạn tối đa 10 Pending Request/Companion (Admin cấu hình) | — | Flow 3 |
| P-23 | VNPay IPN là source of truth; giao dịch lỗi đánh dấu "chờ đối soát" | FR-06b | Flow 2 |

---

## 6. XÁC ĐỊNH BOUNDED CONTEXT (SƠ BỘ)

Dựa trên bức tranh tổng thể, hệ thống bao gồm các cụm Bounded Context chính sau:

1. **Identity Context**: Xử lý đăng nhập, quản lý vai trò người dùng và trạng thái tài khoản.
2. **Profile & Catalogue Context**: Nơi Companion quản lý hồ sơ cá nhân, kho media, và các dịch vụ (Scenario).
3. **Booking Context**: Trái tim nghiệp vụ, điều phối toàn bộ State Machine của lịch hẹn.
4. **Finance Context**: Quản lý số dư Kano-Coin, nạp tiền (VNPay), Escrow, và các giao dịch đối soát (Payout, Thu hoa hồng). **Finance Context sở hữu Escrow aggregate**; Booking Context chỉ trigger các thao tác Freeze/Escrow/Payout thông qua Domain Event.
5. **Interaction Context**: Xử lý giao tiếp (Phòng chat) và Đánh giá (Review) giữa hai bên.
6. **Dispute Resolution Context**: Hỗ trợ Admin tiếp nhận Report và phân xử khiếu nại.
7. **Notification Context**: Quản lý tập trung hạ tầng gửi thông báo (SSE, FCM, Email) như một cross-cutting concern, tránh tình trạng phân mảnh (distributed big ball of mud).

### 6.1. Aggregate gợi ý theo Context

| Bounded Context | Aggregate gợi ý | Mô tả |
|:---|:---|:---|
| Identity | **UserAccount** | Quản lý thông tin tài khoản, role, trạng thái (active/locked), bộ đếm vi phạm |
| Profile & Catalogue | **CompanionProfile** | Thông tin cá nhân, thành phố, ảnh đại diện |
| Profile & Catalogue | **MediaAsset** | Album ảnh, Voice Intro |
| Profile & Catalogue | **Scenario** | Kịch bản dịch vụ (tên, mô tả, thời lượng, phí, địa điểm) |
| Booking | **Booking** | Entity trung tâm — quản lý State Machine (Pending → Accepted → Completed...) |
| Finance | **Wallet** | Số dư Kano-Coin của user |
| Finance | **Transaction** | Lịch sử giao dịch (nạp, freeze, unfreeze, escrow, payout) |
| Finance | **Escrow** | Số dư coin bị giữ cho mỗi booking — **Finance Context sở hữu** |
| Interaction | **ChatRoom** | Phòng chat gắn với booking, quản lý trạng thái active/locked |
| Interaction | **Review** | Đánh giá sao + comment, trạng thái visible/hidden |
| Dispute Resolution | **Dispute** | Report/khiếu nại gắn với booking, phán quyết cuối cùng của Admin |
| Notification | **Notification** | Quản lý nội dung, người nhận, kênh phân phối (SSE/FCM/Email) và trạng thái (sent/failed) |

> ⚠️ **Lưu ý:** Danh sách Aggregate ở trên chỉ ở mức gợi ý sơ bộ từ Big Picture Event Storming. Việc xác định chính xác ranh giới Aggregate, invariants, và consistency boundary sẽ được thực hiện ở phase Process Level / Design Level Event Storming.

### 6.2. Ubiquitous Language (Ngôn ngữ chung) theo Context

Trong môi trường phân tán, một khái niệm có thể mang ý nghĩa khác nhau tùy thuộc vào ngữ cảnh (Bounded Context) mà nó đang đứng:

- **Khái niệm "Booking":**
  - Trong **Booking Context**: "Booking" là một cỗ máy trạng thái (State Machine) quản lý toàn bộ vòng đời, lịch trình, thông tin giao tiếp và sự kiện của cuộc hẹn (Pending, Accepted, Completed...).
  - Trong **Finance Context**: "Booking" chỉ mang ý nghĩa là một "mã tham chiếu" (Reference ID) hoặc là "lý do" để thực hiện nghiệp vụ giữ tiền (Freeze Escrow) hoặc giải ngân (Payout). Finance Context hoàn toàn không quan tâm đến ngày giờ hẹn, ai gặp ai, hay ai là người đánh giá.

---

## 7. HOT SPOTS — ĐÃ GIẢI QUYẾT ✅

Các Hot Spot được phát hiện trong quá trình Event Storming đã được phân tích và giải quyết như sau:

| # | Hot Spot | Quyết định | Policy |
|:---:|:---|:---|:---:|
| HS-01 | Ngưỡng vi phạm Companion: bao nhiêu lần để trigger? Chế tài cụ thể? | **Mặc định 3 lần vi phạm** → tự động tạm khóa tài khoản Companion (không nhận booking mới). Admin xem xét mở khóa hoặc khóa vĩnh viễn. Ngưỡng và chế tài do Admin cấu hình (BR-07a). | P-17 |
| HS-02 | Giới hạn số Pending Request tối đa mỗi Companion? | **Mặc định tối đa 10 Pending Request** cùng lúc cho mỗi Companion. Client không gửi thêm được nếu đạt giới hạn. Admin có thể cấu hình lại. | P-22 |
| HS-03 | VNPay timeout nhưng tiền Client đã bị trừ — xử lý thế nào? | **IPN callback từ VNPay là source of truth duy nhất.** Chỉ cộng coin khi IPN xác nhận thành công. Giao dịch không nhận được IPN hoặc IPN thất bại → đánh dấu "chờ đối soát" để Admin tra cứu thủ công khi Client khiếu nại (FR-06b). | P-23 |
| HS-04 | Ai là owner của Escrow — Finance Context hay Booking Context? | **Finance Context sở hữu Escrow aggregate.** Booking Context chỉ phát ra Domain Event (e.g., `BookingAccepted`), Finance Context lắng nghe và thực hiện thao tác Freeze → Escrow → Payout. Đảm bảo tách biệt trách nhiệm tài chính khỏi luồng nghiệp vụ booking. | P-04 |
