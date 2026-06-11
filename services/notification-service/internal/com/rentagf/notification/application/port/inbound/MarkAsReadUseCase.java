package com.rentagf.notification.application.port.inbound;

import java.util.UUID;

/** Inbound Port đánh dấu một thông báo là đã đọc. */
public interface MarkAsReadUseCase {

  /**
   * Đánh dấu một thông báo là đã đọc, kèm BOLA check bảo vệ quyền sở hữu của user.
   *
   * @param notificationId ID thông báo cần đánh dấu đọc
   * @param userId ID người dùng thực hiện (để xác thực quyền sở hữu)
   * @throws com.rentagf.notification.domain.errors.NotificationNotFoundException Nếu không tìm thấy
   *     thông báo hoặc không có quyền sở hữu
   */
  void markAsRead(UUID notificationId, UUID userId);
}
