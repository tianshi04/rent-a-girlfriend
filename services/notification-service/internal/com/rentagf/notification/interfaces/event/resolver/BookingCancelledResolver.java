package com.rentagf.notification.interfaces.event.resolver;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.springframework.stereotype.Component;

import java.util.List;
import java.util.Map;
import java.util.UUID;

/**
 * Resolver đặc thù cho sự kiện hủy lịch (BookingCancelled).
 * Áp dụng Local Typed Record để lấy Compile-time safety cho payload.
 */
@Component
public class BookingCancelledResolver implements RecipientResolver {

    private final ObjectMapper objectMapper;

    // Payload cục bộ chỉ dùng để parse an toàn trong class này, bỏ qua các trường không cần thiết
    @JsonIgnoreProperties(ignoreUnknown = true)
    private record BookingCancelledPayload(String actorRole, String clientId, String companionId) {}

    public BookingCancelledResolver(ObjectMapper objectMapper) {
        this.objectMapper = objectMapper;
    }

    @Override
    public List<UUID> resolve(Map<String, Object> eventData, String recipientField) {
        try {
            // Ép kiểu Map thành Typed Record. Nếu schema thiếu, sẽ văng lỗi Deserialize
            BookingCancelledPayload payload = objectMapper.convertValue(eventData, BookingCancelledPayload.class);
            
            if (payload.actorRole() == null || payload.clientId() == null || payload.companionId() == null) {
                throw new IllegalArgumentException("Missing required fields (actorRole, clientId, companionId) in BookingCancelled payload");
            }

            // Logic định tuyến chéo: Ai hủy thì báo cho người kia
            if ("CLIENT".equalsIgnoreCase(payload.actorRole())) {
                return List.of(UUID.fromString(payload.companionId()));
            } else if ("COMPANION".equalsIgnoreCase(payload.actorRole())) {
                return List.of(UUID.fromString(payload.clientId()));
            } else {
                throw new IllegalArgumentException("Unknown actorRole: " + payload.actorRole());
            }
        } catch (IllegalArgumentException e) {
            throw e;
        } catch (Exception e) {
            throw new IllegalArgumentException("Invalid payload schema for BookingCancelled", e);
        }
    }

    @Override
    public boolean supports(String eventType) {
        return "com.rentagf.booking.BookingCancelled.v1".equals(eventType);
    }
}
