package com.rentagf.notification.application.port.outbound;

import com.rentagf.notification.domain.vo.enums.DeliveryChannel;

import java.time.Duration;
import java.util.UUID;

/**
 * Port Outbound định nghĩa giao diện lên lịch gửi lại thông báo trì hoãn (Retry).
 * Đây là abstraction cốt lõi để cô lập hạ tầng Retry (In-Memory, Kafka, RabbitMQ) khỏi Domain Logic.
 */
public interface RetrySchedulerPort {

    /**
     * Lên lịch gửi lại một thông báo sau khoảng thời gian delay.
     *
     * @param notificationId ID của Notification
     * @param channel        Kênh gửi tin tương ứng
     * @param delay          Khoảng thời gian trì hoãn (Backoff)
     */
    void scheduleRetry(UUID notificationId, DeliveryChannel channel, Duration delay);
}
