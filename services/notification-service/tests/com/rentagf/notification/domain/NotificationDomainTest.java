package com.rentagf.notification.domain;

import static org.junit.jupiter.api.Assertions.*;

import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.errors.NotificationAlreadyCompletedException;
import com.rentagf.notification.domain.errors.RetryLimitExceededException;
import com.rentagf.notification.domain.vo.DeliveryAttempt;
import com.rentagf.notification.domain.vo.enums.*;
import java.util.HashMap;
import java.util.Map;
import java.util.UUID;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

@Tag("unit")
class NotificationDomainTest {

  private UUID userId;
  private String eventId;
  private Map<String, Object> payload;

  @BeforeEach
  void setUp() {
    userId = UUID.randomUUID();
    eventId = "evt_" + UUID.randomUUID();
    payload = new HashMap<>();
    payload.put("title", "Hello");
    payload.put("body", "World");
  }

  @Test
  void testInitialNotificationState() {
    Notification notification =
        Notification.create(
            userId,
            eventId,
            NotificationType.TRANSACTIONAL,
            NotificationPriority.HIGH,
            payload,
            Map.of());

    assertNotNull(notification.getId());
    assertEquals(userId, notification.getUserId());
    assertEquals(eventId, notification.getEventId());
    assertEquals(NotificationType.TRANSACTIONAL, notification.getType());
    assertEquals(NotificationPriority.HIGH, notification.getPriority());
    assertEquals(NotificationStatus.PENDING, notification.getStatus());
    assertTrue(notification.getAttempts().isEmpty());
    assertNotNull(notification.getCreatedAt());
    assertNotNull(notification.getUpdatedAt());
  }

  @Test
  void testCreateAttemptSuccessfully() {
    Notification notification =
        Notification.create(
            userId,
            eventId,
            NotificationType.TRANSACTIONAL,
            NotificationPriority.HIGH,
            payload,
            Map.of());

    DeliveryAttempt attempt = notification.createAttempt(DeliveryChannel.SSE);

    assertNotNull(attempt.getId());
    assertEquals(notification.getId(), attempt.getNotificationId());
    assertEquals(DeliveryChannel.SSE, attempt.getChannel());
    assertEquals(AttemptStatus.PENDING, attempt.getStatus());
    assertEquals(NotificationStatus.PROCESSING, notification.getStatus());
    assertEquals(1, notification.getAttempts().size());
  }

  @Test
  void testMarkAttemptSuccess() {
    Notification notification =
        Notification.create(
            userId,
            eventId,
            NotificationType.TRANSACTIONAL,
            NotificationPriority.HIGH,
            payload,
            Map.of());
    DeliveryAttempt attempt = notification.createAttempt(DeliveryChannel.SSE);

    notification.markAttemptSuccess(attempt.getId(), "msg_123");

    assertEquals(AttemptStatus.SUCCESS, attempt.getStatus());
    assertEquals("msg_123", attempt.getMessageId());
    assertNotNull(attempt.getResolvedAt());
    assertEquals(NotificationStatus.COMPLETED, notification.getStatus());
  }

  @Test
  void testMarkAttemptFailedUnrecoverable() {
    Notification notification =
        Notification.create(
            userId,
            eventId,
            NotificationType.TRANSACTIONAL,
            NotificationPriority.HIGH,
            payload,
            Map.of());
    DeliveryAttempt attempt = notification.createAttempt(DeliveryChannel.FCM);

    // Unrecoverable error (e.g. invalid token)
    notification.markAttemptFailed(
        attempt.getId(), "FCM_TOKEN_INVALID", "FCM token is expired or invalid", false);

    assertEquals(AttemptStatus.FAILED_UNRECOVERABLE, attempt.getStatus());
    assertEquals("FCM_TOKEN_INVALID", attempt.getErrorCode());
    assertEquals("FCM token is expired or invalid", attempt.getErrorMessage());
    assertNotNull(attempt.getResolvedAt());

    // Should transition directly to FAILED
    assertEquals(NotificationStatus.FAILED, notification.getStatus());
  }

  @Test
  void testMarkAttemptFailedRecoverableKeepProcessing() {
    Notification notification =
        Notification.create(
            userId,
            eventId,
            NotificationType.TRANSACTIONAL,
            NotificationPriority.HIGH,
            payload,
            Map.of());
    DeliveryAttempt attempt = notification.createAttempt(DeliveryChannel.EMAIL);

    // Recoverable error (e.g. connection timeout)
    notification.markAttemptFailed(
        attempt.getId(), "SMTP_TIMEOUT", "SMTP connection timed out", true);

    assertEquals(AttemptStatus.FAILED_RECOVERABLE, attempt.getStatus());
    assertEquals("SMTP_TIMEOUT", attempt.getErrorCode());
    assertNotNull(attempt.getResolvedAt());

    // Should keep status as PROCESSING for retry
    assertEquals(NotificationStatus.PROCESSING, notification.getStatus());
  }

  @Test
  void testInvariantN01MaxRetriesExceeded() {
    Notification notification =
        Notification.create(
            userId,
            eventId,
            NotificationType.TRANSACTIONAL,
            NotificationPriority.HIGH,
            payload,
            Map.of());

    // Attempt 1: Fail Recoverable
    DeliveryAttempt attempt1 = notification.createAttempt(DeliveryChannel.EMAIL);
    notification.markAttemptFailed(attempt1.getId(), "SMTP_TIMEOUT", "Timeout 1", true);
    assertEquals(NotificationStatus.PROCESSING, notification.getStatus());

    // Attempt 2: Fail Recoverable
    DeliveryAttempt attempt2 = notification.createAttempt(DeliveryChannel.EMAIL);
    notification.markAttemptFailed(attempt2.getId(), "SMTP_TIMEOUT", "Timeout 2", true);
    assertEquals(NotificationStatus.PROCESSING, notification.getStatus());

    // Attempt 3: Fail Recoverable -> Should reach 3 failures and mark Notification as FAILED
    DeliveryAttempt attempt3 = notification.createAttempt(DeliveryChannel.EMAIL);
    notification.markAttemptFailed(attempt3.getId(), "SMTP_TIMEOUT", "Timeout 3", true);

    assertEquals(NotificationStatus.FAILED, notification.getStatus());

    // Attempt 4: Should throw RetryLimitExceededException
    assertThrows(
        RetryLimitExceededException.class, () -> notification.createAttempt(DeliveryChannel.EMAIL));
  }

  @Test
  void testInvariantN02CannotAttemptAfterCompleted() {
    Notification notification =
        Notification.create(
            userId,
            eventId,
            NotificationType.TRANSACTIONAL,
            NotificationPriority.HIGH,
            payload,
            Map.of());
    DeliveryAttempt attempt = notification.createAttempt(DeliveryChannel.SSE);

    notification.markAttemptSuccess(attempt.getId(), "msg_123");
    assertEquals(NotificationStatus.COMPLETED, notification.getStatus());

    // Try creating another attempt -> Should throw NotificationAlreadyCompletedException
    assertThrows(
        NotificationAlreadyCompletedException.class,
        () -> notification.createAttempt(DeliveryChannel.FCM));
  }
}
