package com.rentagf.notification.domain.repository;

import com.rentagf.notification.domain.aggregate.Notification;

import java.time.Instant;
import java.util.List;
import java.util.Optional;
import java.util.UUID;

/**
 * Repository Port — ranh giới giữa Domain và Infrastructure.
 * Domain chỉ biết interface này, không biết JPA/SQL.
 * Tham chiếu: docs/data-model.md
 */
public interface NotificationRepository {

    Notification save(Notification notification);

    Optional<Notification> findById(UUID id);

    /**
     * Tìm notification theo eventId + userId (dùng cho Idempotency Guard [INV-N03]).
     */
    Optional<Notification> findByEventIdAndUserId(String eventId, UUID userId);

    /**
     * Inbox query: cursor-based pagination cho userId.
     * Tham chiếu: docs/api-contract.md §2.1, ADR-0004
     *
     * @param userId      ID người dùng
     * @param cursor      cursor timestamp (null = page đầu)
     * @param cursorId    cursor UUID (null = page đầu)
     * @param limit       số bản ghi tối đa
     * @param unreadOnly  chỉ lấy chưa đọc
     * @return danh sách notifications sắp xếp theo created_at DESC
     */
    List<Notification> findByUserId(UUID userId, Instant cursor, UUID cursorId, int limit, boolean unreadOnly);

    /**
     * Đếm số thông báo chưa đọc của user.
     */
    long countUnreadByUserId(UUID userId);

    /**
     * Mark read: set read_at cho 1 notification.
     */
    void markAsRead(UUID notificationId, Instant readAt);

    /**
     * Mark all read: set read_at cho tất cả notifications chưa đọc của user.
     */
    void markAllAsRead(UUID userId, Instant readAt);
}
