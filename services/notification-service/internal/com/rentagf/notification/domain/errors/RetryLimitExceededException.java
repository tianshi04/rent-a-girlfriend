package com.rentagf.notification.domain.errors;

/** [INV-N01] Vi phạm: Số lần retry đã đạt giới hạn tối đa (3 lần). */
public class RetryLimitExceededException extends NotificationDomainException {

  public RetryLimitExceededException(String notificationId) {
    super(
        "RETRY_LIMIT_EXCEEDED",
        String.format("Notification %s has exceeded maximum retry attempts (3)", notificationId));
  }
}
