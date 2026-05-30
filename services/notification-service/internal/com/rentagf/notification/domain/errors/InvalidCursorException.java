package com.rentagf.notification.domain.errors;

/**
 * Ngoại lệ được ném ra khi Cursor phân trang không hợp lệ (không thể giải mã hoặc định dạng sai).
 */
public class InvalidCursorException extends NotificationDomainException {

    public InvalidCursorException(String message) {
        super("INVALID_CURSOR", message);
    }

    public InvalidCursorException(String message, Throwable cause) {
        super("INVALID_CURSOR", message + " - Cause: " + cause.getMessage());
    }
}
