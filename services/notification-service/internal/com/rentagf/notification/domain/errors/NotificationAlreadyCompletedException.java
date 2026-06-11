package com.rentagf.notification.domain.errors;

/** [INV-N02] Vi phạm: Không được tạo attempt mới sau khi Notification đã COMPLETED. */
public class NotificationAlreadyCompletedException extends NotificationDomainException {

  public NotificationAlreadyCompletedException(String notificationId) {
    super(
        "NOTIFICATION_ALREADY_COMPLETED",
        String.format(
            "Notification %s is already completed. No further attempts allowed", notificationId));
  }
}
