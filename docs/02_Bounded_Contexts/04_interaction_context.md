# INTERACTION CONTEXT

**Phân loại Subdomain:** Supporting Subdomain

## 1. TRÁCH NHIỆM CHÍNH (RESPONSIBILITY)
Cung cấp môi trường giao tiếp an toàn (Phòng chat) và hệ thống ghi nhận chất lượng dịch vụ (Đánh giá) sau khi có kết nối giữa hai bên (Client & Companion).

## 2. AGGREGATES & ENTITIES
### Aggregate Root 1: `ChatRoom`
Quản lý quyền truy cập, vòng đời và trạng thái đóng/mở của phòng chat.
*   **State:** `RoomId`, `BookingId`, `ParticipantIds` [Client, Companion], `Status` (ACTIVE, LOCKED).
*   **Entities con:** `ChatMessage` (Lưu text tin nhắn. Có thể lưu dạng Entity con nếu dùng Document DB như MongoDB, hoặc tách riêng nếu dùng RDBMS).

### Aggregate Root 2: `Review`
Quản lý đánh giá độc lập, duy trì tính duy nhất và tính hiển thị của đánh giá.
*   **State:** `ReviewId`, `BookingId`, `ClientId`, `CompanionId`, `Rating`, `Comment`, `IsVisible`.

## 3. VALUE OBJECTS
*   `Rating`: Chứa giá trị Integer từ 1 đến 5. Ném lỗi nếu giá trị nằm ngoài khoảng này.
*   `ChatContent`: Đóng gói text tin nhắn, có thể chứa logic filter từ ngữ nhạy cảm (profanity filter).

## 4. INVARIANTS (QUY TẮC BẤT BIẾN)
*   `[INV-I01]` Chỉ những User có trong `ParticipantIds` mới được gửi/đọc tin nhắn.
*   `[INV-I02]` Không thể gửi tin nhắn nếu Status của ChatRoom là `LOCKED`.
*   `[INV-I03]` Khi ChatRoom đã bị `LOCKED`, không thể chuyển ngược lại thành `ACTIVE`.
*   `[INV-I04]` Mỗi `BookingId` chỉ được tồn tại đúng 1 `Review`.
*   `[INV-I05]` Rating của Review phải nằm trong khoảng từ 1 đến 5.

## 5. THIẾT KẾ COMMAND & EVENT
*(Ghi chú: Interaction Context chủ yếu hoạt động dựa trên việc lắng nghe Command nội bộ từ việc nhận Event của Booking/Dispute Context)*
*   **Lắng nghe:** Khi `BookingAccepted` sinh ra từ Booking Context, Interaction sẽ gọi command `CreateChatRoom` nội bộ.
*   **Sự kiện sinh ra:** `ChatRoomCreated`, `ChatRoomLocked`, `ReviewSubmitted`, `ReviewHidden`.

## 6. DOMAIN SERVICES
*   `ProfanityFilterService`: Chứa logic kiểm tra và làm mờ các từ ngữ vi phạm tiêu chuẩn cộng đồng hoặc chia sẻ thông tin liên lạc bên ngoài (Zalo, SĐT) trong tin nhắn chat.

## 7. REPOSITORIES
*   `IChatRoomRepository`
*   `IReviewRepository`
