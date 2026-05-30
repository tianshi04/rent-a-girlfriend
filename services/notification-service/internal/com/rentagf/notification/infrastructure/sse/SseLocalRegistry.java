package com.rentagf.notification.infrastructure.sse;

import org.springframework.stereotype.Component;
import org.springframework.web.servlet.mvc.method.annotation.SseEmitter;

import java.util.*;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.CopyOnWriteArrayList;

/**
 * Registry cục bộ quản lý Map in-memory chứa danh sách kết nối active SseEmitter.
 * Đảm bảo Single Responsibility Principle (SRP) bằng cách chỉ lo việc quản lý bộ nhớ cục bộ,
 * hoàn toàn không chứa dependency hạ tầng hay nghiệp vụ đồng bộ nào.
 */
@Component
public class SseLocalRegistry {

    private final Map<UUID, List<SseEmitter>> registry = new ConcurrentHashMap<>();

    /**
     * Thêm kết nối SseEmitter mới cho User.
     *
     * @param userId  ID của User.
     * @param emitter Đối tượng SseEmitter mới.
     * @return true nếu đây là kết nối đầu tiên của User trên Pod này.
     */
    public boolean add(UUID userId, SseEmitter emitter) {
        if (userId == null || emitter == null) {
            return false;
        }

        final boolean[] isFirst = {false};
        registry.compute(userId, (key, emitters) -> {
            List<SseEmitter> list = emitters;
            if (list == null) {
                list = new CopyOnWriteArrayList<>();
                isFirst[0] = true;
            }
            list.add(emitter);
            return list;
        });
        return isFirst[0];
    }

    /**
     * Huỷ đăng ký kết nối SseEmitter cho User.
     *
     * @param userId  ID của User.
     * @param emitter Đối tượng SseEmitter cần huỷ.
     * @return true nếu registry của User này trống rỗng hoàn toàn sau khi xoá (kết nối cuối cùng).
     */
    public boolean remove(UUID userId, SseEmitter emitter) {
        if (userId == null || emitter == null) {
            return false;
        }

        final boolean[] isLast = {false};
        registry.computeIfPresent(userId, (key, emitters) -> {
            emitters.remove(emitter);
            if (emitters.isEmpty()) {
                isLast[0] = true;
                return null; // Xoá key khỏi map
            }
            return emitters;
        });
        return isLast[0];
    }

    /**
     * Kiểm tra xem User có đang giữ bất kỳ kết nối active nào trên Pod này không.
     */
    public boolean exists(UUID userId) {
        return userId != null && registry.containsKey(userId);
    }

    /**
     * Lấy danh sách toàn bộ active SseEmitters của User trên Pod này.
     */
    public List<SseEmitter> getEmitters(UUID userId) {
        if (userId == null) {
            return Collections.emptyList();
        }
        List<SseEmitter> emitters = registry.get(userId);
        return emitters != null ? Collections.unmodifiableList(emitters) : Collections.emptyList();
    }
}
