package com.rentagf.notification.application.service;

import com.rentagf.notification.application.port.outbound.NotificationSender;
import com.rentagf.notification.application.port.outbound.SendResult;
import com.rentagf.notification.application.registry.NotificationSenderRegistry;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.repository.NotificationRepository;
import com.rentagf.notification.domain.vo.DeliveryAttempt;
import com.rentagf.notification.domain.vo.enums.*;
import com.rentagf.notification.domain.errors.DuplicateEventException;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.scheduling.annotation.Async;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.Map;
import java.util.UUID;

@Slf4j
@Service
@RequiredArgsConstructor
public class NotificationApplicationService {

    private final NotificationRepository notificationRepository;
    private final NotificationSenderRegistry senderRegistry;

    /**
     * Entry point gửi thông báo. Lưu DB ở trạng thái PENDING trước, sau đó kích hoạt luồng Async.
     */
    @Transactional
    public Notification triggerNotification(UUID userId, String eventId, NotificationType type,
                                            NotificationPriority priority, Map<String, Object> payload,
                                            DeliveryChannel channel) {
        log.info("Triggering notification for user: {}, eventId: {}, channel: {}", userId, eventId, channel);

        // Bảo vệ [INV-N03]: Kiểm tra xem eventId đã tồn tại chưa để tránh xử lý trùng lặp (Idempotency Guard)
        notificationRepository.findByEventIdAndUserId(eventId, userId).ifPresent(n -> {
            throw new DuplicateEventException(eventId, userId.toString());
        });

        Notification notification = Notification.create(userId, eventId, type, priority, payload, Map.of());
        Notification saved = notificationRepository.save(notification);

        // Kích hoạt gửi bất đồng bộ
        sendAsync(saved.getId(), channel);

        return saved;
    }

    /**
     * Gửi tin bất đồng bộ chạy trên Virtual Threads.
     */
    @Async
    public void sendAsync(UUID notificationId, DeliveryChannel channel) {
        log.info("Async sending notification: {} via channel: {}", notificationId, channel);

        Notification notification = notificationRepository.findById(notificationId)
                .orElseThrow(() -> new IllegalArgumentException("Notification not found: " + notificationId));

        DeliveryAttempt attempt = null;
        try {
            // 1. Tạo attempt mới (Chuyển sang trạng thái PROCESSING)
            attempt = notification.createAttempt(channel);
            notificationRepository.save(notification);

            // 2. Tìm Strategy Sender phù hợp
            NotificationSender sender = senderRegistry.getSender(channel)
                    .orElseThrow(() -> new IllegalArgumentException("No sender strategy found for channel: " + channel));

            // 3. Thực thi gửi
            SendResult result = sender.send(notification);

            // 4. Xử lý kết quả dựa trên Invariants & Failure Handling
            if (result.isSuccess()) {
                notification.markAttemptSuccess(attempt.getId(), result.getMessageId());
                log.info("Successfully sent notification: {} via message: {}", notificationId, result.getMessageId());
            } else {
                notification.markAttemptFailed(attempt.getId(), result.getErrorCode(), result.getErrorMessage(), result.isRecoverable());
                log.warn("Failed to send notification: {} via channel: {}. Error: {}", notificationId, channel, result.getErrorMessage());
            }

        } catch (Exception e) {
            log.error("Unexpected error during async send for notification: {}", notificationId, e);
            if (attempt != null) {
                notification.markAttemptFailed(attempt.getId(), "SYSTEM_ERROR", e.getMessage(), true);
            }
        } finally {
            notificationRepository.save(notification);
        }
    }
}
