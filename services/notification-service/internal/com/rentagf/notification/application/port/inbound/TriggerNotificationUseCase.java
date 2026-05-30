package com.rentagf.notification.application.port.inbound;

import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.vo.enums.DeliveryChannel;
import com.rentagf.notification.domain.vo.enums.NotificationPriority;
import com.rentagf.notification.domain.vo.enums.NotificationType;

import java.util.Map;
import java.util.UUID;

/**
 * Inbound Port (UseCase) định nghĩa nghiệp vụ gửi thông báo thủ công qua kênh chỉ định sẵn.
 * Đảm bảo tính tách biệt trách nhiệm (Single Responsibility Principle) hoàn hảo.
 */
public interface TriggerNotificationUseCase {

    /**
     * Gửi thông báo thủ công chỉ định sẵn kênh phân phối.
     *
     * @param userId    UUID người nhận.
     * @param eventId   Idempotency key của sự kiện.
     * @param type      Loại thông báo (TRANSACTIONAL, MARKETING...).
     * @param priority  Mức độ ưu tiên.
     * @param payload   Dữ liệu thông báo (title, body...).
     * @param channel   Kênh phân phối vật lý chỉ định sẵn.
     * @return Notification aggregate sau khi xử lý và kích hoạt gửi bất đồng bộ.
     */
    Notification triggerNotification(UUID userId, String eventId, NotificationType type,
                                    NotificationPriority priority, Map<String, Object> payload,
                                    DeliveryChannel channel);
}
