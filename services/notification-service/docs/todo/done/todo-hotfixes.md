# 🛠️ ĐỀ XUẤT CHỈNH SỬA & TODO HOTFIXES (NOTIFICATION SERVICE)

Tài liệu này tổng hợp toàn bộ checklist công việc và phương án chỉnh sửa chi tiết nhằm khắc phục **4 vấn đề đỏ chí mạng** được phát hiện ngày 22-05-2026, đồng thời tích hợp thiết kế **`idempotency_key`** theo yêu cầu của nhà phát triển để chuẩn hóa theo tiêu chuẩn công nghiệp (API Idempotency).

---

## 📅 DANH SÁCH CÁC TÁC VỤ CẦN THỰC HIỆN NGAY (HOTFIX CHECKLIST)

### 🔴 TASK 1: Chuẩn Hóa Idempotency Key & Sửa Lỗi Khóa Duy Nhất (Chí Mạng)
*   **Mục tiêu**: 
    1.  Chuyển đổi toàn bộ thuật ngữ kỹ thuật từ `event_id` thành `idempotency_key` để đồng bộ cho cả việc nghe Event (Smart Consumer) lẫn nhận API Request (Passive Subscriber).
    2.  Giải phóng hệ thống khỏi lỗi nghẽn gửi tin đa đối tượng (multi-recipient) bằng cách đổi UNIQUE constraint thành composite key.
*   **Phương án xử lý**:
    *   **Database (Flyway)**: 
        *   Cập nhật file [V2__create_notification_tables.sql](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/resources/db/migration/V2__create_notification_tables.sql): Đổi tên cột `event_id` thành `idempotency_key`.
        *   Thay thế constraint cũ:
            ```sql
            -- LOẠI BỎ: CONSTRAINT uk_notifications_event_id UNIQUE (event_id)
            -- THAY THẾ BẰNG: Ràng buộc composite đảm bảo Idempotent theo từng User nhận tin
            CONSTRAINT uk_notifications_idempotency_user UNIQUE (idempotency_key, user_id)
            ```
    *   **Domain Layer**:
        *   [Notification.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/domain/aggregate/Notification.java): Đổi trường `private final String eventId` thành `private final String idempotencyKey`. Cập nhật constructor, static factory `create(...)` và getter `getIdempotencyKey()`.
        *   [NotificationRepository.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/domain/repository/NotificationRepository.java): Thay đổi phương thức `findByEventIdAndUserId` thành `findByIdempotencyKeyAndUserId(String idempotencyKey, UUID userId)`.
        *   [DuplicateEventException.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/domain/errors/DuplicateEventException.java): Cân nhắc đổi tên thành `DuplicateNotificationException.java` hoặc giữ nguyên nhưng cập nhật nội dung message báo lỗi trùng `idempotencyKey`.
    *   **JPA Persistence Adapter**:
        *   [NotificationJpaEntity.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/infrastructure/persistence/jpa/entity/NotificationJpaEntity.java): Đổi tên cột `@Column(name = "event_id")` thành `idempotency_key` và rename thuộc tính thành `idempotencyKey`.
        *   [NotificationJpaRepository.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/infrastructure/persistence/jpa/repository/NotificationJpaRepository.java): Đổi phương thức query thành `Optional<NotificationJpaEntity> findByIdempotencyKeyAndUserId(String idempotencyKey, UUID userId)`.
        *   [NotificationMapper.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/infrastructure/persistence/jpa/mapper/NotificationMapper.java): Cập nhật ánh xạ 2 chiều giữa thuộc tính `idempotencyKey` của Domain và JPA Entity.
        *   [NotificationRepositoryImpl.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/infrastructure/persistence/jpa/NotificationRepositoryImpl.java): Cập nhật implement gọi `findByIdempotencyKeyAndUserId`.
    *   **Application Layer**:
        *   [NotificationApplicationService.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/application/service/NotificationApplicationService.java): Thay đổi tham số `eventId` thành `idempotencyKey` tại phương thức `triggerNotification`. Gọi hàm check trùng `findByIdempotencyKeyAndUserId`.
    *   **Test Suite**:
        *   Cập nhật tất cả các tệp unit test và integration test (`NotificationDomainTest`, `NotificationRepositoryTest`, `AsyncDeliveryTest`) sử dụng thuộc tính mới `idempotencyKey` thay cho `eventId`.

---

### 🔴 TASK 2: Sửa Lỗi Ném Sai Exception Tại Application Service
*   **Mục tiêu**: Đảm bảo khi xảy ra trùng lặp thông báo (vi phạm `idempotency_key`), hệ thống phải trả về mã lỗi **HTTP 409 Conflict** kèm mã `DUPLICATE_EVENT` thay vì HTTP 500 lỗi hệ thống chung.
*   **Phương án xử lý**:
    *   Tại file [NotificationApplicationService.java:L37-L39](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/application/service/NotificationApplicationService.java#L37-L39), thay đổi exception được ném ra:
        ```java
        // TRƯỚC ĐÂY:
        notificationRepository.findByEventIdAndUserId(eventId, userId).ifPresent(n -> {
            throw new IllegalArgumentException("Event already processed: " + eventId);
        });

        // CHỈNH SỬA THÀNH:
        notificationRepository.findByIdempotencyKeyAndUserId(idempotencyKey, userId).ifPresent(n -> {
            throw new DuplicateEventException(idempotencyKey, userId.toString());
        });
        ```

---

### 🔴 TASK 3: Cấu Hình Đúng Tham Số Database Trong Smoke Test
*   **Mục tiêu**: Đảm bảo container của Notification Service có thể kết nối thành công tới container Database PostgreSQL trong môi trường cô lập, vượt qua bài kiểm tra Actuator Health Check.
*   **Phương án xử lý**:
    *   Mở tệp [docker-compose.smoke.yml:L28-32](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/tests/smoke/docker-compose.smoke.yml#L28-32).
    *   Thay thế 3 biến rời rạc `DB_HOST`, `DB_PORT`, `DB_NAME` bằng việc truyền trực tiếp URL kết nối khớp với cấu hình của `application.yml`:
        ```yaml
        # TRƯỚC ĐÂY:
        # - DB_HOST=notification-db
        # - DB_PORT=5432
        # - DB_NAME=notification_db

        # CHỈNH SỬA THÀNH:
        - DB_URL=jdbc:postgresql://notification-db:5432/notification_db
        ```

---

### 🔴 TASK 4: Cập Nhật Tài Liệu Đặc Tả Hàng Đợi Bất Đồng Bộ
*   **Mục tiêu**: Đồng nhất hóa lý thuyết đặc tả và mã nguồn thực tế. Loại bỏ yêu cầu cấu hình `ThreadPoolTaskExecutor` (Anti-pattern đối với Virtual Threads).
*   **Phương án xử lý**:
    *   Cập nhật tài liệu [development-timeline.md:L96-L97](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/docs/development-timeline.md#L96-L97):
        *   Sửa đổi mô tả công việc: Thay vì ghi *Triển khai ThreadPoolTaskExecutor (Worker Pool tối đa 50 threads)*, sửa thành *Tận dụng cơ chế Spring Async Executor mặc định kết hợp Java 21 Virtual Threads (SimpleAsyncTaskExecutor) không giới hạn để đạt hiệu năng xử lý IO tối đa*.
    *   Giữ nguyên mã nguồn hiện tại của [AsyncConfig.java](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/infrastructure/config/AsyncConfig.java) để Spring tự nhận diện Virtual Threads cấu hình từ `application.yml`.
