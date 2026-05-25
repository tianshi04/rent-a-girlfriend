package com.rentagf.notification.infrastructure.adapter;

import com.rentagf.notification.application.event.NotificationReadyEvent;
import com.rentagf.notification.application.port.outbound.RetrySchedulerPort;
import com.rentagf.notification.domain.vo.enums.DeliveryChannel;
import lombok.extern.slf4j.Slf4j;
import org.springframework.context.ApplicationEventPublisher;
import org.springframework.context.annotation.Profile;
import org.springframework.stereotype.Component;

import java.time.Duration;
import java.util.UUID;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

/**
 * Adapter triển khai InMemory Retry sử dụng Java ScheduledExecutorService.
 * Phù hợp hoàn hảo cho môi trường Local Development, Testing và Demo (Profile "!prod").
 * Khi hết thời gian trì hoãn, sự kiện NotificationReadyEvent được phát đi để kích hoạt lại luồng gửi.
 */
@Slf4j
@Component
@Profile("!prod")
public class InMemoryRetryScheduler implements RetrySchedulerPort {

    // Khởi tạo executor scheduler chuyên biệt, sử dụng Virtual Threads làm các workers chạy khi kích hoạt
    private final ScheduledExecutorService scheduler = Executors.newSingleThreadScheduledExecutor(
            r -> {
                Thread t = new Thread(r, "in-memory-retry-scheduler");
                t.setDaemon(true);
                return t;
            }
    );

    private final ApplicationEventPublisher eventPublisher;

    public InMemoryRetryScheduler(ApplicationEventPublisher eventPublisher) {
        this.eventPublisher = eventPublisher;
    }

    @Override
    public void scheduleRetry(UUID notificationId, DeliveryChannel channel, Duration delay) {
        log.info("[RETRY ENGINE] Scheduling retry for notification {} via channel {} in {}s",
                notificationId, channel, delay.toSeconds());

        scheduler.schedule(() -> {
            try {
                log.info("[RETRY ENGINE] Timer expired for notification {}. Re-dispatching event via channel {}",
                        notificationId, channel);
                // Phát hành sự kiện NotificationReadyEvent bất đồng bộ để kích hoạt lại Async worker gửi tin
                eventPublisher.publishEvent(new NotificationReadyEvent(notificationId, channel));
            } catch (Exception e) {
                log.error("[RETRY ENGINE] Failed to dispatch retry event for notification {}", notificationId, e);
            }
        }, delay.toMillis(), TimeUnit.MILLISECONDS);
    }
}
