package com.rentagf.notification.domain.exception;

/**
 * Notification không tìm thấy trong DB.
 */
public class NotificationNotFoundException extends NotificationDomainException {

    public NotificationNotFoundException(String notificationId) {
        super("NOTIFICATION_NOT_FOUND",
                String.format("Notification %s not found", notificationId));
    }
}
