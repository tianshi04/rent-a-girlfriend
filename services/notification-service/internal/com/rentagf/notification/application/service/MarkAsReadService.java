package com.rentagf.notification.application.service;

import com.rentagf.notification.application.port.inbound.MarkAsReadUseCase;
import com.rentagf.notification.domain.errors.NotificationNotFoundException;
import com.rentagf.notification.domain.repository.NotificationRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.Instant;
import java.util.UUID;

/**
 * Service triển khai Use Case MarkAsReadUseCase.
 * Tận dụng Optimistic Update từ Repository để tối ưu hóa IO và ngăn chặn IDOR/BOLA.
 */
@Service
@RequiredArgsConstructor
@Transactional
public class MarkAsReadService implements MarkAsReadUseCase {

    private final NotificationRepository notificationRepository;

    @Override
    public void markAsRead(UUID notificationId, UUID userId) {
        if (notificationId == null) {
            throw new IllegalArgumentException("notificationId cannot be null");
        }
        if (userId == null) {
            throw new IllegalArgumentException("userId cannot be null");
        }

        // Gọi Repository thực hiện đánh dấu đã đọc (Optimistic Update)
        boolean success = notificationRepository.markAsRead(notificationId, userId, Instant.now());

        if (!success) {
            // Nếu trả về false: chứng tỏ thông báo không tồn tại hoặc không thuộc quyền sở hữu của user (ngăn BOLA)
            throw new NotificationNotFoundException(notificationId.toString());
        }
    }
}
