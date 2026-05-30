package com.rentagf.notification.infrastructure.adapter;

import com.rentagf.notification.application.port.outbound.SendResult;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.vo.enums.DeliveryChannel;
import com.rentagf.notification.domain.vo.enums.NotificationPriority;
import com.rentagf.notification.domain.vo.enums.NotificationType;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;
import org.springframework.mail.MailSendException;
import org.springframework.mail.SimpleMailMessage;
import org.springframework.mail.javamail.JavaMailSender;

import java.util.Map;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

class EmailOutboundAdapterTest {

    private JavaMailSender mailSender;
    private EmailOutboundAdapter emailOutboundAdapter;

    @BeforeEach
    void setUp() {
        mailSender = mock(JavaMailSender.class);
        emailOutboundAdapter = new EmailOutboundAdapter(mailSender);
    }

    @Test
    void testSend_Success_DispatchesEmailCorrectly() {
        // Given
        UUID userId = UUID.randomUUID();
        Notification notification = Notification.create(
                userId, "evt-email-test", NotificationType.TRANSACTIONAL,
                NotificationPriority.HIGH, Map.of(
                        "title", "Hello Companion", 
                        "body", "Your booking is accepted!", 
                        "email", "companion@rentagf.com"
                ), Map.of()
        );

        doNothing().when(mailSender).send(any(SimpleMailMessage.class));

        // When
        SendResult result = emailOutboundAdapter.send(notification);

        // Then
        assertTrue(result.isSuccess());
        assertNotNull(result.getMessageId());
        assertTrue(result.getMessageId().startsWith("mail-"));

        ArgumentCaptor<SimpleMailMessage> messageCaptor = ArgumentCaptor.forClass(SimpleMailMessage.class);
        verify(mailSender, times(1)).send(messageCaptor.capture());

        SimpleMailMessage sentMessage = messageCaptor.getValue();
        assertEquals("no-reply@rentagf.com", sentMessage.getFrom());
        assertEquals("companion@rentagf.com", sentMessage.getTo()[0]);
        assertEquals("Hello Companion", sentMessage.getSubject());
        assertEquals("Your booking is accepted!", sentMessage.getText());
    }

    @Test
    void testSend_MailSenderException_ClassifiedAsRecoverable() {
        // Given
        UUID userId = UUID.randomUUID();
        Notification notification = Notification.create(
                userId, "evt-email-error", NotificationType.TRANSACTIONAL,
                NotificationPriority.HIGH, Map.of("title", "Hello"), Map.of()
        );

        // Giả lập SMTP Server bị offline / connection timeout
        doThrow(new MailSendException("SMTP connection timed out")).when(mailSender).send(any(SimpleMailMessage.class));

        // When
        SendResult result = emailOutboundAdapter.send(notification);

        // Then
        assertFalse(result.isSuccess());
        assertEquals("SMTP_SEND_FAILED", result.getErrorCode());
        assertTrue(result.isRecoverable(), "SMTP Connection loss must be recoverable");
    }

    @Test
    void testGetChannel_ReturnsEmail() {
        assertEquals(DeliveryChannel.EMAIL, emailOutboundAdapter.getChannel());
    }
}
