# ADR 0010: Cơ chế Retry Bất đồng bộ Không Block Luồng - InMemory kết hợp DB Self-Healing vs Kafka-based Retry

**Trạng thái:** Chấp nhận (Accepted)  
**Ngày:** 2026-05-24  

---

## 1. Ngữ cảnh (Context)

Trong **Phase 5: Outbound Providers & Retry Execution**, hệ thống cần triển khai cơ chế **Retry trì hoãn (Exponential Backoff)** để tự động gửi lại thông báo khi gặp các lỗi tạm thời (Recoverable) từ các nhà cung cấp bên ngoài (Firebase, SMTP).

Hiện tại, hệ thống của chúng ta đã có sẵn hạ tầng **Apache Kafka** làm Message Broker cho kiến trúc Event-Driven Microservices. Tuy nhiên, việc thiết lập cơ chế Retry có hai lựa chọn kiến trúc lớn:

### Phương án A: In-Memory Retry (ScheduledExecutorService)
- Hệ thống lên lịch retry bằng cách submit một task trì hoãn vào `ScheduledExecutorService` của Java, chạy trên **Virtual Threads**.
- **Ưu điểm**: Cực kỳ gọn nhẹ, không tốn tài nguyên mạng hay hạ tầng, tốc độ xử lý nhanh, dễ dàng viết Unit/Integration Tests.
- **Nhược điểm**: Toàn bộ hàng đợi retry nằm trên RAM của Pod. Nếu Pod bị crash hoặc khi deploy code mới (restart server), các task retry đang xếp hàng chờ sẽ bị **mất sạch**, khiến thông báo bị kẹt ở trạng thái `PROCESSING`.

### Phương án B: Kafka-based Retry (Kafka Delayed / Retry Topics)
- Tận dụng hạ tầng Kafka sẵn có để lưu trữ hàng đợi retry bền vững (Persistent). 
- Khi gửi tin bị lỗi, hệ thống publish một event retry vào một topic đặc biệt (ví dụ: `notification.retry.2s` hoặc `notification.retry.4s`) kèm theo header cấu hình. Consumer sẽ đọc và gửi lại.
- **Ưu điểm**: Bền vững 100% (không sợ mất mát khi restart), chịu lỗi phân tán tuyệt đối, tự động scale tải đa node.
- **Nhược điểm**: Setup hạ tầng rất phức tạp, cần cấu hình các topic trì hoãn riêng biệt, tăng độ trễ và gánh nặng quản lý broker cho đội Devops trong giai đoạn đầu phát triển.

---

## 2. Quyết định Kiến trúc (Decisions)

Để đạt được sự cân bằng tối ưu giữa **tốc độ phát triển sản phẩm (Time-to-Market)** và **độ tin cậy giao hàng chuẩn Production (Resiliency)**, hệ thống quyết định áp dụng giải pháp kiến trúc sau:

### Quyết định 2.1: Triển khai InMemory Retry kết hợp DB Self-Healing cho MVP (Phase 5)
Hệ thống sử dụng giải pháp **InMemory Retry** làm cơ chế thực thi chính trong Phase 5 vì tính gọn nhẹ và dễ demo, đồng thời kết hợp thêm **DB Self-Healing (Tự phục hồi qua Database)** để triệt tiêu hoàn toàn nhược điểm mất mát dữ liệu của In-Memory:

1. **InMemory Execution**: Lên lịch retry trì hoãn bằng `ScheduledExecutorService` trên luồng Virtual Threads cực kỳ nhẹ.
2. **DB Self-Healing (Bảo vệ dự phòng)**: Xây dựng một **Startup Job** và một **Cron Job** (sử dụng `@Scheduled` của Spring) chạy định kỳ (mỗi 2-5 phút) quét cơ sở dữ liệu để tìm các thông báo đang ở trạng thái `PROCESSING` quá lâu (ví dụ > 5 phút) mà chưa hoàn thành. Hệ thống sẽ tự động phục hồi và nạp các thông báo này vào luồng gửi lại.
   - *Kết quả*: Ngay cả khi server bị sập đột ngột và mất sạch hàng đợi trong RAM, khi server khởi động lại hoặc qua chu kỳ quét DB, hệ thống sẽ tự động phục hồi và gửi tiếp, đảm bảo độ bền vững đạt **100%**.

---

### Quyết định 2.2: Thiết kế lỏng qua Port (`RetrySchedulerPort`) để sẵn sàng đổi sang Kafka
Tuân thủ nghiêm ngặt **Kiến trúc Lục giác (Hexagonal Architecture)** để chuẩn bị cho sự thay đổi (**Change Propagation Mindset**):

