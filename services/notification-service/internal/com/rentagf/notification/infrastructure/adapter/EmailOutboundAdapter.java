package com.rentagf.notification.infrastructure.adapter;

import com.rentagf.notification.application.port.outbound.EmailPort;
import com.rentagf.notification.application.port.outbound.SendResult;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.vo.enums.DeliveryChannel;
import lombok.extern.slf4j.Slf4j;
import org.springframework.mail.SimpleMailMessage;
import org.springframework.mail.javamail.JavaMailSender;
import org.springframework.stereotype.Component;

import java.util.UUID;

/**
 * Adapter hạ tầng gửi Email sử dụng Spring Boot Starter Mail (JavaMailSender).
 * Kết nối SMTP Server thực tế để thực hiện gửi Email.
 */
@Slf4j
@Component
public class EmailOutboundAdapter implements EmailPort {

    private final JavaMailSender mailSender;

    public EmailOutboundAdapter(JavaMailSender mailSender) {
        this.mailSender = mailSender;
    }

    @Override
    public SendResult send(Notification notification) {
        log.info("Preparing to send email for notification: {}", notification.getId());

        String title = (String) notification.getPayload().get("title");
        String body = (String) notification.getPayload().get("body");
        UUID userId = notification.getUserId();

        // Email người nhận: Thử bốc từ payload nếu có, nếu không thì fallback sang định dạng mặc định userId@rentagf.com
        String toEmail = (String) notification.getPayload().getOrDefault("email", userId.toString() + "@rentagf.com");

        try {
            SimpleMailMessage message = new SimpleMailMessage();
            message.setFrom("no-reply@rentagf.com");
            message.setTo(toEmail);
            message.setSubject(title != null ? title : "Rent-A-Girlfriend Notification");
            message.setText(body != null ? body : "");

            log.info("Sending SMTP email to: {}", toEmail);
            mailSender.send(message);

            String messageId = "mail-" + UUID.randomUUID();
            return SendResult.success(messageId);

        } catch (Exception e) {
            log.error("Failed to send SMTP email to {}", toEmail, e);
            // Lỗi SMTP thường là recoverable (vd: nghẽn mạng SMTP, timeout), đánh dấu true để cho phép retry ở phase sau
            return SendResult.fail("SMTP_SEND_FAILED", e.getMessage(), true);
        }
    }

    @Override
    public DeliveryChannel getChannel() {
        return DeliveryChannel.EMAIL;
    }
}
