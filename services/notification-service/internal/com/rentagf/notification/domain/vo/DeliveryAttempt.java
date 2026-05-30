package com.rentagf.notification.domain.vo;

import com.rentagf.notification.domain.vo.enums.AttemptStatus;
import com.rentagf.notification.domain.vo.enums.DeliveryChannel;

import java.time.Instant;
import java.util.Objects;
import java.util.UUID;

/**
 * Entity DeliveryAttempt biểu diễn một nỗ lực gửi thông báo cụ thể qua một kênh.
 * Hoàn toàn thuần Java, độc lập với JPA/Hibernate.
 */
public class DeliveryAttempt {

    private final UUID id;
    private final UUID notificationId;
    private final DeliveryChannel channel;
    private AttemptStatus status;
    private String messageId;
    private String errorCode;
    private String errorMessage;
    private final Instant attemptedAt;
    private Instant resolvedAt;

    public DeliveryAttempt(UUID id, UUID notificationId, DeliveryChannel channel,
                           AttemptStatus status, String messageId, String errorCode,
                           String errorMessage, Instant attemptedAt, Instant resolvedAt) {
        this.id = id != null ? id : UUID.randomUUID();
        this.notificationId = Objects.requireNonNull(notificationId, "notificationId must not be null");
        this.channel = Objects.requireNonNull(channel, "channel must not be null");
        this.status = status != null ? status : AttemptStatus.PENDING;
        this.messageId = messageId;
        this.errorCode = errorCode;
        this.errorMessage = errorMessage;
        this.attemptedAt = attemptedAt != null ? attemptedAt : Instant.now();
        this.resolvedAt = resolvedAt;
    }

    public void markSuccess(String messageId) {
        this.status = AttemptStatus.SUCCESS;
        this.messageId = messageId;
        this.resolvedAt = Instant.now();
    }

    public void markFailedRecoverable(String errorCode, String errorMessage) {
        this.status = AttemptStatus.FAILED_RECOVERABLE;
        this.errorCode = errorCode;
        this.errorMessage = errorMessage;
        this.resolvedAt = Instant.now();
    }

    public void markFailedUnrecoverable(String errorCode, String errorMessage) {
        this.status = AttemptStatus.FAILED_UNRECOVERABLE;
        this.errorCode = errorCode;
        this.errorMessage = errorMessage;
        this.resolvedAt = Instant.now();
    }

    // Getters
    public UUID getId() {
        return id;
    }

    public UUID getNotificationId() {
        return notificationId;
    }

    public DeliveryChannel getChannel() {
        return channel;
    }

    public AttemptStatus getStatus() {
        return status;
    }

    public String getMessageId() {
        return messageId;
    }

    public String getErrorCode() {
        return errorCode;
    }

    public String getErrorMessage() {
        return errorMessage;
    }

    public Instant getAttemptedAt() {
        return attemptedAt;
    }

    public Instant getResolvedAt() {
        return resolvedAt;
    }
}
