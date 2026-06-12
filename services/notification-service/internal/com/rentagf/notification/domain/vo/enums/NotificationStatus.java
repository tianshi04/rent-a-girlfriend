package com.rentagf.notification.domain.vo.enums;

/**
 * Trạng thái vòng đời của Notification. Tham chiếu: docs/state-machine.md
 *
 * <p>PENDING → PROCESSING → COMPLETED | FAILED
 */
public enum NotificationStatus {
  PENDING,
  PROCESSING,
  COMPLETED,
  FAILED
}
