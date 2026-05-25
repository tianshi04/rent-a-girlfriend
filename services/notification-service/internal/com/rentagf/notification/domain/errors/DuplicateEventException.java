package com.rentagf.notification.domain.errors;

/**
 * [INV-N03] Vi phạm: Event đã được xử lý trước đó (duplicate eventId + userId).
 */
public class DuplicateEventException extends NotificationDomainException {

    public DuplicateEventException(String idempotencyKey, String userId) {
        super("DUPLICATE_EVENT",
                String.format("Event %s for user %s has already been processed", idempotencyKey, userId));
    }
}
