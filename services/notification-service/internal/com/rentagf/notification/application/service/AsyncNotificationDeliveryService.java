package com.rentagf.notification.application.service;

import com.rentagf.notification.application.event.NotificationReadyEvent;
import com.rentagf.notification.application.port.outbound.NotificationSender;
import com.rentagf.notification.application.port.outbound.RetrySchedulerPort;
import com.rentagf.notification.application.port.outbound.SendResult;
import com.rentagf.notification.application.registry.NotificationSenderRegistry;
import com.rentagf.notification.application.service.NotificationDeliveryTransactionService.DeliveryActionResult;
import com.rentagf.notification.application.service.NotificationDeliveryTransactionService.DeliveryContext;
import com.rentagf.notification.domain.vo.enums.DeliveryChannel;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.context.ApplicationEventPublisher;
import org.springframework.scheduling.annotation.Async;
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.util.UUID;

/**
 * Service chịu trách nhiệm gửi tin vật lý bất đồng bộ (@Async) chạy trên Virtual Threads.
 * Áp dụng thiết kế "Hikari Connection Exhaustion Guard" bằng cách loại bỏ hoàn toàn Database Transaction 
 * khỏi thời gian Virtual Thread block chờ các API mạng ngoại vi phản hồi.
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class AsyncNotificationDeliveryService {

    private final NotificationSenderRegistry senderRegistry;
    private final NotificationDeliveryTransactionService txService;
    private final RetrySchedulerPort retrySchedulerPort;
    private final ApplicationEventPublisher eventPublisher;

    /**
     * Gửi tin bất đồng bộ chạy trên Virtual Threads.
     * TUYỆT ĐỐI không có @Transactional ở đây để giải phóng Connection DB trong suốt thời gian block mạng.
     *
     * @param notificationId ID của Notification.
     * @param channel        Kênh gửi tin vật lý.
     */
    @Async
    public void sendAsync(UUID notificationId, DeliveryChannel channel) {
        log.info("[DELIVERY CORE] Async sending notification: {} via channel: {} inside thread: {}", 
                notificationId, channel, Thread.currentThread());

        // 1. BƯỚC 1: Transaction ngắn 1 - Chuẩn bị Attempt và lưu trạng thái PROCESSING
        DeliveryContext context;
        try {
            context = txService.prepareAttempt(notificationId, channel);
        } catch (Exception e) {
            log.error("[DELIVERY CORE] Failed to prepare attempt for notification: {}", notificationId, e);
            return;
        }

        // 2. BƯỚC 2: Ngoài Transaction - Thực thi gửi mạng I/O (Virtual Thread block chờ thoải mái, HikariCP hoàn toàn rảnh)
        SendResult result;
        try {
            NotificationSender sender = senderRegistry.getSender(channel)
                    .orElseThrow(() -> new IllegalArgumentException("No sender strategy found for channel: " + channel));

            log.info("[DELIVERY CORE] Executing physical delivery via Outbound Adapter for channel: {}", channel);
            result = sender.send(context.getNotification());

        } catch (Exception e) {
            log.error("[DELIVERY CORE] Unexpected error during adapter delivery for notification: {}", notificationId, e);
            // Coi các lỗi ngoại lệ hệ thống bất ngờ là recoverable để cho phép retry tự phục hồi
            result = SendResult.fail("ADAPTER_SYSTEM_ERROR", e.getMessage(), true);
        }

        // 3. BƯỚC 3: Transaction ngắn 2 - Lưu kết quả gửi tin và đưa ra quyết định hành động tiếp theo
        DeliveryActionResult actionResult;
        try {
            actionResult = txService.handleResult(notificationId, context.getAttemptId(), result, channel);
        } catch (Exception e) {
            log.error("[DELIVERY CORE] Failed to handle result in database for notification: {}", notificationId, e);
            return;
        }

        // 4. BƯỚC 4: Ngoài Transaction - Thực thi các Side-effects (Fallback hoặc Retry)
        if (actionResult.isShouldFallbackToFcm()) {
            log.info("[FALLBACK TRIGGER] SSE failed for notification {}. Automatically fallback to FCM Push asynchronously.", notificationId);
            eventPublisher.publishEvent(new NotificationReadyEvent(notificationId, DeliveryChannel.FCM));
            
        } else if (actionResult.isShouldRetry()) {
            Duration delay = calculateBackoffDelay(channel, actionResult.getFailedCount());
            log.info("[RETRY TRIGGER] Scheduling retry for notification {} via channel {} after {}s (Failed attempt #{})",
                    notificationId, channel, delay.toSeconds(), actionResult.getFailedCount());
            
            retrySchedulerPort.scheduleRetry(notificationId, channel, delay);
        }
    }

    /**
     * Tính toán thời gian delay Exponential Backoff tăng dần theo đặc tả kỹ thuật:
     * - FCM: 2s -> 4s -> 8s
     * - Email: 5s -> 15s -> 45s
     */
    private Duration calculateBackoffDelay(DeliveryChannel channel, int failedCount) {
        if (channel == DeliveryChannel.FCM) {
            return switch (failedCount) {
                case 1 -> Duration.ofSeconds(2);
                case 2 -> Duration.ofSeconds(4);
                default -> Duration.ofSeconds(8);
            };
        } else { // EMAIL
            return switch (failedCount) {
                case 1 -> Duration.ofSeconds(5);
                case 2 -> Duration.ofSeconds(15);
                default -> Duration.ofSeconds(45);
            };
        }
    }
}
