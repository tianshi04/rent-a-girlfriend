package com.rentagf.notification.application.service;

import com.rentagf.notification.application.port.inbound.NotificationSubscriptionUseCase;
import com.rentagf.notification.application.port.outbound.SsePort;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.repository.NotificationRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;
import java.util.UUID;

/**
 * Triển khai nghiệp vụ đăng ký nhận thông báo SSE thông qua Inbound Port.
 * Đảm bảo Single Responsibility Principle (SRP) tuyệt đối bằng cách phân tách UseCase độc lập.
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class NotificationSubscriptionService implements NotificationSubscriptionUseCase {

    private final NotificationRepository notificationRepository;
    private final SsePort ssePort;

    /**
     * Khi Client đăng ký kết nối SSE thành công, tự động truy vấn các thông báo chưa đọc
     * cũ từ DB và chủ động đẩy xuống cho Client để đảm bảo không bị mất mát tin nhắn (Inbox Delivery).
     */
    @Override
    @Transactional(readOnly = true)
    public void subscribe(UUID userId) {
        log.info("Processing subscription use case for user: {}", userId);

        // Lấy tối đa 100 thông báo chưa đọc của user
        List<Notification> unreadNotifications = notificationRepository.findByUserId(userId, null, null, 100, true);
        log.info("Found {} unread notifications to push via SSE for user: {}", unreadNotifications.size(), userId);

        for (Notification notification : unreadNotifications) {
            try {
                log.debug("Pushing unread notification {} to user {} via SSE", notification.getId(), userId);
                ssePort.send(notification);
            } catch (Exception e) {
                log.error("Failed to push unread notification {} to user {} via SSE during subscription",
                        notification.getId(), userId, e);
            }
        }
    }
}
