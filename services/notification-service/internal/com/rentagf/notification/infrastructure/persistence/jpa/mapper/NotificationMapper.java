package com.rentagf.notification.infrastructure.persistence.jpa.mapper;

import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.vo.DeliveryAttempt;
import com.rentagf.notification.domain.vo.enums.*;
import com.rentagf.notification.infrastructure.persistence.jpa.entity.DeliveryAttemptJpaEntity;
import com.rentagf.notification.infrastructure.persistence.jpa.entity.NotificationJpaEntity;
import org.springframework.stereotype.Component;

import java.util.ArrayList;
import java.util.List;
import java.util.stream.Collectors;

@Component
public class NotificationMapper {

    public NotificationJpaEntity toJpaEntity(Notification domain) {
        if (domain == null) {
            return null;
        }

        NotificationJpaEntity jpaEntity = NotificationJpaEntity.builder()
                .id(domain.getId())
                .userId(domain.getUserId())
                .eventId(domain.getEventId())
                .type(domain.getType().name())
                .priority(domain.getPriority().name())
                .payload(domain.getPayload())
                .policyOverrides(domain.getPolicyOverrides())
                .status(domain.getStatus().name())
                .readAt(domain.getReadAt())
                .createdAt(domain.getCreatedAt())
                .updatedAt(domain.getUpdatedAt())
                .build();

        if (domain.getAttempts() != null) {
            List<DeliveryAttemptJpaEntity> attemptJpaEntities = domain.getAttempts().stream()
                    .map(attempt -> toAttemptJpaEntity(attempt, jpaEntity))
                    .collect(Collectors.toList());
            jpaEntity.setAttempts(attemptJpaEntities);
        }

        return jpaEntity;
    }

    public Notification toDomain(NotificationJpaEntity jpaEntity) {
        if (jpaEntity == null) {
            return null;
        }

        List<DeliveryAttempt> domainAttempts = new ArrayList<>();
        if (jpaEntity.getAttempts() != null) {
            domainAttempts = jpaEntity.getAttempts().stream()
                    .map(this::toAttemptDomain)
                    .collect(Collectors.toList());
        }

        return new Notification(
                jpaEntity.getId(),
                jpaEntity.getUserId(),
                jpaEntity.getEventId(),
                NotificationType.valueOf(jpaEntity.getType()),
                NotificationPriority.valueOf(jpaEntity.getPriority()),
                jpaEntity.getPayload(),
                jpaEntity.getPolicyOverrides(),
                NotificationStatus.valueOf(jpaEntity.getStatus()),
                jpaEntity.getReadAt(),
                jpaEntity.getCreatedAt(),
                jpaEntity.getUpdatedAt(),
                domainAttempts
        );
    }

    private DeliveryAttemptJpaEntity toAttemptJpaEntity(DeliveryAttempt attempt, NotificationJpaEntity notificationJpaEntity) {
        if (attempt == null) {
            return null;
        }

        return DeliveryAttemptJpaEntity.builder()
                .id(attempt.getId())
                .notification(notificationJpaEntity)
                .channel(attempt.getChannel().name())
                .status(attempt.getStatus().name())
                .messageId(attempt.getMessageId())
                .errorCode(attempt.getErrorCode())
                .errorMessage(attempt.getErrorMessage())
                .attemptedAt(attempt.getAttemptedAt())
                .resolvedAt(attempt.getResolvedAt())
                .build();
    }

    private DeliveryAttempt toAttemptDomain(DeliveryAttemptJpaEntity jpaEntity) {
        if (jpaEntity == null) {
            return null;
        }

        return new DeliveryAttempt(
                jpaEntity.getId(),
                jpaEntity.getNotification().getId(),
                DeliveryChannel.valueOf(jpaEntity.getChannel()),
                AttemptStatus.valueOf(jpaEntity.getStatus()),
                jpaEntity.getMessageId(),
                jpaEntity.getErrorCode(),
                jpaEntity.getErrorMessage(),
                jpaEntity.getAttemptedAt(),
                jpaEntity.getResolvedAt()
        );
    }
}
