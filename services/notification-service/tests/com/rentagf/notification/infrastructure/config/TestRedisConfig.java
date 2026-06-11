package com.rentagf.notification.infrastructure.config;

import org.mockito.Mockito;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Primary;
import org.springframework.data.redis.connection.RedisConnectionFactory;
import org.springframework.data.redis.core.StringRedisTemplate;

/**
 * Cấu hình giả lập Redis cho môi trường Integration Test của toàn hệ thống. Cung cấp các Mock Beans
 * của Redis để Spring Boot Application Context có thể khởi tạo mượt mà mọi Adapter mà không bị ném
 * lỗi thiếu Bean.
 */
@Configuration
public class TestRedisConfig {

  @Bean
  @Primary
  public RedisConnectionFactory redisConnectionFactory() {
    return Mockito.mock(RedisConnectionFactory.class);
  }

  @Bean
  @Primary
  public StringRedisTemplate stringRedisTemplate() {
    return Mockito.mock(StringRedisTemplate.class);
  }
}
