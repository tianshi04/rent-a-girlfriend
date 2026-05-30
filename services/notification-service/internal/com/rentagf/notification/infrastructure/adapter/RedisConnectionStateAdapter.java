package com.rentagf.notification.infrastructure.adapter;

import com.rentagf.notification.application.port.outbound.ConnectionStatePort;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.stereotype.Component;

import java.util.UUID;
import java.util.concurrent.TimeUnit;

/**
 * Adapter triển khai ConnectionStatePort sử dụng Redis làm Connection State Store.
 * Quản lý trạng thái Online/Offline phân tán bằng cách lưu trữ String key trên Redis.
 */
@Component
public class RedisConnectionStateAdapter implements ConnectionStatePort {

    private static final Logger log = LoggerFactory.getLogger(RedisConnectionStateAdapter.class);
    private static final String ONLINE_KEY_PREFIX = "user:online:%s";
    private static final String ONLINE_STATUS_VALUE = "online";

    private final StringRedisTemplate redisTemplate;

    public RedisConnectionStateAdapter(StringRedisTemplate redisTemplate) {
        this.redisTemplate = redisTemplate;
    }

    @Override
    public void setOnline(UUID userId, long ttlSeconds) {
        if (userId == null) return;
        String key = getRedisKey(userId);
        try {
            log.debug("Setting online state in Redis for user: {} with TTL {}s", userId, ttlSeconds);
            redisTemplate.opsForValue().set(key, ONLINE_STATUS_VALUE, ttlSeconds, TimeUnit.SECONDS);
        } catch (Exception e) {
            log.error("Failed to set online state in Redis for user: {}", userId, e);
        }
    }

    @Override
    public void setOffline(UUID userId) {
        if (userId == null) return;
        String key = getRedisKey(userId);
        try {
            log.debug("Deleting online state in Redis for user: {}", userId);
            redisTemplate.delete(key);
        } catch (Exception e) {
            log.error("Failed to delete online state in Redis for user: {}", userId, e);
        }
    }

    @Override
    public boolean isOnline(UUID userId) {
        if (userId == null) return false;
        String key = getRedisKey(userId);
        try {
            Boolean hasKey = redisTemplate.hasKey(key);
            boolean online = hasKey != null && hasKey;
            log.debug("Checking online status in Redis for user {}: {}", userId, online);
            return online;
        } catch (Exception e) {
            log.error("Failed to check online status in Redis for user: {}", userId, e);
            return false; // Fallback an toàn coi như offline
        }
    }

    @Override
    public void refreshHeartbeat(UUID userId, long ttlSeconds) {
        if (userId == null) return;
        String key = getRedisKey(userId);
        try {
            log.debug("Refreshing online state TTL in Redis for user: {} to {}s", userId, ttlSeconds);
            redisTemplate.expire(key, ttlSeconds, TimeUnit.SECONDS);
        } catch (Exception e) {
            log.error("Failed to refresh online state TTL in Redis for user: {}", userId, e);
        }
    }

    private String getRedisKey(UUID userId) {
        return String.format(ONLINE_KEY_PREFIX, userId.toString());
    }
}
