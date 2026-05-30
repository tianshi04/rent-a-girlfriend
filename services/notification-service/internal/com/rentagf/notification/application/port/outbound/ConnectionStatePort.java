package com.rentagf.notification.application.port.outbound;

import java.util.UUID;

/**
 * Outbound Port để quản lý và truy vấn trạng thái Online/Offline của Client trên toàn Cluster.
 */
public interface ConnectionStatePort {

    /**
     * Ghi nhận trạng thái Online của User lên Redis với TTL chỉ định.
     */
    void setOnline(UUID userId, long ttlSeconds);

    /**
     * Ghi nhận trạng thái Offline (xóa trạng thái Online) của User khỏi Redis.
     */
    void setOffline(UUID userId);

    /**
     * Kiểm tra xem User có đang Online trong Cluster hay không.
     */
    boolean isOnline(UUID userId);

    /**
     * Gia hạn TTL của trạng thái Online (khi nhận được Ping Heartbeat).
     */
    void refreshHeartbeat(UUID userId, long ttlSeconds);
}
