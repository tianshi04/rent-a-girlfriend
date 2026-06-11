package com.rentagf.notification.interfaces.http.dto;

import java.util.Map;
import java.util.UUID;

/** Request DTO để kích hoạt trigger thông báo thủ công phục vụ việc demo và kiểm thử. */
public record TriggerNotificationRequest(
    UUID userId,
    String eventId,
    String type,
    String priority,
    Map<String, Object> payload,
    String channel) {}
