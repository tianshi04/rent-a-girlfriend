package com.rentagf.notification.infrastructure.config;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.task.SimpleAsyncTaskExecutor;
import org.springframework.data.redis.connection.RedisConnectionFactory;
import org.springframework.data.redis.listener.RedisMessageListenerContainer;

/**
 * Cấu hình hạ tầng Redis Pub/Sub thời gian thực.
 * Tối ưu hóa Threading bằng cách cho phép Redis Message Listener hoạt động trên Virtual Threads.
 */
@Configuration
public class RedisPubSubConfig {

    private static final Logger log = LoggerFactory.getLogger(RedisPubSubConfig.class);

    /**
     * Khởi tạo container quản lý các kết nối lắng nghe Redis Pub/Sub.
     * Cấu hình sử dụng SimpleAsyncTaskExecutor chạy trên Java 21 Virtual Threads
     * giúp giải phóng tối đa Platform Threads khi nhận tin nhắn phân phối.
     */
    @Bean
    public RedisMessageListenerContainer redisMessageListenerContainer(RedisConnectionFactory connectionFactory) {
        log.info("Configuring RedisMessageListenerContainer with Java 21 Virtual Threads...");
        RedisMessageListenerContainer container = new RedisMessageListenerContainer();
        container.setConnectionFactory(connectionFactory);

        // Sử dụng SimpleAsyncTaskExecutor chạy trên Virtual Threads
        SimpleAsyncTaskExecutor taskExecutor = new SimpleAsyncTaskExecutor("redis-pubsub-vt-");
        container.setTaskExecutor(taskExecutor);

        log.info("RedisMessageListenerContainer configured successfully.");
        return container;
    }
}
