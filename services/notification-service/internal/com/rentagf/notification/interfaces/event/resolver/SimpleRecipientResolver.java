package com.rentagf.notification.interfaces.event.resolver;

import org.springframework.stereotype.Component;
import java.util.List;
import java.util.Map;
import java.util.UUID;

/**
 * Fallback resolver cho các sự kiện thông thường.
 * Trích xuất UUID trực tiếp từ trường dữ liệu được chỉ định (recipientField).
 * Không implement RecipientResolver để đảm bảo nó luôn đứng ngoài danh sách Strategy tự động nạp.
 */
@Component
public class SimpleRecipientResolver {

    public List<UUID> resolve(Map<String, Object> eventData, String recipientField) {
        if (recipientField == null || recipientField.trim().isEmpty()) {
            throw new IllegalArgumentException("recipientField must not be null or empty for simple resolution");
        }

        Object recipientIdObj = eventData.get(recipientField);
        if (recipientIdObj == null) {
            throw new IllegalArgumentException("Missing recipient field in event data: " + recipientField);
        }

        try {
            UUID recipientId = UUID.fromString(recipientIdObj.toString());
            return List.of(recipientId);
        } catch (IllegalArgumentException e) {
            throw new IllegalArgumentException("Field " + recipientField + " is not a valid UUID: " + recipientIdObj);
        }
    }
}
