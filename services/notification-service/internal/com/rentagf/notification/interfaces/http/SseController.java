package com.rentagf.notification.interfaces.http;

import com.rentagf.notification.application.port.inbound.NotificationSubscriptionUseCase;
import com.rentagf.notification.infrastructure.sse.SseConnectionRegistry;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.MediaType;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.servlet.mvc.method.annotation.SseEmitter;

import java.io.IOException;
import java.util.UUID;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ScheduledFuture;
import java.util.concurrent.TimeUnit;

/**
 * Controller Inbound Adapter xử lý kết nối Server-Sent Events (SSE).
 * Duy trì kết nối thời gian thực non-blocking và quản lý heartbeat định kỳ [INV-N05].
 */
@RestController
@RequestMapping("/v1/notifications")
public class SseController {

    private static final Logger log = LoggerFactory.getLogger(SseController.class);

    // Timeout kết nối SSE (30 phút = 1,800,000 miligiây)
    private static final long SSE_TIMEOUT_MS = 1800000L;
    
    // Heartbeat Interval chuẩn theo rule [INV-N05] (15 giây)
    private static final long HEARTBEAT_INTERVAL_SECONDS = 15L;

    private final SseConnectionRegistry connectionRegistry;
    private final NotificationSubscriptionUseCase subscriptionUseCase;
    private final ScheduledExecutorService heartbeatScheduler;

    public SseController(SseConnectionRegistry connectionRegistry, NotificationSubscriptionUseCase subscriptionUseCase) {
        this.connectionRegistry = connectionRegistry;
        this.subscriptionUseCase = subscriptionUseCase;
        
        // Sử dụng Virtual Thread Scheduled Executor để tối ưu hiệu năng I/O blocking
        this.heartbeatScheduler = Executors.newScheduledThreadPool(1, 
                Thread.ofVirtual().name("sse-heartbeat-", 0).factory());
    }

    /**
     * Bắt tay kết nối thời gian thực SSE.
     * Cung cấp luồng streaming một chiều từ Server xuống Client.
     *
     * @param userId UUID của User nhận tin (được Istio verify và inject qua header)
     */
    @GetMapping(value = "/stream", produces = MediaType.TEXT_EVENT_STREAM_VALUE)
    public SseEmitter streamNotifications(@RequestHeader("user-id") UUID userId) {
        log.info("Incoming SSE connection request from user: {}", userId);

        // 1. Khởi tạo SseEmitter với timeout 30 phút
        SseEmitter emitter = new SseEmitter(SSE_TIMEOUT_MS);

        // 2. Lập lịch gửi Heartbeat `: ping\n\n` mỗi 15s để chống idle timeout [INV-N05]
        ScheduledFuture<?> heartbeatTask = heartbeatScheduler.scheduleAtFixedRate(() -> {
            try {
                log.debug("Sending heartbeat ping to user {}", userId);
                emitter.send(SseEmitter.event().comment("ping"));
            } catch (IOException e) {
                log.warn("Failed to send heartbeat to user {}. Closing connection...", userId);
                emitter.completeWithError(e);
            }
        }, HEARTBEAT_INTERVAL_SECONDS, HEARTBEAT_INTERVAL_SECONDS, TimeUnit.SECONDS);

        // 3. Đăng ký callbacks xử lý dọn dẹp kết nối chết [INV-N04]
        Runnable cleanupTask = () -> {
            log.info("SSE connection closed/timeout for user: {}", userId);
            heartbeatTask.cancel(true); // Hủy lịch gửi heartbeat ngầm để tránh rò rỉ CPU
            connectionRegistry.unregister(userId, emitter); // Xóa khỏi Registry
        };

        emitter.onCompletion(cleanupTask);
        emitter.onTimeout(cleanupTask);
        emitter.onError((ex) -> {
            log.error("SSE connection error occurred for user: {}, message: {}", userId, ex.getMessage());
            cleanupTask.run();
        });

        // 4. Lưu kết nối vào Registry cục bộ (Tự động kích hoạt subscribe Redis ở kết nối đầu)
        connectionRegistry.register(userId, emitter);

        // 4.5. Kích hoạt nghiệp vụ qua Inbound Port (Ví dụ: Đẩy các thông báo chưa đọc cũ)
        try {
            subscriptionUseCase.subscribe(userId);
        } catch (Exception e) {
            log.error("Failed to execute subscription use case for user: {}", userId, e);
        }

        // 5. Gửi gói tin bắt tay đầu tiên thông báo kết nối thành công
        try {
            emitter.send(SseEmitter.event().comment("connected"));
            log.info("SSE handshake established successfully for user: {}", userId);
        } catch (IOException e) {
            log.error("Failed to establish SSE handshake for user: {}", userId, e);
            emitter.completeWithError(e);
        }

        return emitter;
    }
}
