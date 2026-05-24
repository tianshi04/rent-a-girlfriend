package com.rentagf.notification.application.service;

import com.rentagf.notification.application.port.inbound.FetchInboxUseCase;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.repository.NotificationRepository;
import com.rentagf.notification.domain.vo.InboxCursor;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;
import java.util.UUID;

/**
 * Service triển khai Use Case FetchInboxUseCase.
 * Đảm bảo transaction read-only để tối ưu hóa hiệu suất PostgreSQL.
 */
@Service
@RequiredArgsConstructor
@Transactional(readOnly = true)
public class FetchInboxService implements FetchInboxUseCase {

    private final NotificationRepository notificationRepository;

    @Override
    public List<Notification> fetchInbox(UUID userId, InboxCursor cursor, int limit, boolean unreadOnly) {
        // Query Repository với kích thước limit + 1 để phục vụ tính toán hasMore ở tầng Controller
        if (cursor == null) {
            return notificationRepository.findByUserId(userId, null, null, limit + 1, unreadOnly);
        } else {
            return notificationRepository.findByUserId(
                userId,
                cursor.createdAt(),
                cursor.id(),
                limit + 1,
                unreadOnly
            );
        }
    }
}
