package com.rentagf.notification.application.port.inbound;

import com.rentagf.notification.domain.aggregate.Notification;

import java.util.List;

/**
 * Inbound Port (UseCase) định nghĩa nghiệp vụ routing và phân phối thông báo đã được dịch.
 *
 * <p>Ranh giới trách nhiệm rõ ràng:
 * <ul>
 *   <li>KHÔNG làm: parse JSON, translate CloudEvent → đó là việc của Inbound Adapter (KafkaEventConsumer).</li>
 *   <li>CHỈ LÀM: kiểm tra Idempotency + định tuyến thông minh từng Notification qua SendNotificationUseCase.</li>
 * </ul>
 * Thiết kế này đảm bảo Application Layer hoàn toàn độc lập với Interfaces Layer (Hexagonal Architecture RULE 4).
 */
public interface ProcessInboundEventUseCase {

    /**
     * Định tuyến và phân phối danh sách Notification đã được translate từ CloudEvent.
     *
     * @param notifications Danh sách Notification đã được dịch nghĩa sẵn từ CloudEvent.
     * @param eventId       ID của event gốc – dùng cho Idempotency logging.
     */
    void process(List<Notification> notifications, String eventId);
}
