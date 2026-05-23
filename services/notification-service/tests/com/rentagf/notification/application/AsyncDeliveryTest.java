package com.rentagf.notification.application;

import com.rentagf.notification.application.port.outbound.EmailPort;
import com.rentagf.notification.application.port.outbound.FcmPort;
import com.rentagf.notification.application.port.outbound.SendResult;
import com.rentagf.notification.application.port.outbound.SsePort;
import com.rentagf.notification.application.service.NotificationApplicationService;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.repository.NotificationRepository;
import com.rentagf.notification.domain.vo.enums.*;
import com.rentagf.notification.domain.errors.DuplicateEventException;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.bean.override.mockito.MockitoBean;

import java.time.Duration;
import java.util.Map;
import java.util.UUID;

import static org.awaitility.Awaitility.await;
import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.when;

@SpringBootTest
@Tag("integration")
class AsyncDeliveryTest {

    @Autowired
    private NotificationApplicationService applicationService;

    @Autowired
    private NotificationRepository notificationRepository;

    @MockitoBean
    private EmailPort emailPort;

    @MockitoBean
    private FcmPort fcmPort;

    @MockitoBean
    private SsePort ssePort;

    @BeforeEach
    void setUp() {
        // Setup channels for mock strategies
        when(emailPort.getChannel()).thenReturn(DeliveryChannel.EMAIL);
        when(fcmPort.getChannel()).thenReturn(DeliveryChannel.FCM);
        when(ssePort.getChannel()).thenReturn(DeliveryChannel.SSE);
    }

    @Test
    void testSuccessfulAsyncDelivery() {
        UUID userId = UUID.randomUUID();
        String eventId = "evt_async_success_" + UUID.randomUUID();

        // Stub EmailPort to succeed
        when(emailPort.send(any(Notification.class)))
                .thenReturn(SendResult.success("msg_email_ok"));

        // Trigger notification
        Notification triggered = applicationService.triggerNotification(
                userId, eventId, NotificationType.TRANSACTIONAL, NotificationPriority.HIGH,
                Map.of("title", "Hello Async"), DeliveryChannel.EMAIL
        );

        // Main thread should return immediately, status could be PENDING or PROCESSING initially
        assertNotNull(triggered);
        
        // Wait using Awaitility for virtual threads to finish and verify final state in database
        await().atMost(Duration.ofSeconds(5))
                .untilAsserted(() -> {
                    Notification found = notificationRepository.findById(triggered.getId()).orElse(null);
                    assertNotNull(found);
                    assertEquals(NotificationStatus.COMPLETED, found.getStatus());
                    assertEquals(1, found.getAttempts().size());
                    assertEquals(AttemptStatus.SUCCESS, found.getAttempts().get(0).getStatus());
                    assertEquals("msg_email_ok", found.getAttempts().get(0).getMessageId());
                });
    }

    @Test
    void testAsyncDeliveryUnrecoverableFailureDirectlyFails() {
        UUID userId = UUID.randomUUID();
        String eventId = "evt_async_unrecoverable_" + UUID.randomUUID();

        // Stub FcmPort to return unrecoverable failure
        when(fcmPort.send(any(Notification.class)))
                .thenReturn(SendResult.fail("FCM_TOKEN_INVALID", "Token is dead", false));

        Notification triggered = applicationService.triggerNotification(
                userId, eventId, NotificationType.TRANSACTIONAL, NotificationPriority.HIGH,
                Map.of("title", "Hello FCM"), DeliveryChannel.FCM
        );

        assertNotNull(triggered);

        await().atMost(Duration.ofSeconds(5))
                .untilAsserted(() -> {
                    Notification found = notificationRepository.findById(triggered.getId()).orElse(null);
                    assertNotNull(found);
                    // Unrecoverable -> instantly FAILED
                    assertEquals(NotificationStatus.FAILED, found.getStatus());
                    assertEquals(1, found.getAttempts().size());
                    assertEquals(AttemptStatus.FAILED_UNRECOVERABLE, found.getAttempts().get(0).getStatus());
                });
    }

    @Test
    void testDuplicateEventExceptionThrown() {
        UUID userId = UUID.randomUUID();
        String eventId = "evt_duplicate_" + UUID.randomUUID();

        // Lần 1: Trigger thành công
        Notification first = applicationService.triggerNotification(
                userId, eventId, NotificationType.TRANSACTIONAL, NotificationPriority.HIGH,
                Map.of("title", "First"), DeliveryChannel.EMAIL
        );
        assertNotNull(first);

        // Lần 2: Trùng lặp -> Expect DuplicateEventException
        assertThrows(DuplicateEventException.class, () -> {
            applicationService.triggerNotification(
                    userId, eventId, NotificationType.TRANSACTIONAL, NotificationPriority.HIGH,
                    Map.of("title", "Second"), DeliveryChannel.EMAIL
            );
        });
    }
}