- Tách biệt hoàn toàn cơ chế Retry ra khỏi Core Logic thông qua interface **`RetrySchedulerPort`**:
  ```java
  public interface RetrySchedulerPort {
      void scheduleRetry(UUID notificationId, DeliveryChannel channel, Duration delay);
  }
  ```
- **Lộ trình dịch chuyển (Migration Plan)**:
  - **Giai đoạn Hiện tại (MVP/Phase 5)**: Sử dụng `InMemoryRetryScheduler` triển khai port này.
  - **Giai đoạn Production quy mô lớn**: Khi lượng tải tăng cao và cần độ bền vững phân tán tuyệt đối, DevOps và Dev Team chỉ cần viết thêm một adapter **`KafkaRetryScheduler`** để triển khai `RetrySchedulerPort` và cấu hình Spring cắm adapter mới vào, **hoàn toàn không cần thay đổi một dòng code nào trong core domain hay application services!**

---

## 3. Bản thiết kế Lộ trình dịch chuyển sang Kafka Retry (Migration Plan)

Khi hệ thống dịch chuyển sang Kafka-based Retry, cấu trúc hạ tầng sẽ được setup như sau:

```
[AsyncNotificationDeliveryService]
       │
       ▼ (Gửi tin lỗi Recoverable)
 [RetrySchedulerPort]
       │
       ▼ (Cắm Adapter mới)
  [KafkaRetryScheduler]
       │
       ▼ (Publish Event với Delay Header)
   [Kafka Broker] ──► Topic: `notification-retry-topic` 
                            │ (Delayed via Kafka Consumer Pause/Resume 
                            │  hoặc Spring Kafka @RetryableTopic)
                            ▼
                     [Kafka Consumer] ──► [AsyncNotificationDeliveryService] (Thử lại)
```

### 3.1. Cấu hình Hạ tầng Kafka Retry cần thiết:
Chúng ta sẽ sử dụng tính năng **Non-blocking Retries** của Spring Kafka (được hỗ trợ mạnh mẽ qua `@RetryableTopic` và `@DltHandler`):
1. **Topic chính**: `notification.requested.v1`
2. **Topic Retry tự động**: Spring Kafka sẽ tự động sinh ra các topic trung gian:
   - `notification.requested-retry-2000` (Thử lại sau 2 giây)
   - `notification.requested-retry-4000` (Thử lại sau 4 giây)
   - `notification.requested-retry-8000` (Thử lại sau 8 giây)
3. **Dead Letter Topic (DLT)**: `notification.requested-dlt` (Nơi chứa các tin nhắn đã thất bại hoàn toàn sau 3 lần retry).

### 3.2. Code mẫu cấu hình Spring Kafka Retry khi chuyển đổi:
```java
@Configuration
@EnableKafka
public class KafkaRetryConfiguration {

    @Bean
    public RetryTopicConfiguration retryTopicConfiguration(KafkaTemplate<String, Object> template) {
        return RetryTopicConfigurationBuilder
                .newInstance()
                .customBackoff((attempt, throwable) -> {
                    // Cấu hình Exponential Backoff động dựa trên attempt
                    if (attempt == 1) return 2000L; // 2s
                    if (attempt == 2) return 4000L; // 4s
                    return 8000L; // 8s
                })
                .maxAttempts(4) // 1 lần đầu + 3 lần retry
                .includeTopic("notification.requested.v1")
                .dltHandlerMethod("notificationDltHandler", "processDlt")
                .create(template);
    }
}
```

---

## 4. Hệ quả (Consequences)

### Tích cực (Positives)
- **Tối ưu hóa thời gian phát triển**: MVP chạy mượt mà ngay lập tức mà không tốn công cấu hình phức tạp các topic Kafka trì hoãn, giảm thiểu rủi ro nghẽn network.
- **Tính chịu lỗi tuyệt vời**: DB Self-Healing loại bỏ hoàn toàn rủi ro mất mát dữ liệu in-memory khi crash/restart Pod, mang lại sự an tâm tuyệt đối.
- **Ranh giới cô lập hoàn hảo**: Hexagonal Architecture bảo vệ mã nguồn core trước các thay đổi hạ tầng trong tương lai.

### Đánh đổi (Trade-offs)
- **Độ trễ phục hồi khi sập**: Khi Pod bị sập, các tin nhắn đang xếp hàng in-memory sẽ phải đợi chu kỳ tiếp theo của Cron Job (vd: 2 phút) quét DB để được gửi lại. Trong khi đó, nếu dùng Kafka, Pod khác sẽ nhận lại partition và gửi lại gần như ngay lập tức. Đây là đánh đổi hoàn toàn chấp nhận được trong giai đoạn MVP.
