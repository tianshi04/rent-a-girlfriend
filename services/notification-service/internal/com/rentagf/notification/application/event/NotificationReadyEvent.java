package com.rentagf.notification.application.event;

import com.rentagf.notification.domain.vo.enums.DeliveryChannel;
import lombok.Getter;
import java.util.UUID;

/**
 * Sự kiện thông báo đã sẵn sàng gửi đi.
 * Được bắn ra từ Application Service sau khi lưu DB thành công trạng thái thông báo.
 */
@Getter
public class NotificationReadyEvent {
    private final UUID notificationId;
    private final DeliveryChannel channel;

    public NotificationReadyEvent(UUID notificationId, DeliveryChannel channel) {
        this.notificationId = notificationId;
        this.channel = channel;
    }
}
