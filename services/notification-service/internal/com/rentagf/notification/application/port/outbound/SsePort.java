package com.rentagf.notification.application.port.outbound;

/**
 * Port cho SSE delivery — driven adapter. Kế thừa NotificationSender phục vụ Strategy Pattern.
 * Infrastructure implement để gửi notification qua SSE stream.
 */
public interface SsePort extends NotificationSender {}
