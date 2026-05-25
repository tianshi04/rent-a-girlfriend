package com.rentagf.notification.infrastructure.adapter;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.rentagf.notification.application.port.outbound.PubSubPort;
import com.rentagf.notification.application.port.outbound.SendResult;
import com.rentagf.notification.application.port.outbound.SsePort;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.vo.enums.DeliveryChannel;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import java.util.HashMap;
import java.util.Map;

/**
 * Adapter hạ tầng chịu trách nhiệm gửi thông báo thời gian thực qua SSE.
 * Thực thi hợp đồng SsePort bằng cách đóng gói tin nhắn và publish lên Redis Pub/Sub.
 */
@Component
public class SseOutboundAdapter implements SsePort {

    private static final Logger log = LoggerFactory.getLogger(SseOutboundAdapter.class);
    private static final String CHANNEL_PREFIX = "user:%s:sse";

    private final PubSubPort pubSubPort;
    private final ObjectMapper objectMapper;

    public SseOutboundAdapter(PubSubPort pubSubPort, ObjectMapper objectMapper) {
        this.pubSubPort = pubSubPort;
        this.objectMapper = objectMapper;
    }

    @Override
    public SendResult send(Notification notification) {
        if (notification == null) {
            return SendResult.fail("NULL_NOTIFICATION", "Notification aggregate must not be null", false);
        }

        try {
            String channel = String.format(CHANNEL_PREFIX, notification.getUserId().toString());
            log.info("Preparing to send SSE notification {} to channel {}", notification.getId(), channel);

            // Xây dựng SSE Payload để gửi xuống client
            Map<String, Object> ssePayload = new HashMap<>(notification.getPayload());
            ssePayload.put("id", notification.getId().toString());
            ssePayload.put("type", notification.getType().name());
            ssePayload.put("priority", notification.getPriority().name());

            // Serialize sang JSON
            String jsonPayload = objectMapper.writeValueAsString(ssePayload);

            // Publish thông qua PubSubPort (Redis)
            pubSubPort.publish(channel, jsonPayload);

            log.info("SSE notification {} published successfully to channel {}", notification.getId(), channel);
            return SendResult.success(notification.getId().toString());

        } catch (JsonProcessingException e) {
            log.error("Failed to serialize SSE payload for notification {}", notification.getId(), e);
            return SendResult.fail("SERIALIZATION_ERROR", e.getMessage(), false);
        } catch (Exception e) {
            log.error("Unexpected error occurred while publishing SSE notification {}", notification.getId(), e);
            // Lỗi kỹ thuật khác được coi là recoverable = true để cho phép retry
            return SendResult.fail("SSE_PUBLISH_FAILED", e.getMessage(), true);
        }
    }

    @Override
    public DeliveryChannel getChannel() {
        return DeliveryChannel.SSE;
    }
}
