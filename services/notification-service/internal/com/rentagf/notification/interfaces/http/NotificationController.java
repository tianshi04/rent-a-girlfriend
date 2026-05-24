package com.rentagf.notification.interfaces.http;

import com.rentagf.notification.application.port.inbound.FetchInboxUseCase;
import com.rentagf.notification.application.port.inbound.MarkAllAsReadUseCase;
import com.rentagf.notification.application.port.inbound.MarkAsReadUseCase;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.interfaces.http.dto.FetchInboxResponse;
import com.rentagf.notification.interfaces.http.dto.MarkAllReadResponse;
import lombok.RequiredArgsConstructor;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.UUID;

/**
 * HTTP Adapter (REST Controller) cho các API của Notification Service.
 * Tuyệt đối KHÔNG tự cài đặt logic xác thực JWT (đã được offload cho Istio Waypoint).
 * Lấy danh tính người dùng trực tiếp qua HTTP Header `user-id`.
 */
@RestController
@RequestMapping("/v1/notifications")
@RequiredArgsConstructor
public class NotificationController {

    private final FetchInboxUseCase fetchInboxUseCase;
    private final MarkAsReadUseCase markAsReadUseCase;
    private final MarkAllAsReadUseCase markAllAsReadUseCase;

    /**
     * Tải danh sách thông báo phân trang cursor-based của người dùng.
     * GET /v1/notifications?cursor=...&limit=20&unread_only=false
     */
    @GetMapping
    public ResponseEntity<FetchInboxResponse> fetchInbox(
        @RequestHeader("user-id") String userIdHeader,
        @RequestParam(name = "cursor", required = false) String cursor,
        @RequestParam(name = "limit", defaultValue = "20") int limit,
        @RequestParam(name = "unread_only", defaultValue = "false") boolean unreadOnly
    ) {
        UUID userId = getAndValidateUserId(userIdHeader);
        if (limit <= 0 || limit > 100) {
            throw new IllegalArgumentException("Limit must be between 1 and 100");
        }

        com.rentagf.notification.domain.vo.InboxCursor decodedCursor = com.rentagf.notification.interfaces.http.codec.CursorCodec.decode(cursor);
        List<Notification> notifications = fetchInboxUseCase.fetchInbox(userId, decodedCursor, limit, unreadOnly);
        FetchInboxResponse response = FetchInboxResponse.of(notifications, limit);

        return ResponseEntity.ok(response);
    }

    /**
     * Đánh dấu một thông báo cụ thể là đã đọc.
     * PATCH /v1/notifications/{id}/read
     */
    @PatchMapping("/{id}/read")
    public ResponseEntity<Void> markAsRead(
        @RequestHeader("user-id") String userIdHeader,
        @PathVariable("id") UUID notificationId
    ) {
        UUID userId = getAndValidateUserId(userIdHeader);
        markAsReadUseCase.markAsRead(notificationId, userId);
        return ResponseEntity.noContent().build();
    }

    /**
     * Đánh dấu toàn bộ thông báo chưa đọc của người dùng là đã đọc.
     * PATCH /v1/notifications/read-all
     */
    @PatchMapping("/read-all")
    public ResponseEntity<MarkAllReadResponse> markAllAsRead(
        @RequestHeader("user-id") String userIdHeader
    ) {
        UUID userId = getAndValidateUserId(userIdHeader);
        int affectedRows = markAllAsReadUseCase.markAllAsRead(userId);
        return ResponseEntity.ok(new MarkAllReadResponse(affectedRows));
    }

    /**
     * Helper validate và parse UUID từ Header user-id.
     */
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
