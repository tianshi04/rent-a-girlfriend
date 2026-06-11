package com.rentagf.notification.application.port.inbound;

import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.vo.InboxCursor;
import java.util.List;
import java.util.UUID;

/** Inbound Port lấy hộp thư thông báo (Inbox) của người dùng. Hỗ trợ phân trang Cursor-based. */
public interface FetchInboxUseCase {

  /**
   * Lấy danh sách thông báo phân trang của người dùng.
   *
   * @param userId ID người dùng
   * @param cursor đối tượng InboxCursor (null nếu lấy trang đầu)
   * @param limit giới hạn bản ghi
   * @param unreadOnly chỉ lấy các thông báo chưa đọc
   * @return danh sách thông báo đã phân trang và sắp xếp theo ORDER BY Invariant
   */
  List<Notification> fetchInbox(UUID userId, InboxCursor cursor, int limit, boolean unreadOnly);
}
