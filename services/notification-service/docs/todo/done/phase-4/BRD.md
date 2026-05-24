# 📝 BUSINESS REQUIREMENTS DOCUMENT (BRD) - PHASE 4: REST API

## 1. Tổng quan & Mục tiêu Nghiệp vụ (Business Goal)
Mục tiêu của Phase 4 là cung cấp bộ **REST APIs** chuẩn hóa để Client có thể:
1. **Fetch Inbox**: Lấy danh sách thông báo của mình, hỗ trợ cuộn vô cực (Infinite Scroll) an toàn, hiệu năng cao.
2. **Badge Count**: Hiển thị số lượng thông báo chưa đọc (`unread_count`).
3. **Mark Read**: Đánh dấu đã đọc cho từng thông báo cụ thể.
4. **Mark All Read**: Đánh dấu đã đọc toàn bộ.

---

## 2. Các Ràng buộc & Quy tắc Nghiệp vụ (Business Invariants & Rules)

### [INV-N06]: Trải Nghiệm Infinite Scroll & JSON Serialization
- Bắt buộc dùng **Cursor-based Pagination** dựa trên cặp khóa kép `(created_at, id)`. Sắp xếp với invariant bắt buộc `ORDER BY created_at DESC, id DESC` để đảm bảo:
  - Phân trang mượt mượt, không bao giờ bị lặp dữ liệu do dòng chảy thời gian thực.
  - Hiệu năng truy vấn ở mức $O(1)$ khi kết hợp với Index thích hợp.
- Cursor được đóng gói thành Value Object và tuần tự hóa bằng định dạng chuẩn JSON (`{"createdAt": "...", "id": "..."}`), sau đó mã hóa Base64 URL-safe. Tuyệt đối không dùng custom delimiter dễ gãy vỡ (fragile serialization).

### [INV-N07]: Bảo vệ Dữ liệu Người dùng - Chống Tấn công BOLA/IDOR và Tối Ưu Update
- Lấy danh tính người dùng an toàn từ header `user-id` do Istio Mesh tiêm vào. Không đọc trực tiếp từ Body/Path.
- Thao tác **Mark Read** (đánh dấu một thông báo đã đọc):
  - Hệ thống sử dụng kỹ thuật **Optimistic Update**: Cố gắng UPDATE dữ liệu với điều kiện ID khớp và `user_id` khớp trực tiếp trong cơ sở dữ liệu.
  - Nếu bản ghi không tồn tại hoặc thuộc người dùng khác, hệ thống phải che giấu thông tin bằng cách trả về lỗi `404 Not Found` (mã lỗi `NOTIFICATION_NOT_FOUND`). Cơ chế này bảo mật (chống rà quét ID - BOLA Guard) và giảm triệt để số vòng truy vấn DB (round-trips).

### [INV-N08]: Idempotent cho trạng thái Đọc (Mark Read Preservation)
- Khi gọi thao tác Mark Read lên một thông báo đã đọc từ trước, hệ thống phải trả về Success và không được phép cập nhật ghi đè mốc thời gian đọc (`read_at`) ban đầu.

---

## 3. Mô hình Nhất quán Dữ liệu (Consistency Model)
Hệ thống áp dụng mô hình **Nhất quán cuối cùng (Eventual Consistency)** đối với trường `unread_count` (Badge Count) khi trả về cùng danh sách phân trang:
* Do việc truy vấn danh sách thông báo và đếm số lượng chưa đọc chạy trên 2 câu truy vấn riêng biệt, số lượng hiển thị có thể lệch nhẹ trong một tích tắc dưới áp lực cao. Frontend tin cậy tuyệt đối vào trường `unread_count` từ API để hiển thị UI thay vì tự tính toán lại các items ở Client.

---

## 4. Danh sách Actors & Use Cases

### 4.1. Actors
- **End-User (Client / Companion)**: Người dùng cuối.
- **Istio Service Mesh Waypoint Proxy**: Tác nhân xác thực an toàn.

### 4.2. Use Cases

#### Use Case 1: Fetch Inbox (Tải hộp thư đến)
- Giải mã Cursor JSON an toàn. Truy vấn danh sách và đếm Badge Count chưa đọc thông qua Composite/Partial Index tối ưu.

#### Use Case 2: Mark Single Notification Read (Đánh dấu một thông báo đã đọc)
- **Mục tiêu**: Cập nhật trạng thái một thông báo thành "đã đọc".
- **Luồng xử lý chính**:
  1. Client gọi `PATCH /v1/notifications/{id}/read`.
  2. Hệ thống đọc header `user-id`.
  3. Thực hiện **Optimistic Update** tại mức row-level ở cơ sở dữ liệu.
  4. Trả về thành công nếu update OK hoặc Idempotent. Báo lỗi `404` nếu không tìm thấy/sai quyền.

#### Use Case 3: Mark All Notifications Read (Đánh dấu tất cả thông báo đã đọc)
- Bulk Update trạng thái đọc.
