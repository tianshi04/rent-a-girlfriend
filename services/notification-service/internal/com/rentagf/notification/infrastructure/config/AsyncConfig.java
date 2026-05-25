package com.rentagf.notification.infrastructure.config;

import org.springframework.context.annotation.Configuration;
import org.springframework.scheduling.annotation.EnableAsync;

/**
 * Cấu hình xử lý tác vụ bất đồng bộ.
 * Khi spring.threads.virtual.enabled: true được bật,
 * Spring sẽ tự động dùng Virtual Threads để thực thi các tác vụ @Async.
 */
@Configuration
@EnableAsync
public class AsyncConfig {
}
