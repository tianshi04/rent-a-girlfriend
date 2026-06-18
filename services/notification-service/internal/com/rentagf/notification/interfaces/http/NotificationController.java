package com.rentagf.notification.interfaces.http;

import com.rentagf.notification.application.port.inbound.FetchInboxUseCase;
import com.rentagf.notification.application.port.inbound.MarkAllAsReadUseCase;
import com.rentagf.notification.application.port.inbound.MarkAsReadUseCase;
import com.rentagf.notification.application.port.inbound.SendNotificationUseCase;
import com.rentagf.notification.application.port.inbound.TriggerNotificationUseCase;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.interfaces.http.dto.FetchInboxResponse;
import com.rentagf.notification.interfaces.http.dto.MarkAllReadResponse;
import com.rentagf.notification.interfaces.http.dto.RouteNotificationRequest;
import com.rentagf.notification.interfaces.http.dto.TriggerNotificationRequest;
import java.util.List;
import java.util.UUID;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

/**
 * HTTP Adapter (REST Controller) cho các API của Notification Service. Tuyệt đối KHÔNG tự cài đặt
 * logic xác thực JWT (đã được offload cho Istio Waypoint). Lấy danh tính người dùng trực tiếp qua
 * HTTP Header `user-id`.
 */
@RestController
@RequestMapping("/v1/notifications")
@RequiredArgsConstructor
public class NotificationController {

  private final FetchInboxUseCase fetchInboxUseCase;
  private final MarkAsReadUseCase markAsReadUseCase;
  private final MarkAllAsReadUseCase markAllAsReadUseCase;
  private final TriggerNotificationUseCase triggerNotificationUseCase;
  private final SendNotificationUseCase sendNotificationUseCase;

  /**
   * Tải danh sách thông báo phân trang cursor-based của người dùng. GET
   * /v1/notifications?cursor=...&limit=20&unreadOnly=false
   */
  @GetMapping
  public ResponseEntity<FetchInboxResponse> fetchInbox(
      @RequestHeader("user-id") String userIdHeader,
      @RequestParam(name = "cursor", required = false) String cursor,
      @RequestParam(name = "limit", defaultValue = "20") int limit,
      @RequestParam(name = "unreadOnly", defaultValue = "false") boolean unreadOnly) {
    UUID userId = getAndValidateUserId(userIdHeader);
    if (limit <= 0 || limit > 100) {
      throw new IllegalArgumentException("Limit must be between 1 and 100");
    }

    com.rentagf.notification.domain.vo.InboxCursor decodedCursor =
        com.rentagf.notification.interfaces.http.codec.CursorCodec.decode(cursor);
    List<Notification> notifications =
        fetchInboxUseCase.fetchInbox(userId, decodedCursor, limit, unreadOnly);
    FetchInboxResponse response = FetchInboxResponse.of(notifications, limit);

    return ResponseEntity.ok(response);
  }

  /** Đánh dấu một thông báo cụ thể là đã đọc. PATCH /v1/notifications/{id}/read */
  @PatchMapping("/{id}/read")
  public ResponseEntity<Void> markAsRead(
      @RequestHeader("user-id") String userIdHeader, @PathVariable("id") UUID notificationId) {
    UUID userId = getAndValidateUserId(userIdHeader);
    markAsReadUseCase.markAsRead(notificationId, userId);
    return ResponseEntity.noContent().build();
  }

  /**
   * Đánh dấu toàn bộ thông báo chưa đọc của người dùng là đã đọc. PATCH /v1/notifications/read-all
   */
  @PatchMapping("/read-all")
  public ResponseEntity<MarkAllReadResponse> markAllAsRead(
      @RequestHeader("user-id") String userIdHeader) {
    UUID userId = getAndValidateUserId(userIdHeader);
    int affectedRows = markAllAsReadUseCase.markAllAsRead(userId);
    return ResponseEntity.ok(new MarkAllReadResponse(affectedRows));
  }

  /**
   * Gửi/Trigger thông báo thủ công phục vụ việc demo và kiểm thử. POST /v1/notifications/trigger
   */
  @PostMapping("/trigger")
  public ResponseEntity<Void> triggerNotification(@RequestBody TriggerNotificationRequest request) {
    UUID userId = request.userId() != null ? request.userId() : UUID.randomUUID();
    String eventId =
        request.eventId() != null && !request.eventId().trim().isEmpty()
            ? request.eventId()
            : "demo-evt-" + UUID.randomUUID();

    com.rentagf.notification.domain.vo.enums.NotificationType type =
        com.rentagf.notification.domain.vo.enums.NotificationType.valueOf(
            request.type() != null ? request.type().toUpperCase() : "TRANSACTIONAL");

    com.rentagf.notification.domain.vo.enums.NotificationPriority priority =
        com.rentagf.notification.domain.vo.enums.NotificationPriority.valueOf(
            request.priority() != null ? request.priority().toUpperCase() : "HIGH");

    com.rentagf.notification.domain.vo.enums.DeliveryChannel channel =
        com.rentagf.notification.domain.vo.enums.DeliveryChannel.valueOf(
            request.channel() != null ? request.channel().toUpperCase() : "SSE");

    triggerNotificationUseCase.triggerNotification(
        userId, eventId, type, priority, request.payload(), channel);
    return ResponseEntity.status(HttpStatus.CREATED).build();
  }

  /**
   * Gửi thông báo tự động thông qua bộ định tuyến thông minh (Smart Routing Engine). POST
   * /v1/notifications/route
   */
  @PostMapping("/route")
  public ResponseEntity<Void> routeNotification(@RequestBody RouteNotificationRequest request) {
    UUID userId = request.userId() != null ? request.userId() : UUID.randomUUID();
    String eventId =
        request.eventId() != null && !request.eventId().trim().isEmpty()
            ? request.eventId()
            : "route-evt-" + UUID.randomUUID();

    com.rentagf.notification.domain.vo.enums.NotificationType type =
        com.rentagf.notification.domain.vo.enums.NotificationType.valueOf(
            request.type() != null ? request.type().toUpperCase() : "TRANSACTIONAL");

    com.rentagf.notification.domain.vo.enums.NotificationPriority priority =
        com.rentagf.notification.domain.vo.enums.NotificationPriority.valueOf(
            request.priority() != null ? request.priority().toUpperCase() : "HIGH");

    java.util.Map<String, Object> policyOverrides = new java.util.HashMap<>();
    if (request.channels() != null) {
      policyOverrides.put("channels", request.channels());
    }

    Notification notification =
        Notification.create(userId, eventId, type, priority, request.payload(), policyOverrides);
    sendNotificationUseCase.routeAndSend(notification);
    return ResponseEntity.status(HttpStatus.CREATED).build();
  }

  /** Helper validate và parse UUID từ Header user-id. */
  private UUID getAndValidateUserId(String userIdHeader) {
    if (userIdHeader == null || userIdHeader.trim().isEmpty()) {
      throw new IllegalArgumentException("Missing required 'user-id' header");
    }
    try {
      return UUID.fromString(userIdHeader.trim());
    } catch (IllegalArgumentException e) {
      throw new IllegalArgumentException("Invalid 'user-id' header format. Must be a valid UUID");
    }
  }
}
