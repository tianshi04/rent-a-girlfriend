package com.rentagf.notification.application.service;

import com.rentagf.notification.application.event.NotificationReadyEvent;
import com.rentagf.notification.application.port.outbound.NotificationSender;
import com.rentagf.notification.application.port.outbound.RetrySchedulerPort;
import com.rentagf.notification.application.port.outbound.SendResult;
import com.rentagf.notification.application.registry.NotificationSenderRegistry;
import com.rentagf.notification.application.service.NotificationDeliveryTransactionService.DeliveryActionResult;
import com.rentagf.notification.application.service.NotificationDeliveryTransactionService.DeliveryContext;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.vo.enums.DeliveryChannel;
import com.rentagf.notification.domain.vo.enums.NotificationPriority;
import com.rentagf.notification.domain.vo.enums.NotificationType;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;
import org.springframework.context.ApplicationEventPublisher;

import java.time.Duration;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.mockito.Mockito.*;

class AsyncNotificationDeliveryServiceTest {

    private NotificationSenderRegistry senderRegistry;
    private NotificationDeliveryTransactionService txService;
    private RetrySchedulerPort retrySchedulerPort;
    private ApplicationEventPublisher eventPublisher;

    private AsyncNotificationDeliveryService asyncDeliveryService;

    @BeforeEach
    void setUp() {
        senderRegistry = mock(NotificationSenderRegistry.class);
        txService = mock(NotificationDeliveryTransactionService.class);
        retrySchedulerPort = mock(RetrySchedulerPort.class);
        eventPublisher = mock(ApplicationEventPublisher.class);

        asyncDeliveryService = new AsyncNotificationDeliveryService(
                senderRegistry, txService, retrySchedulerPort, eventPublisher
        );
    }

    @Test
    void testSendAsync_Success_UpdatesStatusToCompleted() {
        // Given
        UUID notificationId = UUID.randomUUID();
        UUID attemptId = UUID.randomUUID();
        DeliveryChannel channel = DeliveryChannel.EMAIL;

        Notification notification = Notification.create(
                UUID.randomUUID(), "evt-123", NotificationType.TRANSACTIONAL,
                NotificationPriority.HIGH, Map.of("title", "Hello"), Map.of()
        );

        DeliveryContext context = new DeliveryContext(notification, attemptId);
        when(txService.prepareAttempt(notificationId, channel)).thenReturn(context);

        NotificationSender sender = mock(NotificationSender.class);
        when(senderRegistry.getSender(channel)).thenReturn(Optional.of(sender));

        SendResult sendResult = SendResult.success("msg-ok-123");
        when(sender.send(notification)).thenReturn(sendResult);

        DeliveryActionResult actionResult = new DeliveryActionResult(false, false, 0);
        when(txService.handleResult(notificationId, attemptId, sendResult, channel)).thenReturn(actionResult);

        // When
        asyncDeliveryService.sendAsync(notificationId, channel);

        // Then
        verify(txService, times(1)).prepareAttempt(notificationId, channel);
        verify(sender, times(1)).send(notification);
        verify(txService, times(1)).handleResult(notificationId, attemptId, sendResult, channel);
        verifyNoInteractions(retrySchedulerPort);
        verifyNoInteractions(eventPublisher);
    }

    @Test
    void testSendAsync_RecoverableFailure_SchedulesRetry() {
        // Given
        UUID notificationId = UUID.randomUUID();
        UUID attemptId = UUID.randomUUID();
        DeliveryChannel channel = DeliveryChannel.FCM;

        Notification notification = Notification.create(
                UUID.randomUUID(), "evt-123", NotificationType.TRANSACTIONAL,
                NotificationPriority.HIGH, Map.of("title", "Hello"), Map.of()
        );

        DeliveryContext context = new DeliveryContext(notification, attemptId);
        when(txService.prepareAttempt(notificationId, channel)).thenReturn(context);

        NotificationSender sender = mock(NotificationSender.class);
        when(senderRegistry.getSender(channel)).thenReturn(Optional.of(sender));

        SendResult sendResult = SendResult.fail("FCM_TIMEOUT", "Timeout", true);
        when(sender.send(notification)).thenReturn(sendResult);

        // Giả lập sau khi handle thì quyết định retry, attempt thất bại lần 1
        DeliveryActionResult actionResult = new DeliveryActionResult(true, false, 1);
        when(txService.handleResult(notificationId, attemptId, sendResult, channel)).thenReturn(actionResult);

        // When
        asyncDeliveryService.sendAsync(notificationId, channel);

        // Then
        verify(txService, times(1)).prepareAttempt(notificationId, channel);
        verify(sender, times(1)).send(notification);
        verify(txService, times(1)).handleResult(notificationId, attemptId, sendResult, channel);

        // FCM lần 1 -> delay 2s
        verify(retrySchedulerPort, times(1)).scheduleRetry(
                eq(notificationId), eq(channel), eq(Duration.ofSeconds(2))
        );
        verifyNoInteractions(eventPublisher);
    }

