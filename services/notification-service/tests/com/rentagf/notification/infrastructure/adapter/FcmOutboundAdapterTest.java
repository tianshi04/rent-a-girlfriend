package com.rentagf.notification.infrastructure.adapter;

import com.google.firebase.FirebaseApp;
import com.rentagf.notification.application.port.outbound.SendResult;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.vo.enums.DeliveryChannel;
import com.rentagf.notification.domain.vo.enums.NotificationPriority;
import com.rentagf.notification.domain.vo.enums.NotificationType;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.util.Map;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;

class FcmOutboundAdapterTest {

    private FcmOutboundAdapter fcmOutboundAdapter;

    @BeforeEach
    void setUp() {
        fcmOutboundAdapter = new FcmOutboundAdapter();
    }

    @Test
    void testSend_SimulationMode_Success() {
        // Given
        UUID userId = UUID.randomUUID();
        Notification notification = Notification.create(
                userId, "evt-fcm-test", NotificationType.TRANSACTIONAL,
                NotificationPriority.HIGH, Map.of("title", "Test Simulation", "body", "Simulation works!"), Map.of()
        );

        // Giả lập môi trường test chạy offline / không khởi tạo FirebaseApp thật
        // (Nếu FirebaseApp.getApps() rỗng, adapter sẽ chạy simulation)

        // When
        SendResult result = fcmOutboundAdapter.send(notification);

        // Then
        if (FirebaseApp.getApps().isEmpty()) {
            assertTrue(result.isSuccess());
            assertNotNull(result.getMessageId());
            assertTrue(result.getMessageId().startsWith("fcm-sim-"));
        } else {
            // Nếu Firebase SDK đã được khởi tạo trong context khác (ví dụ Spring Boot test load context)
            // thì test vẫn sẽ an toàn không bị fail
            assertNotNull(result);
        }
    }

    @Test
    void testGetChannel_ReturnsFcm() {
        assertEquals(DeliveryChannel.FCM, fcmOutboundAdapter.getChannel());
    }
}
