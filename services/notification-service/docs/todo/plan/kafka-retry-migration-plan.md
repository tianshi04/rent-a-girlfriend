# 🚀 KẾ HOẠCH DỊCH CHUYỂN SANG KAFKA-BASED RETRY (KAFKA RETRY MIGRATION PLAN)

## 1. Bối cảnh & Lý do Dịch chuyển (Context & Rationale)

Trong **Phase 5**, hệ thống **Notification Service** sử dụng cơ chế **In-Memory Retry (ScheduledExecutorService) kết hợp DB Self-Healing Job** để tự động gửi lại thông báo bị lỗi tạm thời (Recoverable). Đây là giải pháp tối ưu cho môi trường Phát triển (Local Dev) và Kiểm thử (TDD/Integration Test) vì tốc độ nhanh và tính gọn nhẹ.

Tuy nhiên, khi đưa hệ thống lên môi trường **Production quy mô lớn** với kiến trúc đa Pod (Distributed Microservices), cơ chế In-Memory bộc lộ các hạn chế:
- **Độ trễ phục hồi khi sập Pod**: Khi một Pod bị sập, các tin nhắn đang xếp hàng chờ retry trong RAM Pod đó sẽ phải chờ đến chu kỳ tiếp theo của DB Self-Healing Job (ví dụ: quét mỗi 2-5 phút) để được phục hồi.
- **Không tận dụng tối đa hạ tầng Event-Driven**: Hiện tại toàn bộ hệ thống đang giao tiếp bất đồng bộ qua Apache Kafka. Việc quản lý retry in-memory làm phân mảnh cơ chế xử lý sự kiện của hệ thống.

**Mục tiêu**: Dịch chuyển an toàn từ **In-Memory Retry** sang **Kafka-based Retry** tận dụng hạ tầng Apache Kafka sẵn có để đạt độ bền vững (Durability) 100%, khả năng chịu lỗi (Fault Tolerance) phân tán và tự động cân bằng tải (Load Balancing) đa node mà **không cần chỉnh sửa core business logic**.

---

## 2. Thiết kế Hạ tầng Kafka (Kafka Topology Design)

Chúng ta sẽ sử dụng giải pháp **Non-blocking Retries** được hỗ trợ mặc định và vô cùng mạnh mẽ bởi **Spring Kafka** thông qua cơ chế tạo các Topic Retry trì hoãn tăng dần và Dead Letter Topic (DLT):

```
[Consumer] requested.v1 ──► (Gửi lỗi Recoverable)
    │
    ├──► [Retry Topic 1] requested-retry-2000 (Delay 2s) ──► [Consumer]
    │                                                            │ (Thất bại)
    ├──► [Retry Topic 2] requested-retry-4000 (Delay 4s) ◄───────┘
    │                                                            │ (Thất bại)
    ├──► [Retry Topic 3] requested-retry-8000 (Delay 8s) ◄───────┘
    │                                                            │ (Thất bại hoàn toàn)
    └──► [Dead Letter Topic] requested-dlt (DLT) ────────────────┘
```

### 2.1. Danh sách Topics cần khởi tạo trên Production:
- **Topic chính**: `notification.requested.v1` (Nhận yêu cầu gửi thông báo từ các microservice khác).
- **Topics Retry (Tự động sinh bởi Spring Kafka)**:
  - `notification.requested-retry-2000` (Thử lại lần 1 - trì hoãn 2 giây).
  - `notification.requested-retry-4000` (Thử lại lần 2 - trì hoãn 4 giây).
  - `notification.requested-retry-8000` (Thử lại lần 3 - trì hoãn 8 giây).
- **Dead Letter Topic (DLT)**:
  - `notification.requested-dlt` (Nơi lưu trữ các tin nhắn đã thất bại hoàn toàn sau 3 lần retry để đội vận hành kiểm tra thủ công).

---

## 3. Triển khai Kỹ thuật (Technical Implementation)

Nhờ áp dụng **Kiến trúc Lục giác (Hexagonal Architecture)** với ranh giới lỏng lẻo qua interface **`RetrySchedulerPort`**, việc dịch chuyển này được thực hiện cực kỳ an toàn bằng cách tạo một Adapter mới cắm vào Port.

### Bước 3.1: Tạo Adapter mới `KafkaRetryScheduler` [NEW]
Adapter này sẽ publish sự kiện retry sang Kafka thay vì in-memory.

