package com.rentagf.notification.interfaces.event;

import com.rentagf.notification.interfaces.event.resolver.RecipientResolver;
import com.rentagf.notification.interfaces.event.resolver.SimpleRecipientResolver;
import org.springframework.stereotype.Component;

import java.util.List;
import java.util.Map;
import java.util.UUID;

/**
 * Registry đóng vai trò điều phối (Orchestrator) việc phân giải người nhận.
 * Áp dụng Design Pattern: Registry + Strategy + Ultimate Fallback.
 */
@Component
public class RecipientResolverRegistry {

    private final List<RecipientResolver> strategyResolvers;
    private final SimpleRecipientResolver simpleFallbackResolver;

    /**
     * @param strategyResolvers      Tự động inject tất cả các class implements RecipientResolver.
     *                               Spring sẽ bỏ qua SimpleRecipientResolver vì nó không implement interface này.
     * @param simpleFallbackResolver Kẻ hứng đáy (Fallback) cho các event thông thường.
     */
    public RecipientResolverRegistry(
            List<RecipientResolver> strategyResolvers,
            SimpleRecipientResolver simpleFallbackResolver) {
        this.strategyResolvers = strategyResolvers;
        this.simpleFallbackResolver = simpleFallbackResolver;
    }

    /**
     * Phân giải danh sách người nhận từ dữ liệu sự kiện.
     *
     * @param eventType     Loại sự kiện (vd: com.rentagf.booking.BookingCancelled.v1).
     * @param data          Payload của sự kiện.
     * @param recipientField Trường dữ liệu dự phòng từ template (nếu có).
     * @return Danh sách UUID của những người cần nhận thông báo.
     */
    public List<UUID> resolve(String eventType, Map<String, Object> data, String recipientField) {
        if (data == null) {
            throw new IllegalArgumentException("Event data must not be null");
        }

        // Ưu tiên duyệt qua các Strategy chuyên biệt
        return strategyResolvers.stream()
                .filter(r -> r.supports(eventType))
                .findFirst()
                .map(r -> r.resolve(data, recipientField))
                // Nếu không có Strategy nào khớp, rớt xuống Fallback
                .orElseGet(() -> simpleFallbackResolver.resolve(data, recipientField));
    }
}
