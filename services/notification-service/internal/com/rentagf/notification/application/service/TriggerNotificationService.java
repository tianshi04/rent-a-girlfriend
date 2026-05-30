package com.rentagf.notification.application.service;

import com.rentagf.notification.application.event.NotificationReadyEvent;
import com.rentagf.notification.application.port.inbound.TriggerNotificationUseCase;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.repository.NotificationRepository;
import com.rentagf.notification.domain.vo.enums.*;
import com.rentagf.notification.domain.errors.DuplicateEventException;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.context.ApplicationEventPublisher;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;
import java.util.Map;
import java.util.UUID;

/**
 * Triển khai TriggerNotificationUseCase.
 * Chịu trách nhiệm duy nhất cho việc gửi thông báo trực tiếp qua kênh chỉ định sẵn (không qua routing động).
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class TriggerNotificationService implements TriggerNotificationUseCase {

    private final NotificationRepository notificationRepository;
    private final ApplicationEventPublisher eventPublisher;

    /**
     * Entry point gửi thông báo thủ công chỉ định sẵn kênh.
     */
    @Override
    @Transactional
    public Notification triggerNotification(UUID userId, String eventId, NotificationType type,
                                             NotificationPriority priority, Map<String, Object> payload,
                                             DeliveryChannel channel) {
        log.info("Triggering manual notification for user: {}, eventId: {}, channel: {}", userId, eventId, channel);

        // 1. Idempotency Guard: Chặn xử lý trùng lặp
        notificationRepository.findByEventIdAndUserId(eventId, userId).ifPresent(n -> {
            throw new DuplicateEventException(eventId, userId.toString());
        });

        // 2. Tạo Notification aggregate trực tiếp gán kênh vào policy
        Notification notification = Notification.create(userId, eventId, type, priority, payload, Map.of("channels", List.of(channel.name())));
        Notification saved = notificationRepository.save(notification);

        // 3. Kích hoạt gửi bất đồng bộ qua Event
        eventPublisher.publishEvent(new NotificationReadyEvent(saved.getId(), channel));

        return saved;
    }
}
