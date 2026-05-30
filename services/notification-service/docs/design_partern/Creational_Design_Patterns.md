# 🏗️ CREATIONAL DESIGN PATTERNS

## 1. Static Factory Method (Phương thức Khởi tạo Tĩnh)

*   **Dẫn chứng trong dự án**:
    *   [Notification.java:L55-L73](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/domain/aggregate/Notification.java#L55-L73) (Aggregate Root)
    ```java
    public static Notification create(UUID userId, String idempotencyKey, NotificationType type,
                                      NotificationPriority priority, Map<String, Object> payload,
                                      Map<String, Object> policyOverrides) {
        Instant now = Instant.now();
        return new Notification(
                UUID.randomUUID(),
                userId,
                idempotencyKey,
                type,
                priority,
                payload,
                policyOverrides,
                NotificationStatus.PENDING,
                null,
                now,
                now,
                new ArrayList<>()
        );
    }
    ```

    *   [FetchInboxResponse.java:L48-L64](file:///e:/LEARN/rent-a-girlfriend/services/notification-service/internal/com/rentagf/notification/interfaces/http/dto/FetchInboxResponse.java#L48-L64) (DTO & Paging Factory)
    ```java
    public static FetchInboxResponse of(List<Notification> notifications, int limit) {
        boolean hasMore = notifications.size() > limit;
        List<Notification> items = hasMore ? notifications.subList(0, limit) : notifications;
        List<NotificationDto> dtoList = items.stream().map(NotificationDto::from).toList();
        
        String nextCursor = null;
        if (hasMore && !items.isEmpty()) {
            Notification lastItem = items.get(items.size() - 1);
            InboxCursor cursor = new InboxCursor(lastItem.getCreatedAt(), lastItem.getId());
            nextCursor = CursorCodec.encode(cursor);
        }
        return new FetchInboxResponse(dtoList, new PagingInfo(nextCursor, hasMore));
    }
    ```

### 📝 Tại sao sử dụng và hoạt động thế nào?
*   **Vấn đề**: Hàm khởi tạo (Constructor) thông thường của `Notification` đòi hỏi rất nhiều tham số kỹ thuật (như `id`, `status`, `createdAt`, `updatedAt`, và list `attempts` rỗng). Khi Client (Use Case) muốn tạo mới một Notification nghiệp vụ, họ không nên tự quyết định hoặc quan tâm tới các chi tiết kỹ thuật này. Việc sử dụng trực tiếp constructor `new` sẽ làm lộ chi tiết cấu trúc nội bộ của Aggregate Root.
*   **Giải pháp**: Cung cấp một phương thức tĩnh `Notification.create(...)` có ý nghĩa nghiệp vụ rõ ràng (Ubiquitous Language). Lớp Client chỉ truyền vào các thông tin nghiệp vụ cốt lõi (User, Event, Priority, Payload). Phương thức này sẽ tự động khởi tạo các thông tin kỹ thuật mặc định (`UUID.randomUUID()`, trạng thái `PENDING`, thời gian khởi tạo `Instant.now()`, danh sách attempt rỗng) rồi trả về thực thể hoàn chỉnh.

### 📈 Ưu điểm đạt được
1.  **Tính đóng gói cao (Encapsulation)**: Che giấu chi tiết khởi tạo và các trường kỹ thuật của Entity khỏi tầng Application.
2.  **Tên gọi có ý nghĩa (Descriptive Name)**: Thay vì dùng constructor chung chung, tên phương thức `create` chỉ rõ hành động nghiệp vụ sinh mới thông báo.
3.  **Bảo vệ Invariants**: Đảm bảo thực thể sinh mới luôn ở trạng thái hợp lệ ban đầu (`PENDING`).
