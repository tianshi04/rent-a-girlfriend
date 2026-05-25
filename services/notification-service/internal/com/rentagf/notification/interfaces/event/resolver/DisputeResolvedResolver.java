package com.rentagf.notification.interfaces.event.resolver;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.springframework.stereotype.Component;

import java.util.List;
import java.util.Map;
import java.util.UUID;

/**
 * Resolver đặc thù cho sự kiện giải quyết khiếu nại (DisputeResolved).
 * Cần gửi thông báo cho cả 2 bên (Client và Companion).
 */
@Component
public class DisputeResolvedResolver implements RecipientResolver {

    private final ObjectMapper objectMapper;

    // Payload cục bộ chỉ dùng để parse an toàn trong class này, bỏ qua các trường không cần thiết
    @JsonIgnoreProperties(ignoreUnknown = true)
    private record DisputeResolvedPayload(String clientId, String companionId) {}

    public DisputeResolvedResolver(ObjectMapper objectMapper) {
        this.objectMapper = objectMapper;
    }

    @Override
    public List<UUID> resolve(Map<String, Object> eventData, String recipientField) {
        try {
            DisputeResolvedPayload payload = objectMapper.convertValue(eventData, DisputeResolvedPayload.class);

            if (payload.clientId() == null || payload.companionId() == null) {
                throw new IllegalArgumentException("Missing required fields (clientId, companionId) in DisputeResolved payload");
            }

            // Trả về danh sách chứa cả 2 ID để Translator sinh ra 2 bản Notification độc lập
            return List.of(
                    UUID.fromString(payload.clientId()),
                    UUID.fromString(payload.companionId())
            );
        } catch (IllegalArgumentException e) {
            throw e;
        } catch (Exception e) {
            throw new IllegalArgumentException("Invalid payload schema for DisputeResolved", e);
        }
    }

    @Override
    public boolean supports(String eventType) {
        return "com.rentagf.dispute.DisputeResolved.v1".equals(eventType);
    }
}
