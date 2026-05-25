package com.rentagf.notification.application.listener;

import com.rentagf.notification.application.event.NotificationReadyEvent;
import com.rentagf.notification.application.service.AsyncNotificationDeliveryService;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;
import org.springframework.transaction.event.TransactionPhase;
import org.springframework.transaction.event.TransactionalEventListener;

/**
 * Listener lắng nghe sự kiện gửi thông báo sau khi transaction chính đã commit.
 */
@Slf4j
@Component
@RequiredArgsConstructor
public class NotificationEventListener {

    private final AsyncNotificationDeliveryService asyncDeliveryService;

    /**
     * Xử lý gửi tin bất đồng bộ.
     * Sử dụng @TransactionalEventListener để Spring tự động trigger AFTER_COMMIT.
     * fallbackExecution = true đảm bảo chạy được trong cả ngữ cảnh non-transactional (như unit test).
     */
    @TransactionalEventListener(phase = TransactionPhase.AFTER_COMMIT, fallbackExecution = true)
    public void handleNotificationReady(NotificationReadyEvent event) {
        log.info("Handling NotificationReadyEvent for notification: {} via channel: {}", 
                event.getNotificationId(), event.getChannel());
        asyncDeliveryService.sendAsync(event.getNotificationId(), event.getChannel());
    }
}
