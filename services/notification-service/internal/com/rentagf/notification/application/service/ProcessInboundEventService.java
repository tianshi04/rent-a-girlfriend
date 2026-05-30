package com.rentagf.notification.application.service;

import com.rentagf.notification.application.port.inbound.ProcessInboundEventUseCase;
import com.rentagf.notification.application.port.inbound.SendNotificationUseCase;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.errors.DuplicateEventException;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;

import java.util.List;

/**
 * Triển khai ProcessInboundEventUseCase.
 *
 * <p>Trách nhiệm duy nhất: nhận danh sách Notification đã được dịch và định tuyến từng cái
 * thông qua {@link SendNotificationUseCase}.
 *
 * <p>KHÔNG chứa bất kỳ logic parse hay translate nào – đảm bảo Application Layer
 * hoàn toàn độc lập với Interfaces Layer (Hexagonal Architecture RULE 4).
 */
@Slf4j
@Service
@RequiredArgsConstructor
public class ProcessInboundEventService implements ProcessInboundEventUseCase {

    private final SendNotificationUseCase sendNotificationUseCase;

    @Override
    public void process(List<Notification> notifications, String eventId) {
        log.info("Routing {} notification(s) for eventId: {}", notifications.size(), eventId);

        for (Notification notification : notifications) {
            try {
                sendNotificationUseCase.routeAndSend(notification);
                log.info("Notification {} routed successfully for user: {}",
                        notification.getId(), notification.getUserId());
            } catch (DuplicateEventException e) {
                // Idempotency Guard – ghi nhận cảnh báo và bỏ qua an toàn
                log.warn("Idempotency Alert: Duplicate event detected and ignored. " +
                        "EventId: {}, UserId: {}", eventId, notification.getUserId());
            } catch (Exception e) {
                // Lỗi hệ thống trên từng notification không được lan ra để không block các notification còn lại
                log.error("Unexpected error routing notification {}. EventId: {}",
                        notification.getId(), eventId, e);
            }
        }
    }
}