    @Test
    void testSendAsync_UnrecoverableFailure_NoRetryAndFailsNotification() {
        // Given
        UUID notificationId = UUID.randomUUID();
        UUID attemptId = UUID.randomUUID();
        DeliveryChannel channel = DeliveryChannel.EMAIL;

        Notification notification = Notification.create(
                UUID.randomUUID(), "evt-123", NotificationType.TRANSACTIONAL,
                NotificationPriority.HIGH, Map.of("title", "Hello"), Map.of()
        );

        DeliveryContext context = new DeliveryContext(notification, attemptId);
        when(txService.prepareAttempt(notificationId, channel)).thenReturn(context);

        NotificationSender sender = mock(NotificationSender.class);
        when(senderRegistry.getSender(channel)).thenReturn(Optional.of(sender));

        SendResult sendResult = SendResult.fail("SMTP_INVALID_RECIPIENT", "No such user", false);
        when(sender.send(notification)).thenReturn(sendResult);

        // Giả lập lỗi unrecoverable -> không retry, không fallback
        DeliveryActionResult actionResult = new DeliveryActionResult(false, false, 1);
        when(txService.handleResult(notificationId, attemptId, sendResult, channel)).thenReturn(actionResult);

        // When
        asyncDeliveryService.sendAsync(notificationId, channel);

        // Then
        verify(txService, times(1)).prepareAttempt(notificationId, channel);
        verify(sender, times(1)).send(notification);
        verify(txService, times(1)).handleResult(notificationId, attemptId, sendResult, channel);
        verifyNoInteractions(retrySchedulerPort);
        verifyNoInteractions(eventPublisher);
    }

    @Test
    void testSendAsync_SseFailure_TriggersFcmFallback() {
        // Given
        UUID notificationId = UUID.randomUUID();
        UUID attemptId = UUID.randomUUID();
        DeliveryChannel channel = DeliveryChannel.SSE;

        Notification notification = Notification.create(
                UUID.randomUUID(), "evt-123", NotificationType.TRANSACTIONAL,
                NotificationPriority.HIGH, Map.of("title", "Hello"), Map.of("channels", List.of("SSE", "FCM"))
        );

        DeliveryContext context = new DeliveryContext(notification, attemptId);
        when(txService.prepareAttempt(notificationId, channel)).thenReturn(context);

        NotificationSender sender = mock(NotificationSender.class);
        when(senderRegistry.getSender(channel)).thenReturn(Optional.of(sender));

        SendResult sendResult = SendResult.fail("SSE_CLOSED", "Client disconnected", true);
        when(sender.send(notification)).thenReturn(sendResult);

        // Giả lập SSE hỏng -> trigger fallback sang FCM
        DeliveryActionResult actionResult = new DeliveryActionResult(false, true, 1);
        when(txService.handleResult(notificationId, attemptId, sendResult, channel)).thenReturn(actionResult);

        // When
        asyncDeliveryService.sendAsync(notificationId, channel);

        // Then
        verify(txService, times(1)).prepareAttempt(notificationId, channel);
        verify(sender, times(1)).send(notification);
        verify(txService, times(1)).handleResult(notificationId, attemptId, sendResult, channel);
        verifyNoInteractions(retrySchedulerPort);

        // Kiểm tra xem sự kiện NotificationReadyEvent(FCM) được phát hành
        ArgumentCaptor<NotificationReadyEvent> eventCaptor = ArgumentCaptor.forClass(NotificationReadyEvent.class);
        verify(eventPublisher, times(1)).publishEvent(eventCaptor.capture());

        NotificationReadyEvent firedEvent = eventCaptor.getValue();
        assertEquals(notificationId, firedEvent.getNotificationId());
        assertEquals(DeliveryChannel.FCM, firedEvent.getChannel());
    }
}
