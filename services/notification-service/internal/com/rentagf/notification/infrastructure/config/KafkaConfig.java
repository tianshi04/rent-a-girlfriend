package com.rentagf.notification.infrastructure.config;

import lombok.extern.slf4j.Slf4j;
import org.springframework.boot.autoconfigure.condition.ConditionalOnBean;
import org.springframework.boot.autoconfigure.kafka.KafkaProperties;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.task.SimpleAsyncTaskExecutor;
import org.springframework.kafka.annotation.EnableKafka;
import org.springframework.kafka.config.ConcurrentKafkaListenerContainerFactory;
import org.springframework.kafka.core.ConsumerFactory;
import org.springframework.kafka.core.DefaultKafkaConsumerFactory;

/**
 * Cấu hình Spring Kafka Consumer theo chuẩn Spring Boot.
 *
 * <p>Chiến lược: reuse toàn bộ cấu hình từ application.yml (bao gồm ack-mode, error handler...)
 * thông qua {@link KafkaProperties}, sau đó override duy nhất Executor để tích hợp
 * Java 21 Virtual Threads.
 *
 * <p>{@code @ConditionalOnBean(ConsumerFactory.class)} đảm bảo bean này chỉ được khởi tạo
 * khi KafkaAutoConfiguration đang hoạt động (production / integration test với EmbeddedKafka).
 * Các unit test exclude Kafka sẽ không bị ảnh hưởng.
 */
@Slf4j
@EnableKafka
@Configuration
@ConditionalOnBean(ConsumerFactory.class)
public class KafkaConfig {

    /**
     * Định nghĩa {@link ConcurrentKafkaListenerContainerFactory} theo pattern chuẩn Spring Boot.
     *
     * <p>Sử dụng {@link KafkaProperties} để tái sử dụng toàn bộ cấu hình đã khai báo trong
     * {@code application.yml} (ack-mode, group-id, deserializer, error-handler, v.v.).
     * Sau đó override Executor để chạy trên Java 21 Virtual Threads thay vì Platform Threads.
     *
     * @param kafkaProperties Cấu hình Kafka từ application.yml (auto-injected bởi Spring Boot).
     * @return Factory được cấu hình đầy đủ cho Virtual Threads.
     */
    @Bean
    public ConcurrentKafkaListenerContainerFactory<String, String> kafkaListenerContainerFactory(
            KafkaProperties kafkaProperties) {

        // Tạo ConsumerFactory dựa trên toàn bộ config từ application.yml
        ConsumerFactory<String, String> consumerFactory =
                new DefaultKafkaConsumerFactory<>(kafkaProperties.buildConsumerProperties(null));

        ConcurrentKafkaListenerContainerFactory<String, String> factory =
                new ConcurrentKafkaListenerContainerFactory<>();

        factory.setConsumerFactory(consumerFactory);

        // Áp dụng AckMode từ application.yml (spring.kafka.listener.ack-mode: manual)
        factory.getContainerProperties().setAckMode(
                kafkaProperties.getListener().getAckMode()
        );

        // Override Executor: Java 21 Virtual Threads để không block Platform Thread
        SimpleAsyncTaskExecutor executor = new SimpleAsyncTaskExecutor("kafka-vt-");
        executor.setVirtualThreads(true);
        factory.getContainerProperties().setListenerTaskExecutor(executor);

        log.info("Kafka Consumer Container Factory initialized successfully with Java 21 Virtual Threads. " +
                "AckMode: {}", kafkaProperties.getListener().getAckMode());
        return factory;
    }
}
