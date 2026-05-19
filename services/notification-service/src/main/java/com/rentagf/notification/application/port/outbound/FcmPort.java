package com.rentagf.notification.application.port.outbound;

/**
 * Port cho FCM Push delivery — driven adapter.
 * MVP: Mock implementation.
 */
public interface FcmPort {

    /**
     * Gửi push notification qua Firebase Cloud Messaging.
     *
     * @param userId  ID người nhận
     * @param title   tiêu đề
     * @param body    nội dung
     * @return true nếu gửi thành công
     */
    boolean send(String userId, String title, String body);
}
