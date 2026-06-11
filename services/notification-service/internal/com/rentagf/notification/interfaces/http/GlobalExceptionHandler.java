package com.rentagf.notification.interfaces.http;

import com.rentagf.notification.domain.errors.DuplicateEventException;
import com.rentagf.notification.domain.errors.NotificationAlreadyCompletedException;
import com.rentagf.notification.domain.errors.NotificationDomainException;
import com.rentagf.notification.domain.errors.NotificationNotFoundException;
import com.rentagf.notification.domain.errors.RetryLimitExceededException;
import java.util.List;
import java.util.Map;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;

/**
 * Global Error Handler — map Domain Errors sang HTTP responses. Tham chiếu: docs/api-contract.md §3
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
  public ResponseEntity<Map<String, Object>> handleAlreadyCompleted(
      NotificationAlreadyCompletedException ex) {
    log.warn("Already completed: {}", ex.getMessage());
    return buildResponse(HttpStatus.CONFLICT, ex);
  }

  @ExceptionHandler(RetryLimitExceededException.class)
  public ResponseEntity<Map<String, Object>> handleRetryExceeded(RetryLimitExceededException ex) {
    log.warn("Retry limit exceeded: {}", ex.getMessage());
    return buildResponse(HttpStatus.UNPROCESSABLE_ENTITY, ex);
  }

  @ExceptionHandler(com.rentagf.notification.domain.errors.InvalidCursorException.class)
  public ResponseEntity<Map<String, Object>> handleInvalidCursor(
      com.rentagf.notification.domain.errors.InvalidCursorException ex) {
    log.warn("Invalid cursor: {}", ex.getMessage());
    return buildResponse(HttpStatus.BAD_REQUEST, ex);
  }

  @ExceptionHandler(IllegalArgumentException.class)
  public ResponseEntity<Map<String, Object>> handleIllegalArgument(IllegalArgumentException ex) {
    log.warn("Invalid argument: {}", ex.getMessage());
    return ResponseEntity.status(HttpStatus.BAD_REQUEST)
        .body(
            Map.of(
                "error",
                Map.of(
                    "code", "INVALID_PARAMETER",
                    "message", ex.getMessage(),
                    "details", List.of())));
  }

  @ExceptionHandler(org.springframework.web.bind.MissingRequestHeaderException.class)
  public ResponseEntity<Map<String, Object>> handleMissingHeader(
      org.springframework.web.bind.MissingRequestHeaderException ex) {
    log.warn("Missing request header: {}", ex.getMessage());
    return ResponseEntity.status(HttpStatus.BAD_REQUEST)
        .body(
            Map.of(
                "error",
                Map.of(
                    "code", "MISSING_HEADER",
                    "message",
                        String.format("Missing required request header '%s'", ex.getHeaderName()),
                    "details", List.of())));
  }

  @ExceptionHandler(Exception.class)
  public ResponseEntity<Map<String, Object>> handleUnexpected(Exception ex) {
    log.error("Unexpected error", ex);
    return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR)
        .body(
            Map.of(
                "error",
                Map.of(
                    "code", "INTERNAL_ERROR",
                    "message", "An unexpected error occurred",
                    "details", List.of())));
  }

  private ResponseEntity<Map<String, Object>> buildResponse(
      HttpStatus status, NotificationDomainException ex) {
    return ResponseEntity.status(status)
        .body(
            Map.of(
                "error",
                Map.of(
                    "code", ex.getErrorCode(),
                    "message", ex.getMessage(),
                    "details", List.of())));
  }
}
