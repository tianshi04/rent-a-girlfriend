package com.rentagf.notification.infrastructure.persistence.jpa.repository;

import com.rentagf.notification.infrastructure.persistence.jpa.entity.NotificationJpaEntity;
import org.springframework.data.domain.Limit;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Modifying;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

import java.time.Instant;
import java.util.List;
import java.util.Optional;
import java.util.UUID;

@Repository
public interface NotificationJpaRepository extends JpaRepository<NotificationJpaEntity, UUID> {

    Optional<NotificationJpaEntity> findByEventId(String eventId);

    Optional<NotificationJpaEntity> findByEventIdAndUserId(String eventId, UUID userId);

    @Query("SELECT n FROM NotificationJpaEntity n WHERE n.userId = :userId AND (:unreadOnly = false OR n.readAt IS NULL) ORDER BY n.createdAt DESC, n.id DESC")
    List<NotificationJpaEntity> findFirstPage(@Param("userId") UUID userId, @Param("unreadOnly") boolean unreadOnly, Limit limit);

    @Query("SELECT n FROM NotificationJpaEntity n WHERE n.userId = :userId AND (:unreadOnly = false OR n.readAt IS NULL) AND (n.createdAt < :cursor OR (n.createdAt = :cursor AND n.id < :cursorId)) ORDER BY n.createdAt DESC, n.id DESC")
    List<NotificationJpaEntity> findWithCursor(@Param("userId") UUID userId, @Param("cursor") Instant cursor, @Param("cursorId") UUID cursorId, @Param("unreadOnly") boolean unreadOnly, Limit limit);

    long countByUserIdAndReadAtIsNull(UUID userId);

    @Modifying
    @Query("UPDATE NotificationJpaEntity n SET n.readAt = :readAt, n.updatedAt = :now WHERE n.id = :id AND n.userId = :userId AND n.readAt IS NULL")
    int markSingleAsRead(@Param("id") UUID id, @Param("userId") UUID userId, @Param("readAt") Instant readAt, @Param("now") Instant now);

    @Modifying
    @Query("UPDATE NotificationJpaEntity n SET n.readAt = :readAt, n.updatedAt = :now WHERE n.userId = :userId AND n.readAt IS NULL")
    int markAllAsRead(@Param("userId") UUID userId, @Param("readAt") Instant readAt, @Param("now") Instant now);

    boolean existsByIdAndUserId(UUID id, UUID userId);

    List<NotificationJpaEntity> findAllByStatusAndCreatedAtBefore(String status, Instant before);
}
