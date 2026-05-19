package com.rentagf.notification.application.port.outbound;

/**
 * Port cho Email delivery — driven adapter.
 * MVP: Mock implementation.
 */
public interface EmailPort {

    /**
     * Gửi email thông báo.
     *
     * @param toEmail  email người nhận
     * @param subject  tiêu đề
     * @param body     nội dung
     * @return true nếu gửi thành công
     */
    boolean send(String toEmail, String subject, String body);
}
