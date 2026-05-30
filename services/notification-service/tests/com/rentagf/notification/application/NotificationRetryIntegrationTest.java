package com.rentagf.notification.application;

import com.rentagf.notification.application.port.inbound.TriggerNotificationUseCase;
import com.rentagf.notification.application.port.outbound.ConnectionStatePort;
import com.rentagf.notification.application.port.outbound.EmailPort;
import com.rentagf.notification.application.port.outbound.PubSubPort;
import com.rentagf.notification.application.port.outbound.SendResult;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.repository.NotificationRepository;
import com.rentagf.notification.domain.vo.enums.*;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.mockito.stubbing.Answer;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.bean.override.mockito.MockitoBean;

import java.time.Duration;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.atomic.AtomicInteger;

import static org.awaitility.Awaitility.await;
import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;

/**
 * Integration Test End-to-End kiểm chứng luồng Tự động Retry trì hoãn bất đồng bộ.
 *
 * <p>Mô phỏng:
 * Lần 1: SMTP offline -> Gửi lỗi recoverable -> DB ghi nhận FAILED_RECOVERABLE, Notification giữ PROCESSING.
 * Lên lịch retry trì hoãn (Backoff).
 * Lần 2: SMTP online -> Gửi thành công -> DB ghi nhận SUCCESS, Notification chuyển sang COMPLETED.
 */
@SpringBootTest
@Tag("integration")
class NotificationRetryIntegrationTest {

    @Autowired
    private TriggerNotificationUseCase triggerService;

    @Autowired
    private NotificationRepository notificationRepository;

    @MockitoBean
    private ConnectionStatePort connectionStatePort;

    @MockitoBean
    private PubSubPort pubSubPort;

    @MockitoBean
    private EmailPort emailPort;

    @BeforeEach
    void setUp() {
        when(emailPort.getChannel()).thenReturn(DeliveryChannel.EMAIL);
        when(connectionStatePort.isOnline(any(UUID.class))).thenReturn(false);
    }

    @Test
    void testEndToEnd_RetryUntilSuccess() {
        UUID userId = UUID.randomUUID();
        String eventId = "evt_e2e_retry_" + UUID.randomUUID();

        // Đếm số lần gọi send của emailPort
        AtomicInteger sendAttempts = new AtomicInteger(0);

        // Giả lập: Lần 1 lỗi Recoverable, Lần 2 thành công
        when(emailPort.send(any(Notification.class))).thenAnswer((Answer<SendResult>) invocation -> {
            int attemptNum = sendAttempts.incrementAndGet();
            if (attemptNum == 1) {
                // Thất bại tạm thời
                return SendResult.fail("SMTP_TIMEOUT", "SMTP server is temporary down", true);
            } else {
                // Thử lại thành công
                return SendResult.success("msg_retry_success_999");
            }
        });

        // 1. Kích hoạt gửi thông báo (Kênh Email có initial-delay-ms là 5s, nhưng ta test in-memory cực nhanh)
        // Vì class cấu hình inmemory retry có thể được test với delay nhỏ, 
        // Tuy nhiên, để test chạy nhanh hơn, ta kiểm tra cấu hình trong application.yml có initial-delay-ms hay không.
        // Ta trigger gửi thông báo
        Notification triggered = triggerService.triggerNotification(
                userId, eventId, NotificationType.TRANSACTIONAL, NotificationPriority.HIGH,
                Map.of("title", "Test E2E Retry"), DeliveryChannel.EMAIL
        );

        assertNotNull(triggered);

        // 2. Awaitility: Đợi luồng thứ nhất chạy xong -> attempt 1 thất bại -> DB lưu FAILED_RECOVERABLE
        await().atMost(Duration.ofSeconds(3))
                .untilAsserted(() -> {
                    Notification found = notificationRepository.findById(triggered.getId()).orElse(null);
                    assertNotNull(found);
                    // Lần đầu thất bại nhưng recoverable -> status Notification vẫn là PROCESSING để chờ retry
                    assertEquals(NotificationStatus.PROCESSING, found.getStatus());
                    assertTrue(found.getAttempts().size() >= 1);
                    assertEquals(AttemptStatus.FAILED_RECOVERABLE, found.getAttempts().get(0).getStatus());
                });

        // 3. Awaitility: Đợi scheduler kích hoạt lại lần 2 (Sau 5 giây backoff của Email) và gửi thành công
        // Ta set thời gian chờ tối đa là 8 giây ( Email delay lần đầu là 5s )
        await().atMost(Duration.ofSeconds(10))
                .untilAsserted(() -> {
                    Notification found = notificationRepository.findById(triggered.getId()).orElse(null);
                    assertNotNull(found);
                    // Lần 2 thành công -> status Notification chuyển sang COMPLETED
                    assertEquals(NotificationStatus.COMPLETED, found.getStatus());
                    assertEquals(2, found.getAttempts().size());
                    assertEquals(AttemptStatus.FAILED_RECOVERABLE, found.getAttempts().get(0).getStatus());
                    assertEquals(AttemptStatus.SUCCESS, found.getAttempts().get(1).getStatus());
                    assertEquals("msg_retry_success_999", found.getAttempts().get(1).getMessageId());
                });

        // Verify emailPort thực sự được gọi 2 lần
        verify(emailPort, times(2)).send(any(Notification.class));
    }
}
