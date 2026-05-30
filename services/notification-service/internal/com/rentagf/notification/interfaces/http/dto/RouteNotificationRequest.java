package com.rentagf.notification.interfaces.http.dto;

import java.util.List;
import java.util.Map;
import java.util.UUID;

/**
 * Request DTO để gửi thông báo thông qua Smart Routing Engine (định tuyến động).
 */
public record RouteNotificationRequest(
    UUID userId,
    String eventId,
    String type,
    String priority,
    Map<String, Object> payload,
    List<String> channels
) {
}
