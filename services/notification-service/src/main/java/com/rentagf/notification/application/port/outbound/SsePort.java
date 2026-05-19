package com.rentagf.notification.application.port.outbound;

/**
 * Port cho SSE delivery — driven adapter.
 * Infrastructure implement để gửi notification qua SSE stream.
 */
public interface SsePort {

    /**
     * Gửi notification tới user qua SSE.
     *
     * @param userId  ID người nhận
     * @param payload JSON payload
     * @return true nếu user đang online và nhận được, false nếu offline
     */
    boolean send(String userId, String payload);
}
