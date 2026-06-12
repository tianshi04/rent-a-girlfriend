package com.rentagf.notification.domain.vo;

import java.time.Instant;
import java.util.UUID;

/**
 * Value Object đại diện cho cursor phân trang của danh sách thông báo. Lưu trữ thời gian tạo và ID
 * duy nhất để đảm bảo tính bất biến của thuật toán Cursor-based Pagination.
 */
public record InboxCursor(Instant createdAt, UUID id) {
  public InboxCursor {
    if (createdAt == null) {
      throw new IllegalArgumentException("createdAt cannot be null");
    }
    if (id == null) {
      throw new IllegalArgumentException("id cannot be null");
    }
  }
}
