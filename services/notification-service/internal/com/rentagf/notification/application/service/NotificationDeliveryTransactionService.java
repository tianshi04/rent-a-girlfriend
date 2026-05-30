package com.rentagf.notification.application.service;

import com.rentagf.notification.application.port.outbound.SendResult;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.repository.NotificationRepository;
import com.rentagf.notification.domain.vo.DeliveryAttempt;
import com.rentagf.notification.domain.vo.enums.DeliveryChannel;
import lombok.Getter;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Propagation;
import org.springframework.transaction.annotation.Transactional;

import java.util.ArrayList;
import java.util.Collection;
import java.util.List;
import java.util.UUID;

/**
 * Helper Service chứa các phương thức thực thi cơ sở dữ liệu ngắn chạy trong Transaction mới.
 * Thiết kế này là cốt lõi của "Hikari Connection Exhaustion Guard", đảm bảo giải phóng connection DB
 * ngay lập tức trước khi luồng thực hiện các cuộc gọi mạng I/O dài (FCM, Email).
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class NotificationDeliveryTransactionService {

    private final NotificationRepository notificationRepository;

    /**
     * DTO lưu trữ ngữ cảnh chuẩn bị gửi.
     */
    @Getter
    @RequiredArgsConstructor
    public static class DeliveryContext {
        private final Notification notification;
        private final UUID attemptId;
    }

    /**
     * DTO lưu trữ kết quả hành động sau khi xử lý kết quả gửi.
     */
    @Getter
    @RequiredArgsConstructor
    public static class DeliveryActionResult {
        private final boolean shouldRetry;
        private final boolean shouldFallbackToFcm;
        private final int failedCount;
    }

    /**
     * Bước 1: Transaction ngắn - Load Notification, tạo Attempt và lưu để chuyển trạng thái sang PROCESSING.
     */
    @Transactional(propagation = Propagation.REQUIRES_NEW, rollbackFor = Exception.class)
    public DeliveryContext prepareAttempt(UUID notificationId, DeliveryChannel channel) {
        log.debug("[TX-PREPARE] Creating attempt for notification {} via channel {}", notificationId, channel);

        Notification notification = notificationRepository.findById(notificationId)
                .orElseThrow(() -> new IllegalArgumentException("Notification not found: " + notificationId));

        // Tạo attempt mới (Invariants [INV-N01], [INV-N02] tự động được bảo vệ ở mức Domain)
        DeliveryAttempt attempt = notification.createAttempt(channel);
        
        // Lưu trạng thái PENDING của attempt và status PROCESSING của Notification vào DB
        Notification saved = notificationRepository.save(notification);

        return new DeliveryContext(saved, attempt.getId());
    }

    /**
     * Bước 3: Transaction ngắn - Ghi nhận kết quả gửi tin và đưa ra quyết định hành động tiếp theo.
     */
    @Transactional(propagation = Propagation.REQUIRES_NEW, rollbackFor = Exception.class)
    public DeliveryActionResult handleResult(UUID notificationId, UUID attemptId, SendResult result, DeliveryChannel channel) {
        log.debug("[TX-RESULT] Processing delivery result for notification {}, attempt {}", notificationId, attemptId);

        Notification notification = notificationRepository.findById(notificationId)
                .orElseThrow(() -> new IllegalArgumentException("Notification not found: " + notificationId));

        boolean shouldRetry = false;
        boolean shouldFallbackToFcm = false;
        int failedCount = 0;

        if (result.isSuccess()) {
            notification.markAttemptSuccess(attemptId, result.getMessageId());
            log.info("[TX-RESULT] Notification {} marked as COMPLETED via channel {}", notificationId, channel);
        } else {
            // Lỗi gửi tin ngoại vi
            notification.markAttemptFailed(attemptId, result.getErrorCode(), result.getErrorMessage(), result.isRecoverable());
            log.warn("[TX-RESULT] Attempt {} failed. Code: {}, Error: {}", attemptId, result.getErrorCode(), result.getErrorMessage());

            // Đếm số lần thất bại (phục vụ tính toán delay retry)
            failedCount = (int) notification.getAttempts().stream()
                    .filter(a -> a.getStatus().name().startsWith("FAILED"))
                    .count();

            // 1. Kiểm tra SSE Failover [INV-N09]
            if (channel == DeliveryChannel.SSE) {
                // SSE hỏng -> Xem cấu hình có kênh FCM không để kích hoạt fallback
                List<String> configuredChannels = extractChannels(notification);
                if (configuredChannels.contains("FCM")) {
                    shouldFallbackToFcm = true;
                }
            } else {
                // 2. Kiểm tra chính sách Retry [INV-N10]
                // Chỉ retry nếu lỗi là recoverable và chưa quá giới hạn 3 lần
                if (result.isRecoverable() && failedCount < 3) {
                    shouldRetry = true;
                }
            }
        }

        // Lưu kết quả cuối cùng vào DB
        notificationRepository.save(notification);

        return new DeliveryActionResult(shouldRetry, shouldFallbackToFcm, failedCount);
    }

    /**
     * Trích xuất các kênh cấu hình an toàn từ policyOverrides.
     */
    @SuppressWarnings("unchecked")
    private List<String> extractChannels(Notification notification) {
        Object channelsObj = notification.getPolicyOverrides().get("channels");
        if (channelsObj == null) {
            return List.of();
        }
        if (channelsObj instanceof List) {
            List<?> rawList = (List<?>) channelsObj;
            List<String> list = new ArrayList<>();
            for (Object o : rawList) {
                if (o != null) {
                    list.add(o.toString());
                }
            }
            return list;
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
            return List.of();
        }
    }
}
