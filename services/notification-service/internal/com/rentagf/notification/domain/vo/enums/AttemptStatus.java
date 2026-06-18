package com.rentagf.notification.domain.vo.enums;

/** Trạng thái của một DeliveryAttempt. Tham chiếu: docs/state-machine.md §2 */
public enum AttemptStatus {
  PENDING,
  SUCCESS,
  FAILED_RECOVERABLE,
  FAILED_UNRECOVERABLE
}
