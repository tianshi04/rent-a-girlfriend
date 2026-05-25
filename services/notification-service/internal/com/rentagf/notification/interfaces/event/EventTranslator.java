package com.rentagf.notification.interfaces.event;

import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.vo.enums.NotificationPriority;
import com.rentagf.notification.domain.vo.enums.NotificationType;
import org.springframework.stereotype.Component;

import java.util.*;

/**
 * Trình biên dịch sự kiện (Translator).
 * Dịch CloudEvent thô thành một danh sách các Domain Notification.
 * Tích hợp TemplateEngine để ráp ngôn ngữ và RecipientResolverRegistry để xác định người nhận.
 */
@Component
public class EventTranslator {

    private final TemplateEngine templateEngine;
    private final RecipientResolverRegistry recipientResolverRegistry;

    public EventTranslator(TemplateEngine templateEngine, RecipientResolverRegistry recipientResolverRegistry) {
        this.templateEngine = templateEngine;
        this.recipientResolverRegistry = recipientResolverRegistry;
    }

    /**
     * Dịch một CloudEvent thành danh sách các Notification.
     * Áp dụng FAIL-FAST: Nếu event không hợp lệ, ném Exception ngay lập tức.
     *
     * @param event Đối tượng CloudEvent thô từ Kafka.
     * @return Danh sách Notification (có thể là 1 hoặc nhiều người nhận).
     */
    public List<Notification> translate(CloudEvent event) {
        if (event == null) {
            throw new IllegalArgumentException("CloudEvent must not be null");
        }

        String eventType = event.getType();
        String eventId = event.getId();
        Map<String, Object> data = event.getData();

        if (eventType == null || eventId == null || data == null) {
            throw new IllegalArgumentException("CloudEvent missing required metadata (type, id, or data)");
        }

        // 1. Kiểm tra cấu hình event trong templates.yaml
        // Phương thức này sẽ tự động ném IllegalArgumentException nếu eventType không tồn tại
        Map<String, Object> config = templateEngine.getEventConfig(eventType);

        // 2. Xác định danh sách người nhận (UUIDs) bằng Registry
        String recipientField = templateEngine.getRecipientField(eventType);
        List<UUID> recipientIds = recipientResolverRegistry.resolve(eventType, data, recipientField);

        if (recipientIds == null || recipientIds.isEmpty()) {
            throw new IllegalArgumentException("No recipients resolved for event type: " + eventType);
        }

        // 3. Phân tích các thông số loại và độ ưu tiên
        NotificationType type = parseNotificationType((String) config.get("type"));
        NotificationPriority priority = parseNotificationPriority((String) config.get("priority"));

        // 3.5. Trích xuất danh sách kênh cấu hình
        List<String> channels = templateEngine.getChannels(eventType);

        // 4. Ráp dữ liệu động vào tiêu đề và body tiếng Việt (FAIL-FAST nếu thiếu biến)
        String title = templateEngine.render(eventType, data, true);
        String body = templateEngine.render(eventType, data, false);

        Map<String, Object> payload = Map.of(
                "title", title,
                "body", body
        );

        // 5. Sinh ra danh sách Notification
        List<Notification> notifications = new ArrayList<>();
        for (UUID userId : recipientIds) {
            Notification notification = Notification.create(
                    userId,
                    eventId,
                    type,
                    priority,
                    payload,
                    Map.of("channels", channels != null ? channels : List.of()) // Lưu kênh cấu hình vào policyOverrides
            );
            notifications.add(notification);
        }

        return notifications;
    }

    private NotificationType parseNotificationType(String typeStr) {
        if (typeStr == null) {
            return NotificationType.TRANSACTIONAL;
        }
        try {
            return NotificationType.valueOf(typeStr.toUpperCase());
        } catch (IllegalArgumentException e) {
            return NotificationType.TRANSACTIONAL; // Fallback
        }
    }

    private NotificationPriority parseNotificationPriority(String priorityStr) {
        if (priorityStr == null) {
            return NotificationPriority.MEDIUM;
        }
        try {
            return NotificationPriority.valueOf(priorityStr.toUpperCase());
        } catch (IllegalArgumentException e) {
            return NotificationPriority.MEDIUM; // Fallback
        }
    }
}
