package com.rentagf.notification.interfaces.http.dto;

import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.vo.InboxCursor;
import com.rentagf.notification.interfaces.http.codec.CursorCodec;
import java.time.Instant;
import java.util.List;
import java.util.Map;
import java.util.UUID;

/** DTO đại diện cho response lấy danh sách thông báo phân trang cursor-based. */
public record FetchInboxResponse(List<NotificationDto> data, PagingInfo paging) {
  public record NotificationDto(
      UUID id,
      UUID userId,
      String eventId,
      String type,
      String priority,
      Map<String, Object> payload,
      String status,
      Instant readAt,
      Instant createdAt,
      Instant updatedAt) {
    public static NotificationDto from(Notification n) {
      return new NotificationDto(
          n.getId(),
          n.getUserId(),
          n.getEventId(),
          n.getType().name(),
          n.getPriority().name(),
          n.getPayload(),
          n.getStatus().name(),
          n.getReadAt(),
          n.getCreatedAt(),
          n.getUpdatedAt());
    }
  }

  public record PagingInfo(String nextCursor, boolean hasMore) {}

  /**
   * Khởi tạo FetchInboxResponse từ danh sách thông báo. Hỗ trợ tự động cắt bớt phần tử thừa (phần
   * tử thứ limit + 1) và tính toán nextCursor.
   *
   * @param notifications Danh sách thông báo (đã được query với kích thước limit + 1)
   * @param limit Giới hạn kích thước trang yêu cầu từ client
   * @return FetchInboxResponse
   */
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
}
