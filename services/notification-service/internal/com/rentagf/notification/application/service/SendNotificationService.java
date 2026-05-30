package com.rentagf.notification.application.service;

import com.rentagf.notification.application.event.NotificationReadyEvent;
import com.rentagf.notification.application.port.inbound.SendNotificationUseCase;
import com.rentagf.notification.application.port.outbound.ConnectionStatePort;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.repository.NotificationRepository;
import com.rentagf.notification.domain.vo.enums.*;
import com.rentagf.notification.domain.errors.DuplicateEventException;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.context.ApplicationEventPublisher;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.ArrayList;
import java.util.Collection;
import java.util.Collections;
import java.util.List;

/**
 * Triển khai SendNotificationUseCase.
 * Chịu trách nhiệm duy nhất cho việc định tuyến thông minh (Smart Routing Engine)
 * và uỷ thác gửi tin bất đồng bộ.
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class SendNotificationService implements SendNotificationUseCase {

    private final NotificationRepository notificationRepository;
    private final ConnectionStatePort connectionStatePort;
    private final ApplicationEventPublisher eventPublisher;

    /**
     * Entry point gửi thông báo động (Smart Routing Engine).
     * Dựa vào cấu hình kênh trong policyOverrides và trạng thái Online để định tuyến.
     */
    @Override
    @Transactional
    public Notification routeAndSend(Notification notification) {
        log.info("Processing routing and sending for notification: {}, eventId: {}", notification.getId(), notification.getEventId());

        // 1. Idempotency Guard: Chặn xử lý trùng lặp
        notificationRepository.findByEventIdAndUserId(notification.getEventId(), notification.getUserId()).ifPresent(n -> {
            throw new DuplicateEventException(notification.getEventId(), notification.getUserId().toString());
        });

        // 2. Lưu trạng thái PENDING ban đầu vào Database
        Notification saved = notificationRepository.save(notification);

        // 3. Trích xuất danh sách kênh cấu hình từ policyOverrides (an toàn với cả Java List & Scala List)
        List<String> channels = extractChannels(saved);
        if (channels.isEmpty()) {
            log.warn("No channels configured for notification: {}. Skipping delivery.", saved.getId());
            return saved;
        }

        boolean sentViaSse = false;

        // 4. Kênh SSE: Chỉ gửi nếu User đang Online
        if (channels.contains("SSE")) {
            if (connectionStatePort.isOnline(saved.getUserId())) {
                log.info("User {} is ONLINE in cluster. Dispatching SSE delivery.", saved.getUserId());
                eventPublisher.publishEvent(new NotificationReadyEvent(saved.getId(), DeliveryChannel.SSE));
                sentViaSse = true;
            } else {
                log.info("User {} is OFFLINE in cluster. Skipping SSE delivery.", saved.getUserId());
            }
        }

        // 5. Kênh FCM: Gửi nếu cấu hình có FCM và (User Offline HOẶC có độ ưu tiên cao HIGH)
        if (channels.contains("FCM")) {
            boolean shouldSendFcm = !sentViaSse || saved.getPriority() == NotificationPriority.HIGH;
            if (shouldSendFcm) {
                log.info("Dispatching FCM push delivery for user: {}", saved.getUserId());
                eventPublisher.publishEvent(new NotificationReadyEvent(saved.getId(), DeliveryChannel.FCM));
            }
        }

        // 6. Kênh EMAIL: Gửi song song bất đồng bộ nếu có cấu hình EMAIL
        if (channels.contains("EMAIL")) {
            log.info("Dispatching Email delivery for user: {}", saved.getUserId());
            eventPublisher.publishEvent(new NotificationReadyEvent(saved.getId(), DeliveryChannel.EMAIL));
        }

        return saved;
    }

    /**
     * Trích xuất danh sách channels an toàn từ policyOverrides, tránh lỗi ép kiểu trong
     * trường hợp Jackson deserialize JSON array thành Scala List (do classpath test có Scala).
     */
    @SuppressWarnings("unchecked")
    private List<String> extractChannels(Notification notification) {
        Object channelsObj = notification.getPolicyOverrides().get("channels");
        if (channelsObj == null) {
            return Collections.emptyList();
        }
        if (channelsObj instanceof List) {
            List<?> rawList = (List<?>) channelsObj;
            if (rawList.isEmpty()) {
                return Collections.emptyList();
            }
            try {
                List<String> list = new ArrayList<>();
                for (Object o : rawList) {
                    if (o != null) {
                        list.add(o.toString());
                    }
                }
                return list;
            } catch (Exception e) {
                // fallback
            }
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
            log.error("Failed to convert channels object: {}", channelsObj, e);
            return Collections.emptyList();
        }
    }
}
