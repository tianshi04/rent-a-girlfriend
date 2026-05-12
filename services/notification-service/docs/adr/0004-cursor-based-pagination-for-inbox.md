# ADR 0004: Cursor-based Pagination for Inbox API

## Trạng thái (Status)
Đã duyệt (Accepted)

## Bối cảnh (Context)
Notification Service cần cung cấp API `GET /v1/notifications` để Frontend (Web/Mobile) hiển thị danh sách Inbox (Quả chuông thông báo). 
Khi số lượng thông báo lớn, chúng ta bắt buộc phải phân trang (Pagination). Có hai phương pháp phổ biến:
1. **Offset/Page-based Pagination** (Dùng `page=1&size=20`).
2. **Cursor-based Pagination** (Dùng `cursor=xxx&limit=20`).

Đặc thù của hệ thống Realtime là dữ liệu mới (thông báo mới) có thể liên tục chèn vào đầu danh sách (được đẩy qua SSE). Nếu Frontend đang dùng Offset-based và đang ở `page=1`, có 2 thông báo mới bay vào, khi Frontend gọi `page=2`, họ sẽ bị thấy **lặp lại** 2 thông báo cuối của trang 1 (do toàn bộ dữ liệu trong DB đã bị đẩy lùi xuống). Điều này gây lỗi hiển thị (Duplicate UI Keys) và trải nghiệm tồi tệ.

## Quyết định (Decision)
1. **Bắt buộc áp dụng Cursor-based Pagination** cho API lấy danh sách Inbox ngay từ giai đoạn MVP.
2. Cursor sẽ được thiết kế dựa trên một mã hóa (Base64) của trường `created_at` (Timestamp) cộng với `id` (UUID) để đảm bảo tính duy nhất và hỗ trợ sắp xếp.
3. API response sẽ luôn trả về một trường `next_cursor`. Khi Frontend muốn load-more, họ chỉ cần lấy giá trị này nhét vào Request Parameter tiếp theo.

## Hệ quả (Consequences)
- **Tích cực:** 
  - Khắc phục hoàn toàn lỗi lặp data (Duplicate records) khi có tin nhắn realtime xen vào.
  - Hiệu năng Database (PostgreSQL) cực tốt: Tránh được bài toán "Slow Offset" khi user lướt tới hàng ngàn tin nhắn cũ. Câu lệnh SQL chỉ dùng điều kiện `WHERE created_at < ?`.
- **Tiêu cực:** 
  - Đội ngũ Frontend không thể làm tính năng "Nhảy cóc tới trang số 10" (nhưng đối với tính năng Inbox kiểu cuộn vô hạn - Infinite Scroll thì việc nhảy trang là không cần thiết).
  - Code Backend phức tạp hơn đôi chút ở khâu giải mã/mã hóa Cursor thay vì chỉ truyền `page` và `size`.
