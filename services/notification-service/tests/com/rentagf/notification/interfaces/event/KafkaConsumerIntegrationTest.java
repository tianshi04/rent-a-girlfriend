package com.rentagf.notification.interfaces.event;

import com.rentagf.notification.application.port.outbound.ConnectionStatePort;
import com.rentagf.notification.application.port.outbound.PubSubPort;
import com.rentagf.notification.infrastructure.persistence.jpa.entity.NotificationJpaEntity;
import com.rentagf.notification.infrastructure.persistence.jpa.repository.NotificationJpaRepository;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.bean.override.mockito.MockitoBean;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.kafka.test.context.EmbeddedKafka;
import org.springframework.mail.javamail.JavaMailSender;
import org.springframework.test.context.ActiveProfiles;

import java.util.List;
import java.util.UUID;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.when;

@SpringBootTest(properties = {
    // Chỉ exclude những gì không cần trong E2E test này:
    // - Redis: không có Redis trong test
    // - MailHealthContributor: Spring Actuator cố tạo mail health check nhưng không có real JavaMailSender
    "spring.autoconfigure.exclude=" +
        "org.springframework.boot.autoconfigure.data.redis.RedisAutoConfiguration," +
        "org.springframework.boot.autoconfigure.data.redis.RedisRepositoriesAutoConfiguration," +
        "org.springframework.boot.actuate.autoconfigure.mail.MailHealthContributorAutoConfiguration",
    // KafkaAutoConfiguration KHÔNG exclude → được bật lại để test E2E hoạt động
    "spring.kafka.consumer.auto-offset-reset=earliest",
    "spring.kafka.listener.ack-mode=manual"
})
@EmbeddedKafka(
    partitions = 1,
    bootstrapServersProperty = "spring.kafka.bootstrap-servers",
    topics = {
        "booking-events",
        "finance-events",
        "interaction-events",
        "profile-events",
        "identity-events",
        "dispute-events"
    }
)
@ActiveProfiles("test")
@Tag("integration")
public class KafkaConsumerIntegrationTest {

    @Autowired
    private KafkaTemplate<String, String> kafkaTemplate;

    @Autowired
    private NotificationJpaRepository jpaRepository;

    @MockitoBean
    private ConnectionStatePort connectionStatePort;

    @MockitoBean
    private PubSubPort pubSubPort;

    @MockitoBean
    private JavaMailSender javaMailSender;

    @BeforeEach
    public void setUp() {
        jpaRepository.deleteAll();
        // Mặc định cho rằng user offline để test luồng fallback Push/Email
        when(connectionStatePort.isOnline(any(UUID.class))).thenReturn(false);
    }

    @Test
    public void testConsumeKanoCoinDepositedSuccessfully() throws Exception {
        UUID userId = UUID.randomUUID();
        String eventId = UUID.randomUUID().toString();

        // 1. Tạo JSON CloudEvent thô giả lập từ Finance Service gửi lên Kafka
        String rawEventJson = "{"
                + "\"specversion\":\"1.0\","
                + "\"type\":\"com.rentagf.finance.KanoCoinDeposited.v1\","
                + "\"source\":\"/services/finance\","
                + "\"id\":\"" + eventId + "\","
                + "\"time\":\"2026-05-23T12:00:00Z\","
                + "\"datacontenttype\":\"application/json\","
                + "\"data\":{"
                + "  \"userId\":\"" + userId + "\","
                + "  \"amount\":1500,"
                + "  \"transactionId\":\"tx-888\""
                + "}"
                + "}";

        // 2. Publish lên Embedded Kafka Topic
        kafkaTemplate.send("finance-events", rawEventJson).get();

        // 3. Đợi luồng tiêu thụ bất đồng bộ chạy trên Virtual Threads hoàn thành việc xử lý và lưu DB
        CountDownLatch latch = new CountDownLatch(1);
        for (int i = 0; i < 50; i++) {
            List<NotificationJpaEntity> saved = jpaRepository.findAll();
            if (!saved.isEmpty()) {
                latch.countDown();
                break;
            }
            TimeUnit.MILLISECONDS.sleep(100);
        }

        assertTrue(latch.await(5, TimeUnit.SECONDS), "Notification should be consumed and saved to DB");

        // 4. Kiểm chứng dữ liệu trong Database để chứng minh toàn bộ E2E Flow chạy thành công!
        List<NotificationJpaEntity> savedNotifications = jpaRepository.findAll();
        assertEquals(1, savedNotifications.size());

        NotificationJpaEntity entity = savedNotifications.getFirst();
        assertEquals(userId, entity.getUserId());
        assertEquals(eventId, entity.getEventId());
        
        // Kiểm chứng nội dung tiếng Việt đã dịch
        assertEquals("Nạp tiền thành công 💰", entity.getPayload().get("title"));
        assertEquals("Bạn vừa nạp thành công 1500 Kano-Coin.", entity.getPayload().get("body"));
    }
}
