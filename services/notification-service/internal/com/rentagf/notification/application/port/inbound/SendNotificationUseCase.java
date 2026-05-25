package com.rentagf.notification.application.port.inbound;

import com.rentagf.notification.domain.aggregate.Notification;

/**
 * Inbound Port (UseCase) định nghĩa nghiệp vụ định tuyến và gửi thông báo tự động.
 * Đảm bảo Single Responsibility Principle (SRP).
 */
public interface SendNotificationUseCase {

    /**
     * Định tuyến thông minh và phân phối thông báo đến người nhận tương ứng.
     *
     * @param notification Notification aggregate đã được chuẩn bị và dịch nghĩa.
     * @return Notification aggregate sau khi đã xử lý định tuyến và lưu DB.
     */
    Notification routeAndSend(Notification notification);
}
