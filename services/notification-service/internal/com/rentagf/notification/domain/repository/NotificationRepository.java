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
     * Mark read: set read_at cho 1 notification của user cụ thể.
     * Sử dụng Optimistic Update để tối ưu số database round-trips.
     *
     * @param notificationId ID thông báo
     * @param userId         ID người dùng sở hữu (ngăn chặn tấn công BOLA)
     * @param readAt         Thời điểm đọc
     * @return true nếu cập nhật thành công (hoặc thông báo đã được đọc từ trước - Idempotent),
     *         false nếu thông báo không tồn tại hoặc không thuộc về người dùng này.
     */
    boolean markAsRead(UUID notificationId, UUID userId, Instant readAt);

    /**
     * Mark all read: set read_at cho tất cả notifications chưa đọc của user.
     *
     * @param userId ID người dùng
     * @param readAt Thời điểm đọc
     * @return số lượng thông báo đã được chuyển trạng thái sang đã đọc thành công
     */
    int markAllAsRead(UUID userId, Instant readAt);

    /**
     * Tìm tất cả notifications theo status và được tạo trước mốc thời gian (dùng cho Self-healing).
     */
    List<Notification> findAllByStatusAndCreatedAtBefore(com.rentagf.notification.domain.vo.enums.NotificationStatus status, Instant before);
}
