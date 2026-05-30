package com.rentagf.notification.application;

import com.rentagf.notification.application.port.outbound.ConnectionStatePort;
import com.rentagf.notification.application.port.outbound.EmailPort;
import com.rentagf.notification.application.port.outbound.FcmPort;
import com.rentagf.notification.application.port.outbound.PubSubPort;
import com.rentagf.notification.application.port.outbound.SendResult;
import com.rentagf.notification.application.port.outbound.SsePort;
import com.rentagf.notification.application.port.inbound.SendNotificationUseCase;
import com.rentagf.notification.application.port.inbound.TriggerNotificationUseCase;
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

/**
 * Integration test kiểm chứng luồng gửi thông báo bất đồng bộ trên Virtual Threads.
 *
 * <p>Sử dụng H2 in-memory (cấu hình trong tests/application.yml) và Mockito Beans
 * để thay thế toàn bộ hạ tầng ngoài (Redis, Kafka, SMTP, FCM).
 */
@SpringBootTest
@Tag("integration")
class AsyncDeliveryTest {

    @Autowired
    private SendNotificationUseCase applicationService;

    @Autowired
    private TriggerNotificationUseCase triggerService;

    @Autowired
    private NotificationRepository notificationRepository;

    // --- Mock toàn bộ hạ tầng ngoài ---

    /** Thay thế RedisConnectionStateAdapter – không cần Redis khi test. */
    @MockitoBean
    private ConnectionStatePort connectionStatePort;

    /** Thay thế RedisPubSubAdapter – không cần StringRedisTemplate khi test. */
    @MockitoBean
    private PubSubPort pubSubPort;

    @MockitoBean
    private EmailPort emailPort;

    @MockitoBean
    private FcmPort fcmPort;

    @MockitoBean
    private SsePort ssePort;

    @BeforeEach
    void setUp() {
        // Stub channel để NotificationSenderRegistry map đúng strategy
        when(emailPort.getChannel()).thenReturn(DeliveryChannel.EMAIL);
        when(fcmPort.getChannel()).thenReturn(DeliveryChannel.FCM);
        when(ssePort.getChannel()).thenReturn(DeliveryChannel.SSE);

        // Mặc định user offline – đảm bảo SendNotificationService không dispatch SSE
        when(connectionStatePort.isOnline(any(UUID.class))).thenReturn(false);
    }

    @Test
    void testSuccessfulAsyncDelivery() {
        UUID userId = UUID.randomUUID();
        String eventId = "evt_async_success_" + UUID.randomUUID();

        // Stub EmailPort trả về thành công
        when(emailPort.send(any(Notification.class)))
                .thenReturn(SendResult.success("msg_email_ok"));

        // Kích hoạt gửi thông báo
        Notification triggered = triggerService.triggerNotification(
                userId, eventId, NotificationType.TRANSACTIONAL, NotificationPriority.HIGH,
                Map.of("title", "Hello Async"), DeliveryChannel.EMAIL
        );

        assertNotNull(triggered);

        // Awaitility: chờ Virtual Thread gửi xong và cập nhật DB
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

        // Stub FcmPort trả về lỗi không phục hồi được
        when(fcmPort.send(any(Notification.class)))
                .thenReturn(SendResult.fail("FCM_TOKEN_INVALID", "Token is dead", false));

        Notification triggered = triggerService.triggerNotification(
                userId, eventId, NotificationType.TRANSACTIONAL, NotificationPriority.HIGH,
                Map.of("title", "Hello FCM"), DeliveryChannel.FCM
        );

        assertNotNull(triggered);

        await().atMost(Duration.ofSeconds(5))
                .untilAsserted(() -> {
                    Notification found = notificationRepository.findById(triggered.getId()).orElse(null);
                    assertNotNull(found);
                    // Unrecoverable → đóng FAILED ngay lập tức
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
        Notification first = triggerService.triggerNotification(
                userId, eventId, NotificationType.TRANSACTIONAL, NotificationPriority.HIGH,
                Map.of("title", "First"), DeliveryChannel.EMAIL
        );
        assertNotNull(first);

        // Lần 2: Cùng eventId + userId → phải throw DuplicateEventException
        assertThrows(DuplicateEventException.class, () ->
                triggerService.triggerNotification(
                        userId, eventId, NotificationType.TRANSACTIONAL, NotificationPriority.HIGH,
                        Map.of("title", "Second"), DeliveryChannel.EMAIL
                )
        );
    }
}
