package com.rentagf.notification.application.port.inbound;

import java.util.UUID;

/**
 * Inbound Port đánh dấu đã đọc toàn bộ thông báo của một người dùng.
 */
public interface MarkAllAsReadUseCase {

    /**
     * Đánh dấu toàn bộ thông báo chưa đọc của user là đã đọc.
     *
     * @param userId ID người dùng
     * @return số lượng thông báo đã được cập nhật thành công
     */
    int markAllAsRead(UUID userId);
}
