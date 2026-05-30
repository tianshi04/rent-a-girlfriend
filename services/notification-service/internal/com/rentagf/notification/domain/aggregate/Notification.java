package com.rentagf.notification.domain.aggregate;

import com.rentagf.notification.domain.errors.NotificationAlreadyCompletedException;
import com.rentagf.notification.domain.errors.NotificationNotFoundException;
import com.rentagf.notification.domain.errors.RetryLimitExceededException;
import com.rentagf.notification.domain.vo.DeliveryAttempt;
import com.rentagf.notification.domain.vo.enums.*;

import java.time.Instant;
import java.util.*;

/**
 * Aggregate Root cho Notification.
 * Hoàn toàn thuần Java, độc lập với JPA/Hibernate.
 * Tự bảo vệ các Invariants [INV-N01], [INV-N02].
 */
public class Notification {

    private final UUID id;
    private final UUID userId;
    private final String eventId;
    private final NotificationType type;
    private final NotificationPriority priority;
    private final Map<String, Object> payload;
    private final Map<String, Object> policyOverrides;
    private NotificationStatus status;
    private Instant readAt;
    private final Instant createdAt;
    private Instant updatedAt;
    private final List<DeliveryAttempt> attempts;

    // Full constructor
    public Notification(UUID id, UUID userId, String eventId, NotificationType type,
                        NotificationPriority priority, Map<String, Object> payload,
                        Map<String, Object> policyOverrides, NotificationStatus status,
                        Instant readAt, Instant createdAt, Instant updatedAt,
                        List<DeliveryAttempt> attempts) {
        this.id = id != null ? id : UUID.randomUUID();
        this.userId = Objects.requireNonNull(userId, "userId must not be null");
        this.eventId = Objects.requireNonNull(eventId, "eventId must not be null");
        this.type = Objects.requireNonNull(type, "type must not be null");
        this.priority = Objects.requireNonNull(priority, "priority must not be null");
        this.payload = payload != null ? new HashMap<>(payload) : new HashMap<>();
        this.policyOverrides = policyOverrides != null ? new HashMap<>(policyOverrides) : new HashMap<>();
        this.status = status != null ? status : NotificationStatus.PENDING;
        this.readAt = readAt;
        this.createdAt = createdAt != null ? createdAt : Instant.now();
        this.updatedAt = updatedAt != null ? updatedAt : Instant.now();
        this.attempts = attempts != null ? new ArrayList<>(attempts) : new ArrayList<>();
    }

    /**
     * Khởi tạo Notification mới từ UseCase.
     */
    public static Notification create(UUID userId, String eventId, NotificationType type,
                                      NotificationPriority priority, Map<String, Object> payload,
                                      Map<String, Object> policyOverrides) {
        Instant now = Instant.now();
        return new Notification(
                UUID.randomUUID(),
                userId,
                eventId,
                type,
                priority,
                payload,
                policyOverrides,
                NotificationStatus.PENDING,
                null,
                now,
                now,
                new ArrayList<>()
        );
    }

    /**
     * Tạo một nỗ lực gửi mới.
     * Bảo vệ [INV-N01] và [INV-N02].
     */
    public DeliveryAttempt createAttempt(DeliveryChannel channel) {
        // [INV-N02]: Không tạo attempt mới sau khi Notification đã COMPLETED
        if (this.status == NotificationStatus.COMPLETED) {
            throw new NotificationAlreadyCompletedException(this.id.toString());
        }

        // [INV-N01]: Giới hạn Retry tối đa 3 lần thất bại
        long failedCount = countFailedAttempts();
        if (failedCount >= 3) {
            throw new RetryLimitExceededException(this.id.toString());
        }

        DeliveryAttempt attempt = new DeliveryAttempt(
                UUID.randomUUID(),
                this.id,
                channel,
                AttemptStatus.PENDING,
                null,
                null,
                null,
                Instant.now(),
                null
        );

        this.status = NotificationStatus.PROCESSING;
        this.attempts.add(attempt);
        this.updatedAt = Instant.now();

        return attempt;
    }

    /**
     * Đánh dấu nỗ lực gửi thành công.
     */
    public void markAttemptSuccess(UUID attemptId, String messageId) {
        DeliveryAttempt attempt = findAttempt(attemptId);
        attempt.markSuccess(messageId);
        
        this.status = NotificationStatus.COMPLETED;
        this.updatedAt = Instant.now();
    }

    /**
     * Đánh dấu nỗ lực gửi thất bại.
     * Cập nhật trạng thái Notification dựa trên tính Recoverable của lỗi.
     */
    public void markAttemptFailed(UUID attemptId, String errorCode, String errorMessage, boolean isRecoverable) {
        DeliveryAttempt attempt = findAttempt(attemptId);
        
        if (isRecoverable) {
            attempt.markFailedRecoverable(errorCode, errorMessage);
            long failedCount = countFailedAttempts();
            if (failedCount >= 3) {
                // Đạt giới hạn 3 lần thử thất bại -> Notification FAILED [INV-N01]
                this.status = NotificationStatus.FAILED;
            } else {
                // Vẫn giữ PROCESSING để tiếp tục retry
                this.status = NotificationStatus.PROCESSING;
            }
        } else {
            // Lỗi Unrecoverable -> Đóng FAILED lập tức
            attempt.markFailedUnrecoverable(errorCode, errorMessage);
            this.status = NotificationStatus.FAILED;
        }

        this.updatedAt = Instant.now();
    }

    /**
     * Đánh dấu đã đọc.
     */
    public void markAsRead(Instant readAt) {
        this.readAt = readAt != null ? readAt : Instant.now();
        this.updatedAt = Instant.now();
    }

    private long countFailedAttempts() {
        return this.attempts.stream()
                .filter(a -> a.getStatus() == AttemptStatus.FAILED_RECOVERABLE || a.getStatus() == AttemptStatus.FAILED_UNRECOVERABLE)
                .count();
    }

    private DeliveryAttempt findAttempt(UUID attemptId) {
        return this.attempts.stream()
                .filter(a -> a.getId().equals(attemptId))
                .findFirst()
                .orElseThrow(() -> new IllegalArgumentException("Attempt with ID " + attemptId + " not found in Notification " + this.id));
    }

    // Getters
    public UUID getId() { return id; }
    public UUID getUserId() { return userId; }
    public String getEventId() { return eventId; }
    public NotificationType getType() { return type; }
    public NotificationPriority getPriority() { return priority; }
    public Map<String, Object> getPayload() { return Collections.unmodifiableMap(payload); }
    public Map<String, Object> getPolicyOverrides() { return Collections.unmodifiableMap(policyOverrides); }
    public NotificationStatus getStatus() { return status; }
    public Instant getReadAt() { return readAt; }
    public Instant getCreatedAt() { return createdAt; }
    public Instant getUpdatedAt() { return updatedAt; }
    public List<DeliveryAttempt> getAttempts() { return Collections.unmodifiableList(attempts); }
}
