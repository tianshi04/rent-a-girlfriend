package com.rentagf.notification.infrastructure.adapter;

import com.rentagf.notification.application.port.outbound.PubSubPort;
import com.rentagf.notification.infrastructure.sse.SseConnectionRegistry;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.context.annotation.Lazy;
import org.springframework.data.redis.connection.Message;
import org.springframework.data.redis.connection.MessageListener;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.listener.ChannelTopic;
import org.springframework.data.redis.listener.RedisMessageListenerContainer;
import org.springframework.stereotype.Component;
import org.springframework.web.servlet.mvc.method.annotation.SseEmitter;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.util.List;
import java.util.UUID;

/**
 * Adapter hạ tầng sử dụng Redis Pub/Sub để truyền thông điệp liên-máy (Inter-Pod).
 * Triển khai PubSubPort (Outbound) và MessageListener (Inbound/Distributed).
 */
@Component
public class RedisPubSubAdapter implements PubSubPort, MessageListener {

    private static final Logger log = LoggerFactory.getLogger(RedisPubSubAdapter.class);

    private final StringRedisTemplate redisTemplate;
    private final RedisMessageListenerContainer listenerContainer;
    private final SseConnectionRegistry sseConnectionRegistry;

    /**
     * Constructor Injection.
     * Sử dụng @Lazy trên sseConnectionRegistry để tháo gỡ hoàn toàn lỗi
     * Circular Dependency (SseConnectionRegistry <-> RedisPubSubAdapter) lúc startup.
     */
    public RedisPubSubAdapter(
            StringRedisTemplate redisTemplate,
            RedisMessageListenerContainer listenerContainer,
            @Lazy SseConnectionRegistry sseConnectionRegistry) {
        this.redisTemplate = redisTemplate;
        this.listenerContainer = listenerContainer;
        this.sseConnectionRegistry = sseConnectionRegistry;
    }

    // ==========================================
    // TÁC VỤ GỬI TIN (OUTBOUND - PubSubPort)
    // ==========================================

    @Override
    public void publish(String channel, String message) {
        if (channel == null || message == null) {
            return;
        }
        log.debug("Publishing message to Redis channel: {}", channel);
        redisTemplate.convertAndSend(channel, message);
    }

    @Override
    public void subscribe(String channel) {
        if (channel == null) {
            return;
        }
        log.info("Subscribing dynamically to Redis channel: {}", channel);
        listenerContainer.addMessageListener(this, new ChannelTopic(channel));
    }

    @Override
    public void unsubscribe(String channel) {
        if (channel == null) {
            return;
        }
        log.info("Unsubscribing from Redis channel: {}", channel);
        listenerContainer.removeMessageListener(this, new ChannelTopic(channel));
    }

    // ==========================================
    // TÁC VỤ NHẬN TIN (INBOUND - MessageListener)
    // ==========================================

    @Override
    public void onMessage(Message message, byte[] pattern) {
        String channel = new String(message.getChannel(), StandardCharsets.UTF_8);
        String body = new String(message.getBody(), StandardCharsets.UTF_8);

        log.debug("Received message from Redis channel [{}]: {}", channel, body);

        // Bóc tách userId từ tên kênh "user:{userId}:sse"
        UUID userId = extractUserIdFromChannel(channel);
        if (userId == null) {
            log.warn("Failed to extract userId from Redis channel name: {}", channel);
            return;
        }

        // Lấy danh sách SseEmitter đang active của User trên Pod này từ Registry
        List<SseEmitter> emitters = sseConnectionRegistry.getEmitters(userId);
        if (emitters.isEmpty()) {
            log.debug("No active SSE connections found on this pod for user {}", userId);
            return;
        }

        log.debug("Found {} active SSE connections on this pod for user {}. Pushing data...", emitters.size(), userId);

        // Phát tín hiệu thông báo thời gian thực tới tất cả các kết nối active của User
        for (SseEmitter emitter : emitters) {
            try {
                // Spring Data Redis tự động bọc chuỗi trong dấu nháy kép khi convert,
                // chúng ta gửi payload JSON sạch đi
                String cleanBody = sanitizePayload(body);
                emitter.send(SseEmitter.event()
                        .name("notification")
                        .data(cleanBody));
            } catch (IOException e) {
                log.warn("Broken pipe detected when sending SSE to user {}. Triggering error handler...", userId);
                // Spring Boot sẽ tự kích hoạt callback onError/onCompletion đã đăng ký để unregister
                emitter.completeWithError(e);
            }
        }
    }

    /**
     * Bóc tách UUID userId từ chuỗi channel "user:<userId>:sse"
     */
    private UUID extractUserIdFromChannel(String channel) {
        try {
            String[] parts = channel.split(":");
            if (parts.length >= 2) {
                return UUID.fromString(parts[1]);
            }
        } catch (IllegalArgumentException e) {
            log.error("Channel name contains invalid UUID format: {}", channel, e);
        }
        return null;
    }

    /**
     * Dọn dẹp dấu nháy kép thừa nếu Redis serializer sinh ra chuỗi JSON thô dạng String.
     */
    private String sanitizePayload(String body) {
        if (body.startsWith("\"") && body.endsWith("\"") && body.length() > 1) {
            return body.substring(1, body.length() - 1).replace("\\\"", "\"");
        }
        return body;
    }
}
