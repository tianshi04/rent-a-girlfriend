# Production Cleanup Plan — Notification Service

> **Mục tiêu:** Xóa / sửa chính xác những gì được thêm vào phục vụ **dev/test**, không thuộc production.  
> **Nguyên tắc:** Chỉ xóa đúng phần test, không đụng vào logic nghiệp vụ thật.

---

## 1. `NotificationController.java` — Xóa 2 endpoint test

### Những gì cần xóa (chính xác từng dòng)

**Import thêm vào (xóa 4 dòng):**
```java
// XÓA — chỉ phục vụ 2 endpoint test bên dưới
import com.rentagf.notification.application.port.inbound.SendNotificationUseCase;
import com.rentagf.notification.application.port.inbound.TriggerNotificationUseCase;
import com.rentagf.notification.interfaces.http.dto.RouteNotificationRequest;
import com.rentagf.notification.interfaces.http.dto.TriggerNotificationRequest;
```

**Field inject thêm vào (xóa 2 dòng):**
```java
// XÓA
private final TriggerNotificationUseCase triggerNotificationUseCase;
private final SendNotificationUseCase sendNotificationUseCase;
```

**Endpoint `/trigger` (xóa toàn bộ method — dòng 85→153 trong diff):**
```java
// XÓA TOÀN BỘ METHOD NÀY
@PostMapping("/trigger")
public ResponseEntity<Void> triggerNotification(
    @RequestBody TriggerNotificationRequest request
) {
    ...  // ~30 dòng
}
```

**Endpoint `/route` (xóa toàn bộ method):**
```java
// XÓA TOÀN BỘ METHOD NÀY
@PostMapping("/route")
public ResponseEntity<Void> routeNotification(
    @RequestBody RouteNotificationRequest request
) {
    ...  // ~30 dòng
}
```

**Blank line thừa (xóa 1 dòng):**
```java
// Trước @RestController có 1 blank line thừa được thêm vào — xóa để khớp style gốc
```

---

## 2. `SseController.java` — Xóa 1 blank line thừa

Diff chỉ có **1 dòng thay đổi duy nhất**: thêm 1 blank line trước `@RestController`.

```java
// XÓA dòng trống thừa này (trước @RestController)

@RestController   // ← giữ nguyên
```

> **Lý do:** SseController không có logic test nào — chỉ bị thêm whitespace. Có thể bỏ qua nếu muốn, nhưng nên giữ diff sạch.

---

## 3. `RouteNotificationRequest.java` — Xóa file

**File:** `internal/com/rentagf/notification/interfaces/http/dto/RouteNotificationRequest.java`

```
XÓA TOÀN BỘ FILE (19 dòng)
- Chỉ phục vụ endpoint /route (test thủ công)
- Không có trong luồng nghiệp vụ thật (nghiệp vụ nhận event qua Kafka)
```

---

## 4. `TriggerNotificationRequest.java` — Xóa file

**File:** `internal/com/rentagf/notification/interfaces/http/dto/TriggerNotificationRequest.java`

```
XÓA TOÀN BỘ FILE (18 dòng)
- Chỉ phục vụ endpoint /trigger (test thủ công)
- Không có trong luồng nghiệp vụ thật
```

---

## 5. Các file khác trong git status

| File | Thay đổi thực sự | Hành động |
|------|-----------------|-----------|
| `Dockerfile` | Thêm multi-stage build (Stage 1 builder) | ✅ **Giữ** — cải thiện production image |
| `Dockerfile.local` | File mới — dành cho `make dev` | ❌ **Không merge vào main** — gitignore hoặc xóa |
| `Makefile` | Thêm target `up`, `dev`, `down` | ✅ **Giữ** — target `up` dùng được cho CI |
| `docker-compose.local.yml` | File mới — dành cho `make dev` | ❌ **Không merge vào main** |
| `WebConfig.java` | CORS config — tạo để test local với Live Server | ❌ **Xóa cả file** — production dùng Istio Waypoint xử lý CORS tại L7 |
| `application.yml` | Chỉ thêm 2 dòng comment + 1 whitespace | ✅ **Giữ** — không ảnh hưởng production |
| `package.json` + `package-lock.json` | Từ lần thử npm install | ❌ **Xóa** |
| `resources/static/` | Toàn bộ dev UI | ❌ **Xóa** |
| `templates.yaml` | Template thông báo | ✅ **Giữ** |

---

## 6. Checklist xóa theo thứ tự

```
[ ] 1. Xóa file: dto/TriggerNotificationRequest.java
[ ] 2. Xóa file: dto/RouteNotificationRequest.java
[ ] 3. Sửa NotificationController.java:
        - Xóa 4 import (TriggerNotificationUseCase, SendNotificationUseCase, 2 DTO)
        - Xóa 2 field inject
        - Xóa method triggerNotification() + @PostMapping("/trigger")
        - Xóa method routeNotification() + @PostMapping("/route")
        - Xóa 1 blank line thừa trước @RestController
[ ] 4. Sửa SseController.java: xóa 1 blank line thừa trước @RestController
[ ] 5. Xóa file: infrastructure/config/WebConfig.java
        → Production dùng Istio Waypoint xử lý CORS tại L7, không cần Spring CorsFilter
[ ] 6. Xóa resources/static/ (toàn bộ thư mục)
[ ] 7. Xóa package.json, package-lock.json
[ ] 8. Chạy: ./gradlew build -x test  → đảm bảo compile pass
[ ] 9. Chạy: ./gradlew test           → đảm bảo không có test nào broken
```

---

*Cập nhật: 2026-05-25 — dựa trên `git diff HEAD` từng file.*
