package com.rentagf.notification;

import com.rentagf.notification.application.port.outbound.ConnectionStatePort;
import com.rentagf.notification.application.port.outbound.PubSubPort;
import org.junit.jupiter.api.Test;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.mail.javamail.JavaMailSender;
import org.springframework.test.context.bean.override.mockito.MockitoBean;

/**
 * Smoke test: đảm bảo toàn bộ Spring ApplicationContext khởi động thành công.
 *
 * <p>Mock các bean hạ tầng ngoài (Redis, Mail) để test có thể chạy mà không cần
 * kết nối thực tế.
 */
@SpringBootTest
class NotificationServiceApplicationTests {

    /** Thay thế RedisConnectionStateAdapter – StringRedisTemplate bị exclude trong test. */
    @MockitoBean
    private ConnectionStatePort connectionStatePort;

    /** Thay thế RedisPubSubAdapter – StringRedisTemplate + RedisMessageListenerContainer bị exclude trong test. */
    @MockitoBean
    private PubSubPort pubSubPort;

    /** Thay thế JavaMailSender để tránh kết nối SMTP thực khi test. */
    @MockitoBean
    private JavaMailSender javaMailSender;

    @Test
    void contextLoads() {
        // Spring context load thành công là đủ
    }
}
