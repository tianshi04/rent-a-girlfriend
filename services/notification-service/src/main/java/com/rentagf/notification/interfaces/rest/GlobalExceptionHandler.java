package com.rentagf.notification.interfaces.rest;

import com.rentagf.notification.domain.exception.DuplicateEventException;
import com.rentagf.notification.domain.exception.NotificationAlreadyCompletedException;
import com.rentagf.notification.domain.exception.NotificationDomainException;
import com.rentagf.notification.domain.exception.NotificationNotFoundException;
import com.rentagf.notification.domain.exception.RetryLimitExceededException;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;

import java.time.Instant;
import java.util.Map;

/**
 * Global Error Handler — map Domain Errors sang HTTP responses.
 * Tham chiếu: docs/api-contract.md §3
 */
@Slf4j
@RestControllerAdvice
public class GlobalExceptionHandler {

    @ExceptionHandler(NotificationNotFoundException.class)
    public ResponseEntity<Map<String, Object>> handleNotFound(NotificationNotFoundException ex) {
        log.warn("Not found: {}", ex.getMessage());
        return buildResponse(HttpStatus.NOT_FOUND, ex);
    }

    @ExceptionHandler(DuplicateEventException.class)
    public ResponseEntity<Map<String, Object>> handleDuplicate(DuplicateEventException ex) {
        log.warn("Duplicate event: {}", ex.getMessage());
        return buildResponse(HttpStatus.CONFLICT, ex);
    }

    @ExceptionHandler(NotificationAlreadyCompletedException.class)
    public ResponseEntity<Map<String, Object>> handleAlreadyCompleted(NotificationAlreadyCompletedException ex) {
        log.warn("Already completed: {}", ex.getMessage());
        return buildResponse(HttpStatus.CONFLICT, ex);
    }

    @ExceptionHandler(RetryLimitExceededException.class)
    public ResponseEntity<Map<String, Object>> handleRetryExceeded(RetryLimitExceededException ex) {
        log.warn("Retry limit exceeded: {}", ex.getMessage());
        return buildResponse(HttpStatus.UNPROCESSABLE_ENTITY, ex);
    }

    @ExceptionHandler(Exception.class)
    public ResponseEntity<Map<String, Object>> handleUnexpected(Exception ex) {
        log.error("Unexpected error", ex);
        return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR)
                .body(Map.of(
                        "error_code", "INTERNAL_ERROR",
                        "message", "An unexpected error occurred",
                        "timestamp", Instant.now().toString()
                ));
    }

    private ResponseEntity<Map<String, Object>> buildResponse(HttpStatus status, NotificationDomainException ex) {
        return ResponseEntity.status(status)
                .body(Map.of(
                        "error_code", ex.getErrorCode(),
                        "message", ex.getMessage(),
                        "timestamp", Instant.now().toString()
                ));
    }
}
