package com.rentagf.notification.infrastructure.config;

import com.rentagf.notification.interfaces.event.TemplateEngine;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.io.Resource;
import org.springframework.core.io.ResourceLoader;

import java.io.IOException;
import java.io.InputStream;

/**
 * Cấu hình khởi tạo TemplateEngine tải file templates.yaml.
 * Hỗ trợ tải động từ classpath trong cả môi trường local và đóng gói jar chạy thật.
 */
@Configuration
public class TemplateConfig {

    private static final Logger log = LoggerFactory.getLogger(TemplateConfig.class);

    @Bean
    public TemplateEngine templateEngine(
            NotificationProperties notificationProperties,
            ResourceLoader resourceLoader) {
        
        String templatesPath = notificationProperties.getTemplates().getPath();
        log.info("Initializing TemplateEngine with templates path: {}", templatesPath);
        Resource resource = resourceLoader.getResource(templatesPath);
        
        if (!resource.exists()) {
            throw new IllegalArgumentException("Templates configuration file not found at: " + templatesPath);
        }

        try (InputStream inputStream = resource.getInputStream()) {
            TemplateEngine engine = new TemplateEngine(inputStream);
            log.info("TemplateEngine initialized successfully from resource.");
            return engine;
        } catch (IOException e) {
            throw new IllegalStateException("Failed to read templates configuration from " + templatesPath, e);
        }
    }
}
