# InMemoryConnectionStatePort

## Mục đích
- Cung cấp một **implementation** in‑memory cho `ConnectionStatePort` dùng trong **unit test** và **local development**.
- Lưu danh sách các `UUID` của người dùng đang online trong một `Set` thread‑safe (dựa trên `ConcurrentHashMap`).
- Các phương thức:
  - `setOnline(UUID userId, long ttlSeconds)`: thêm `userId` vào set (không xử lý TTL).
  - `setOffline(UUID userId)`: xóa `userId` khỏi set.
  - `isOnline(UUID userId)`: kiểm tra `userId` có trong set.
  - `refreshHeartbeat(UUID userId, long ttlSeconds)`: **no‑op** cho phiên bản in‑memory.

## Khi nào dùng
- **Test / Development**: Khi chạy các test như `AsyncDeliveryTest` không muốn phụ thuộc vào Redis hoặc một store phân tán thực tế. Bean này cung cấp một cách nhanh chóng, không cần cấu hình hạ tầng ngoài.

## Không dùng trong Production
- Production yêu cầu trạng thái online/offline **được chia sẻ giữa các instance** và có **TTL** để tự hết hạn. Điều này chỉ có thể đạt được bằng một store phân tán (Redis, Hazelcast, …).
- Dự án đã có `RedisConnectionStateAdapter` (hoặc một adapter khác) được cấu hình trong `application.yml` để thực hiện chức năng này.

## Đảm bảo an toàn tránh bean in‑memory bị bật trong Production
- Thêm annotation **profile** hoặc **conditional** vào lớp:
```java
@Profile("test") // hoặc "dev"
@Component
public class InMemoryConnectionStatePort implements ConnectionStatePort { … }
```
hoặc
```java
@Component
@ConditionalOnMissingBean(ConnectionStatePort.class)
public class InMemoryConnectionStatePort implements ConnectionStatePort { … }
```
- Cập nhật `application.yml` để có profile `test` riêng, ví dụ:
```yaml
spring:
  profiles: test
```
- Khi chạy ở production, Spring sẽ tải `RedisConnectionStateAdapter`; bean `InMemoryConnectionStatePort` sẽ **không được tạo**.

## Tóm tắt quyết định
- **Test / dev**: giữ `InMemoryConnectionStatePort` để các test (như `AsyncDeliveryTest`) chạy nhanh, không phụ thuộc vào Redis.
- **Production**: không commit/triển khai bean này; sử dụng `RedisConnectionStateAdapter` hoặc một adapter khác được cấu hình trong `application.yml`.
- Đảm bảo **chỉ một bean** tồn tại bằng cách dùng `@Profile` hoặc `@ConditionalOnMissingBean`.

---
*File này được tạo để tài liệu hoá quyết định thiết kế và hướng dẫn sử dụng cho các nhà phát triển tiếp theo.*
