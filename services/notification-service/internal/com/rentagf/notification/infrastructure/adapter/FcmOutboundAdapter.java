package com.rentagf.notification.infrastructure.adapter;

import com.google.firebase.FirebaseApp;
import com.google.firebase.messaging.FirebaseMessaging;
import com.google.firebase.messaging.Message;
import com.rentagf.notification.application.port.outbound.FcmPort;
import com.rentagf.notification.application.port.outbound.SendResult;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.vo.enums.DeliveryChannel;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;

import java.util.UUID;

/**
 * Adapter hạ tầng gửi Push Notification qua Firebase Cloud Messaging (FCM).
 * Tích hợp Firebase Admin SDK chính thức.
 * Hỗ trợ chế độ mô phỏng chi tiết (Simulation Mode) nếu ứng dụng khởi chạy không có file credential Service Account.
 */
@Slf4j
@Component
public class FcmOutboundAdapter implements FcmPort {

    @Override
    public SendResult send(Notification notification) {
        log.info("Preparing to send FCM push for notification: {}", notification.getId());

        String title = (String) notification.getPayload().get("title");
        String body = (String) notification.getPayload().get("body");
        UUID userId = notification.getUserId();

        // 1. Kiểm tra xem Firebase SDK đã được khởi tạo chưa (bằng Service Account credentials)
        if (FirebaseApp.getApps().isEmpty()) {
            // Chế độ mô phỏng (Simulation Mode) phục vụ Local Dev & Unit/Integration Testing
            log.warn("=== [FCM SIMULATION MODE] ===");
            log.warn("Firebase Admin SDK is not initialized (missing Service Account credentials).");
            log.warn("Simulating FCM Push Delivery to User: {}", userId);
            log.warn("FCM Title: {}", title);
            log.warn("FCM Body: {}", body);
            log.warn("=== [SIMULATION END SUCCESS] ===");

            String simulatedMessageId = "fcm-sim-" + UUID.randomUUID();
            return SendResult.success(simulatedMessageId);
        }

        // 2. Chế độ gửi thật sử dụng Firebase Messaging API
        try {
            // Sử dụng fully qualified name để tránh conflict với Domain Notification
            com.google.firebase.messaging.Notification fcmNotification = 
                com.google.firebase.messaging.Notification.builder()
                    .setTitle(title)
                    .setBody(body)
                    .build();

            // Gửi qua Topic cụ thể của User (ví dụ: user-123) để push tới toàn bộ thiết bị của user đó
            String userTopic = "user-" + userId.toString();

            Message message = Message.builder()
                    .setNotification(fcmNotification)
                    .setTopic(userTopic)
                    .putData("notificationId", notification.getId().toString())
                    .build();

            log.info("Sending real FCM Push to user topic: {}", userTopic);
            String messageId = FirebaseMessaging.getInstance().send(message);
            
            return SendResult.success(messageId);

        } catch (Exception e) {
            log.error("Failed to send FCM push to user: {}", userId, e);
            // Lỗi FCM API có thể là recoverable (timeout, rate limit) hoặc unrecoverable (sai token, credential hết hạn)
            // Phân tích nhanh để gán tính recoverable
            boolean isRecoverable = e.getMessage() != null && 
                (e.getMessage().contains("timeout") || e.getMessage().contains("unavailable") || e.getMessage().contains("Rate limit"));
            
            return SendResult.fail("FCM_SEND_FAILED", e.getMessage(), isRecoverable);
        }
    }

    @Override
    public DeliveryChannel getChannel() {
        return DeliveryChannel.FCM;
    }
}