```java
package com.rentagf.notification.infrastructure.adapter;

import com.rentagf.notification.application.port.outbound.RetrySchedulerPort;
import com.rentagf.notification.domain.vo.enums.DeliveryChannel;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.context.annotation.Profile;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.kafka.support.KafkaHeaders;
import org.springframework.messaging.support.MessageBuilder;
import org.springframework.stereotype.Component;

import java.time.Duration;
import java.util.UUID;

/**
 * Adapter hạ tầng gửi sự kiện Retry qua Apache Kafka dành cho Production.
 * Chỉ được kích hoạt khi chạy với profile "prod".
 */
@Slf4j
@Component
@Profile("prod")
@RequiredArgsConstructor
public class KafkaRetryScheduler implements RetrySchedulerPort {

    private final KafkaTemplate<String, Object> kafkaTemplate;
    private static final String RETRY_TOPIC = "notification.retry.v1";

    @Override
    public void scheduleRetry(UUID notificationId, DeliveryChannel channel, Duration delay) {
        log.info("Production: Dispatching Kafka retry event for notification {} via channel {} with delay {}s",
                notificationId, channel, delay.toSeconds());

        // Đóng gói thông điệp gửi sang Kafka kèm các headers điều hướng delay
        var message = MessageBuilder.withPayload(notificationId.toString())
                .setHeader(KafkaHeaders.TOPIC, RETRY_TOPIC)
                .setHeader("delivery-channel", channel.name())
                .setHeader("retry-delay-ms", delay.toMillis())
                .build();

        kafkaTemplate.send(message);
    }
}
```

### Bước 3.2: Cấu hình Spring Kafka Non-blocking Retries
Chúng ta khai báo Bean cấu hình để Spring tự động quản lý các topics retry và cơ chế delay backoff:

```java
package com.rentagf.notification.infrastructure.config;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.Profile;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.kafka.retrytopic.RetryTopicConfiguration;
import org.springframework.kafka.retrytopic.RetryTopicConfigurationBuilder;

@Configuration
@Profile("prod")
public class KafkaRetryConfiguration {

    @Bean
    public RetryTopicConfiguration retryTopicConfiguration(KafkaTemplate<String, Object> template) {
        return RetryTopicConfigurationBuilder
                .newInstance()
                .customBackoff((attempt, throwable) -> {
                    // Cấu hình Exponential Backoff khớp với Invariants [INV-N10]
                    if (attempt == 1) return 2000L;  // 2s cho FCM/Email lần đầu
                    if (attempt == 2) return 5000L;  // Tăng dần theo cấu hình kênh
                    return 15000L;
                })
                .maxAttempts(4) // 1 lần đầu + 3 lần retry
                .includeTopic("notification.requested.v1")
                .create(template);
    }
}
```

### Bước 3.3: Chuyển đổi linh hoạt thông qua Spring Profiles
- **Môi trường Dev/Test**: Chạy mặc định hoặc với profile `!prod`. 
  - `InMemoryRetryScheduler` sẽ được kích hoạt thông qua `@Profile("!prod")` hoặc `@ConditionalOnMissingBean`.
- **Môi trường Production**: Chạy với command line argument `-Dspring.profiles.active=prod`.
  - `KafkaRetryScheduler` sẽ tự động được tiêm vào `AsyncNotificationDeliveryService` thay thế hoàn toàn cho in-memory scheduler.

---

## 4. Kế hoạch Triển khai & Kiểm thử (Rollout & Verification Plan)

### 4.1. Các bước triển khai (Deployment Steps):
1. **Khởi tạo Topics**: Đội DevOps chạy script tạo sẵn các topic `notification.requested.v1` và `notification.requested-dlt` trên Kafka cluster Production.
2. **Cập nhật cấu hình môi trường**: Bổ sung các biến môi trường cấu hình Kafka connection vào Kubernetes ConfigMap/Secret.
3. **Rollout Canary**: Deploy phiên bản mới lên 10% số Pods với Profile `prod`, giám sát log hệ thống để kiểm tra xem việc gửi tin lỗi recoverable có đẩy event vào retry topics thành công không.
4. **Full Rollout**: Deploy toàn bộ 100% Pods nếu không phát hiện lỗi.

### 4.2. Kịch bản kiểm thử tích hợp Kafka Retry:
- **Test Case 1**: Gửi tin qua `notification.requested.v1` -> Mock SMTP Offline -> Kiểm tra xem Kafka Broker có nhận được tin nhắn trên topic `notification.requested-retry-2000` hay không.
- **Test Case 2**: Chờ 2 giây -> Kiểm tra xem consumer của retry topic có tự động trigger lại luồng gửi tin bất đồng bộ hay không.
