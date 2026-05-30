package com.rentagf.notification.interfaces.event;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.vo.enums.NotificationPriority;
import com.rentagf.notification.domain.vo.enums.NotificationType;
import com.rentagf.notification.interfaces.event.resolver.BookingCancelledResolver;
import com.rentagf.notification.interfaces.event.resolver.DisputeResolvedResolver;
import com.rentagf.notification.interfaces.event.resolver.SimpleRecipientResolver;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import java.io.File;
import java.time.Instant;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import static org.junit.jupiter.api.Assertions.*;

@Tag("unit")
public class EventTranslatorTest {

    private EventTranslator translator;

    @BeforeEach
    public void setUp() throws Exception {
        // Tải templates.yaml thật từ thư mục config của dự án để chạy test chính xác
        File templateFile = new File("config/templates.yaml");
        if (!templateFile.exists()) {
            // Fallback nếu chạy test từ thư mục con hoặc IDE cấu hình working dir khác
            templateFile = new File("services/notification-service/config/templates.yaml");
        }
        
        assertTrue(templateFile.exists(), "templates.yaml file must exist for testing");
        
        ObjectMapper objectMapper = new ObjectMapper();
        SimpleRecipientResolver simpleFallbackResolver = new SimpleRecipientResolver();
        
        List<com.rentagf.notification.interfaces.event.resolver.RecipientResolver> resolverList = List.of(
            new BookingCancelledResolver(objectMapper),
            new DisputeResolvedResolver(objectMapper)
        );
        
        RecipientResolverRegistry recipientResolverRegistry = new RecipientResolverRegistry(resolverList, simpleFallbackResolver);
        TemplateEngine templateEngine = new TemplateEngine(templateFile.getAbsolutePath());
        
        translator = new EventTranslator(templateEngine, recipientResolverRegistry);
    }

    @Test
    public void testTranslateKanoCoinDepositedSuccessfully() {
        UUID userId = UUID.randomUUID();
        String eventId = UUID.randomUUID().toString();
        
        CloudEvent event = new CloudEvent(
            "1.0",
            "com.rentagf.finance.KanoCoinDeposited.v1",
            "/services/finance",
            eventId,
            Instant.now(),
            "application/json",
            Map.of(
                "userId", userId.toString(),
                "amount", 500,
                "transactionId", "tx-999"
            )
        );

        List<Notification> notifications = translator.translate(event);

        assertEquals(1, notifications.size());
        Notification notification = notifications.getFirst();
        
        assertEquals(userId, notification.getUserId());
        assertEquals(eventId, notification.getEventId());
        assertEquals(NotificationType.TRANSACTIONAL, notification.getType());
        assertEquals(NotificationPriority.MEDIUM, notification.getPriority());
        
        // Kiểm tra placeholder interpolation tiếng Việt
        Map<String, Object> payload = notification.getPayload();
        assertEquals("Nạp tiền thành công 💰", payload.get("title"));
        assertEquals("Bạn vừa nạp thành công 500 Kano-Coin.", payload.get("body"));
    }

    @Test
    public void testTranslateBookingCancelledByClientResolvesDynamicRecipient() {
        UUID clientId = UUID.randomUUID();
        UUID companionId = UUID.randomUUID();
        String eventId = UUID.randomUUID().toString();
        
        CloudEvent event = new CloudEvent(
            "1.0",
            "com.rentagf.booking.BookingCancelled.v1",
            "/services/booking",
            eventId,
            Instant.now(),
            "application/json",
            Map.of(
                "bookingId", "b-888",
                "clientId", clientId.toString(),
                "companionId", companionId.toString(),
                "actorRole", "CLIENT"
            )
        );

        // Recipient của BookingCancelled phải là đối phương -> actorRole: CLIENT thì người nhận là companionId
        List<Notification> notifications = translator.translate(event);

        assertEquals(1, notifications.size());
        Notification notification = notifications.getFirst();
        
        assertEquals(companionId, notification.getUserId()); // Companion nhận tin
        assertEquals("Lịch hẹn bị hủy ⚠️", notification.getPayload().get("title"));
        assertEquals("Lịch hẹn #b-888 đã bị hủy bởi CLIENT.", notification.getPayload().get("body"));
    }

    @Test
    public void testTranslateDisputeResolvedSendsToBothParties() {
        UUID clientId = UUID.randomUUID();
        UUID companionId = UUID.randomUUID();
        String eventId = UUID.randomUUID().toString();
        
        CloudEvent event = new CloudEvent(
            "1.0",
            "com.rentagf.dispute.DisputeResolved.v1",
            "/services/dispute",
            eventId,
            Instant.now(),
            "application/json",
            Map.of(
                "disputeId", "d-777",
                "bookingId", "b-555",
                "clientId", clientId.toString(),
                "companionId", companionId.toString(),
                "resolution", "Hoàn trả 50% Kano-Coin"
            )
        );

        // DisputeResolved phải sinh ra 2 notifications riêng biệt cho cả hai bên
        List<Notification> notifications = translator.translate(event);

        assertEquals(2, notifications.size());
        
        boolean hasClient = false;
        boolean hasCompanion = false;
        
        for (Notification notification : notifications) {
            assertEquals(eventId, notification.getEventId());
            assertEquals("Kết quả khiếu nại 📋", notification.getPayload().get("title"));
            assertEquals("Khiếu nại cho booking đã được giải quyết: Hoàn trả 50% Kano-Coin.", notification.getPayload().get("body"));
            
            if (notification.getUserId().equals(clientId)) {
                hasClient = true;
            } else if (notification.getUserId().equals(companionId)) {
                hasCompanion = true;
            }
        }
        
        assertTrue(hasClient, "Should send notification to Client");
        assertTrue(hasCompanion, "Should send notification to Companion");
    }

    @Test
    public void testTranslateUnknownEventTypeThrowsIllegalArgumentException() {
        CloudEvent unknownEvent = new CloudEvent(
            "1.0",
            "com.rentagf.unknown.Event.v1",
            "/services/unknown",
            UUID.randomUUID().toString(),
            Instant.now(),
            "application/json",
            Map.of("userId", UUID.randomUUID().toString())
        );

        assertThrows(IllegalArgumentException.class, () -> translator.translate(unknownEvent));
    }
}
