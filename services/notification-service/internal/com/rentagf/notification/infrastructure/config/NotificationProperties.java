package com.rentagf.notification.infrastructure.config;

import lombok.Getter;
import lombok.Setter;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.context.annotation.Configuration;

/**
 * Custom config properties cho Notification Service. Đọc từ application.yml prefix "notification".
 */
@Getter
@Setter
@Configuration
@ConfigurationProperties(prefix = "notification")
public class NotificationProperties {

  private Templates templates = new Templates();
  private Sse sse = new Sse();
  private Retry retry = new Retry();
  private WorkerPool workerPool = new WorkerPool();
  private String defaultLocale = "vi";

  @Getter
  @Setter
  public static class Templates {
    private String path = "classpath:templates.yaml";
  }

  @Getter
  @Setter
  public static class Sse {
    private int heartbeatIntervalSeconds = 15;
  }

  @Getter
  @Setter
  public static class Retry {
    private int maxAttempts = 3;
    private int backoffMultiplier = 2;
    private int initialDelayMs = 2000;
  }

  @Getter
  @Setter
  public static class WorkerPool {
    private int size = 50;
  }
}
