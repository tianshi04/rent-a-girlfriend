package com.rentagf.notification.infrastructure.sse;

import com.rentagf.notification.application.port.outbound.ConnectionStatePort;
import com.rentagf.notification.application.port.outbound.PubSubPort;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;
import org.springframework.web.servlet.mvc.method.annotation.SseEmitter;

import java.util.*;

/**
 * Orchestrator điều phối vòng đời kết nối SSE và đồng bộ trạng thái phân tán.
 * Đảm bảo Single Responsibility Principle (SRP) bằng cách chỉ lo việc phối hợp các ports hạ tầng,
 * uỷ thác hoàn toàn việc lưu trữ in-memory cục bộ cho SseLocalRegistry.
 */
@Component
public class SseConnectionRegistry {

    private static final Logger log = LoggerFactory.getLogger(SseConnectionRegistry.class);
    private static final String CHANNEL_PREFIX = "user:%s:sse";
    private static final long STATE_TTL_SECONDS = 45L; // TTL 45s (Heartbeat 15s + buffer)

    private final SseLocalRegistry localRegistry;
    private final PubSubPort pubSubPort;
    private final ConnectionStatePort connectionStatePort;

    public SseConnectionRegistry(
            SseLocalRegistry localRegistry,
            PubSubPort pubSubPort,
            ConnectionStatePort connectionStatePort) {
        this.localRegistry = localRegistry;
        this.pubSubPort = pubSubPort;
        this.connectionStatePort = connectionStatePort;
    }

    /**
     * Đăng ký một kết nối SSE mới của User.
     * Nếu đây là kết nối đầu tiên của User trên Pod này:
     * - Kích hoạt subscribe kênh Pub/Sub.
     * - Ghi nhận trạng thái Online lên Redis Connection State Store.
     *
     * @param userId  ID của người nhận (UUID)
     * @param emitter Đối tượng SseEmitter được tạo lập
     */
    public void register(UUID userId, SseEmitter emitter) {
        if (userId == null || emitter == null) {
            log.warn("Cannot register SSE connection with null userId or null emitter");
            return;
        }

        boolean isFirst = localRegistry.add(userId, emitter);
        if (isFirst) {
            String channel = getChannelName(userId);
            log.info("First SSE connection for user {}. Activating subscription on channel: {} and setting Redis Online status", userId, channel);
            pubSubPort.subscribe(channel);
            connectionStatePort.setOnline(userId, STATE_TTL_SECONDS);
        }
    }

    /**
     * Hủy đăng ký kết nối SSE của User.
     * Nếu đây là kết nối active cuối cùng của User trên Pod này:
     * - Kích hoạt unsubscribe kênh Pub/Sub.
     * - Ghi nhận trạng thái Offline trên Redis.
     *
     * @param userId  ID của người nhận (UUID)
     * @param emitter Đối tượng SseEmitter cần hủy
     */
    public void unregister(UUID userId, SseEmitter emitter) {
        if (userId == null || emitter == null) {
            return;
        }

        boolean isLast = localRegistry.remove(userId, emitter);
        if (isLast) {
            String channel = getChannelName(userId);
            log.info("Last SSE connection closed for user {}. Deactivating subscription on channel: {} and removing Redis Online status", userId, channel);
            pubSubPort.unsubscribe(channel);
            connectionStatePort.setOffline(userId);
        }
    }

    /**
     * Nhận sự kiện Ping Heartbeat thành công từ Controller.
     * Refresh lại TTL của trạng thái Online trên Redis để tránh hết hạn.
     */
    public void heartbeat(UUID userId) {
        if (userId == null) return;
        if (localRegistry.exists(userId)) {
            connectionStatePort.refreshHeartbeat(userId, STATE_TTL_SECONDS);
        }
    }

    /**
     * Lấy danh sách toàn bộ kết nối SseEmitter đang active của User trên Pod này.
     *
     * @param userId ID của User
     * @return Danh sách SseEmitter
     */
    public List<SseEmitter> getEmitters(UUID userId) {
        return localRegistry.getEmitters(userId);
    }

    /**
     * Sinh tên kênh Pub/Sub theo định dạng chuẩn nghiệp vụ.
     */
    private String getChannelName(UUID userId) {
        return String.format(CHANNEL_PREFIX, userId.toString());
    }
}
