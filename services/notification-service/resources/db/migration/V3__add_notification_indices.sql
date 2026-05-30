-- V3__add_notification_indices.sql
-- Tối ưu hóa Database cho phân trang Cursor-based và đếm thông báo chưa đọc

-- 1. Composite Index tối ưu tuyệt đối cho truy vấn phân trang Cursor-based theo (user_id, created_at DESC, id DESC)
CREATE INDEX idx_notifications_cursor_pagination ON notifications (user_id, created_at DESC, id DESC);

-- 2. Partial Index tối ưu hóa việc đếm và lọc các thông báo chưa đọc (unread_only = true) của từng người dùng
CREATE INDEX idx_notifications_unread_partial ON notifications (user_id) WHERE read_at IS NULL;
