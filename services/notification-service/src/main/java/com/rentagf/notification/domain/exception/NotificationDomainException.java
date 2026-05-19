package com.rentagf.notification.domain.exception;

/**
 * Base class cho tất cả Domain Errors của Notification Service.
 * Map sang HTTP/gRPC code ở tầng interfaces.
 */
public abstract class NotificationDomainException extends RuntimeException {

    private final String errorCode;

    protected NotificationDomainException(String errorCode, String message) {
        super(message);
        this.errorCode = errorCode;
    }

    public String getErrorCode() {
        return errorCode;
    }
}
