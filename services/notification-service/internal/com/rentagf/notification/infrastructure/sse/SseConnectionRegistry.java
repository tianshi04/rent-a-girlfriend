package com.rentagf.notification.infrastructure.sse;

import com.rentagf.notification.application.port.outbound.PubSubPort;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;
import org.springframework.web.servlet.mvc.method.annotation.SseEmitter;

import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.CopyOnWriteArrayList;

/**
 * Thread-safe Registry quản lý danh sách kết nối SSE cục bộ của người dùng trên Pod này.
 * Áp dụng cơ chế Reference Counting để đăng ký/hủy đăng ký subscribe động trên hạ tầng Pub/Sub.
 */
@Component
public class SseConnectionRegistry {

    private static final Logger log = LoggerFactory.getLogger(SseConnectionRegistry.class);
    private static final String CHANNEL_PREFIX = "user:%s:sse";

    private final Map<UUID, List<SseEmitter>> registry = new ConcurrentHashMap<>();
    private final PubSubPort pubSubPort;

    public SseConnectionRegistry(PubSubPort pubSubPort) {
        this.pubSubPort = pubSubPort;
    }

    /**
     * Đăng ký một kết nối SSE mới của User vào Registry cục bộ.
     * Nếu đây là kết nối đầu tiên của User trên Pod này, kích hoạt subscribe kênh Pub/Sub.
     *
     * @param userId  ID của người nhận (UUID)
     * @param emitter Đối tượng SseEmitter được tạo lập
     */
    public void register(UUID userId, SseEmitter emitter) {
        if (userId == null || emitter == null) {
            log.warn("Cannot register SSE connection with null userId or null emitter");
            return;
        }

        registry.compute(userId, (key, emitters) -> {
            if (emitters == null) {
                emitters = new CopyOnWriteArrayList<>();
                String channel = getChannelName(userId);
                log.info("First SSE connection for user {}. Activating subscription on channel: {}", userId, channel);
                pubSubPort.subscribe(channel);
            }
            emitters.add(emitter);
            log.debug("Registered SSE connection for user {}. Active connections on this pod: {}", userId, emitters.size());
            return emitters;
        });
    }

    /**
     * Hủy đăng ký kết nối SSE của User khỏi Registry.
     * Nếu đây là kết nối active cuối cùng của User trên Pod này, kích hoạt unsubscribe kênh Pub/Sub.
     *
     * @param userId  ID của người nhận (UUID)
     * @param emitter Đối tượng SseEmitter cần hủy
     */
    public void unregister(UUID userId, SseEmitter emitter) {
        if (userId == null || emitter == null) {
            return;
        }

        registry.computeIfPresent(userId, (key, emitters) -> {
            boolean removed = emitters.remove(emitter);
            if (removed) {
                log.debug("Unregistered active SSE connection for user {}. Remaining connections: {}", userId, emitters.size());
            }

            if (emitters.isEmpty()) {
                String channel = getChannelName(userId);
                log.info("Last SSE connection closed for user {}. Deactivating subscription on channel: {}", userId, channel);
                pubSubPort.unsubscribe(channel);
                return null; // Xóa hoàn toàn key khỏi ConcurrentHashMap
            }
            return emitters;
        });
    }

    /**
     * Lấy danh sách toàn bộ kết nối SseEmitter đang active của User trên Pod này từ Registry.
     *
     * @param userId ID của User
     * @return Danh sách SseEmitter (trả về Empty List nếu User không có kết nối nào)
     */
    public List<SseEmitter> getEmitters(UUID userId) {
        if (userId == null) {
            return Collections.emptyList();
        }
        List<SseEmitter> emitters = registry.get(userId);
        return emitters != null ? Collections.unmodifiableList(emitters) : Collections.emptyList();
    }

    /**
     * Sinh tên kênh Pub/Sub theo định dạng chuẩn nghiệp vụ.
     */
    private String getChannelName(UUID userId) {
        return String.format(CHANNEL_PREFIX, userId.toString());
    }
}
