package com.rentagf.notification.application.service;

import com.rentagf.notification.application.event.NotificationReadyEvent;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.repository.NotificationRepository;
import com.rentagf.notification.domain.vo.DeliveryAttempt;
import com.rentagf.notification.domain.vo.enums.DeliveryChannel;
import com.rentagf.notification.domain.vo.enums.NotificationStatus;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.boot.context.event.ApplicationReadyEvent;
import org.springframework.context.ApplicationEventPublisher;
import org.springframework.context.event.EventListener;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;

import java.time.Duration;
import java.time.Instant;
import java.util.ArrayList;
import java.util.Collection;
import java.util.List;

/**
 * Service Self-Healing (Tự động lành lặn) đảm bảo tính tin cậy tuyệt đối của hệ thống thông báo.
 * Định kỳ quét DB và lắng nghe sự kiện Startup để khôi phục các thông báo bị kẹt ở trạng thái PROCESSING
 * (ví dụ do server đột ngột bị sập, restart hoặc mất điện).
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class NotificationSelfHealingJob {

    private final NotificationRepository notificationRepository;
    private final ApplicationEventPublisher eventPublisher;

    private static final long STUCK_THRESHOLD_MINUTES = 5L;

    /**
     * Tự động khôi phục các thông báo bị kẹt ngay khi ứng dụng khởi chạy thành công (Startup Recovery).
     */
    @EventListener(ApplicationReadyEvent.class)
    public void onApplicationReady() {
        log.info("[SELF-HEALING] Application started. Executing startup recovery job for stuck notifications...");
        executeRecovery();
    }

    /**
     * Định kỳ quét DB tìm các thông báo bị kẹt mỗi 2 phút (định cấu hình qua property).
     */
    @Scheduled(fixedDelayString = "${notification.self-healing.fixed-delay-ms:120000}")
    public void scheduleRecovery() {
        log.debug("[SELF-HEALING] Running periodic background recovery job...");
        executeRecovery();
    }

    /**
     * Logic khôi phục cốt lõi:
     * Quét các Notification ở trạng thái PROCESSING được tạo trước 5 phút mà chưa hoàn tất.
     */
    private void executeRecovery() {
        try {
            Instant thresholdTime = Instant.now().minus(Duration.ofMinutes(STUCK_THRESHOLD_MINUTES));
            List<Notification> stuckNotifications = notificationRepository.findAllByStatusAndCreatedAtBefore(
                    NotificationStatus.PROCESSING, thresholdTime
            );

            if (stuckNotifications.isEmpty()) {
                log.debug("[SELF-HEALING] No stuck notifications detected.");
                return;
            }

            log.info("[SELF-HEALING] Detected {} stuck notifications created before {}!", stuckNotifications.size(), thresholdTime);

            for (Notification notification : stuckNotifications) {
                try {
                    // Xác định kênh gửi tin phù hợp để trigger lại
                    DeliveryChannel channel = determineDeliveryChannel(notification);
                    
                    log.info("[SELF-HEALING] Re-dispatching stuck notification {} via channel {}...", 
                            notification.getId(), channel);
                    
                    // Phát hành sự kiện để luồng bất đồng bộ nhặt và gửi lại
                    eventPublisher.publishEvent(new NotificationReadyEvent(notification.getId(), channel));

                } catch (Exception ex) {
                    log.error("[SELF-HEALING] Failed to recover stuck notification: {}", notification.getId(), ex);
                }
            }

        } catch (Exception e) {
            log.error("[SELF-HEALING] Unexpected error during execution of self-healing job", e);
        }
    }

    /**
     * Xác định kênh gửi thích hợp để phục hồi thông báo bị kẹt:
     * 1. Ưu tiên lấy kênh của nỗ lực gửi (attempt) gần nhất.
     * 2. Nếu chưa có attempt nào, lấy kênh đầu tiên trong danh sách cấu hình của policyOverrides.
     * 3. Fallback mặc định sang FCM.
     */
    private DeliveryChannel determineDeliveryChannel(Notification notification) {
        List<DeliveryAttempt> attempts = notification.getAttempts();
        if (attempts != null && !attempts.isEmpty()) {
            // Lấy attempt cuối cùng
            DeliveryAttempt lastAttempt = attempts.get(attempts.size() - 1);
            return lastAttempt.getChannel();
        }

        // Đọc từ policyOverrides
        List<String> configuredChannels = extractChannels(notification);
        if (!configuredChannels.isEmpty()) {
            try {
                return DeliveryChannel.valueOf(configuredChannels.get(0).toUpperCase());
            } catch (Exception e) {
                log.warn("[SELF-HEALING] Failed to parse configured channel {} for notification {}", 
                        configuredChannels.get(0), notification.getId());
            }
        }

        // Default fallback
        return DeliveryChannel.FCM;
    }

    /**
     * Trích xuất các kênh cấu hình an toàn từ policyOverrides.
     */
    @SuppressWarnings("unchecked")
    private List<String> extractChannels(Notification notification) {
        Object channelsObj = notification.getPolicyOverrides().get("channels");
        if (channelsObj == null) {
            return List.of();
        }
        if (channelsObj instanceof List) {
            List<?> rawList = (List<?>) channelsObj;
            List<String> list = new ArrayList<>();
            for (Object o : rawList) {
                if (o != null) {
                    list.add(o.toString());
                }
            }
            return list;
        }
        if (channelsObj instanceof Collection) {
            Collection<?> col = (Collection<?>) channelsObj;
            List<String> list = new ArrayList<>();
            for (Object o : col) {
                if (o != null) {
                    list.add(o.toString());
                }
            }
            return list;
        }
        try {
            return new com.fasterxml.jackson.databind.ObjectMapper()
                    .convertValue(channelsObj, new com.fasterxml.jackson.core.type.TypeReference<List<String>>() {});
        } catch (Exception e) {
            return List.of();
        }
    }
}
