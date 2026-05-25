package com.rentagf.notification.interfaces.event.resolver;

import java.util.List;
import java.util.Map;
import java.util.UUID;

/**
 * Interface chiến lược (Strategy Pattern) để phân giải người nhận từ Event Data.
 * Giúp hệ thống dễ dàng mở rộng thêm các logic phân giải người nhận mới
 * mà không cần chỉnh sửa mã nguồn hiện tại (tuân thủ nguyên lý Open-Closed).
 */
public interface RecipientResolver {

    /**
     * Phân giải danh sách người nhận (UUIDs) từ dữ liệu sự kiện.
     *
     * @param eventData      Dữ liệu sự kiện.
     * @param recipientField Trường người nhận tĩnh (từ template, có thể bị ignore bởi các strategy động).
     * @return Danh sách UUID người nhận.
     */
    List<UUID> resolve(Map<String, Object> eventData, String recipientField);

    /**
     * Xác định xem Resolver này có hỗ trợ xử lý loại sự kiện hiện tại hay không.
     *
     * @param eventType Loại sự kiện (Ví dụ: com.rentagf.booking.BookingCancelled.v1).
     * @return true nếu hỗ trợ.
     */
    boolean supports(String eventType);
}
