package com.rentagf.notification.interfaces.event;

import com.rentagf.notification.application.port.inbound.ProcessInboundEventUseCase;
import com.rentagf.notification.domain.aggregate.Notification;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.kafka.support.Acknowledgment;
import org.springframework.stereotype.Component;

import java.util.List;

/**
 * Inbound Adapter tiêu thụ (consume) sự kiện từ Kafka Broker.
 *
 * <p>Trách nhiệm của lớp này bao gồm:
 * <ol>
 *   <li>Nhận raw JSON message từ Kafka.</li>
 *   <li>Parse JSON thô → {@link CloudEvent} (Fail-Fast nếu sai cấu trúc).</li>
 *   <li>Translate {@link CloudEvent} → {@code List<Notification>} (Fail-Fast nếu thiếu template biến).</li>
 *   <li>Uỷ thác routing cho {@link ProcessInboundEventUseCase} (Application layer).</li>
 *   <li>Luôn commit offset (Acknowledge) ở finally để bảo vệ hệ thống không bị nghẽn queue.</li>
 * </ol>
 *
 * <p>Thiết kế này giữ Application Layer hoàn toàn sạch (không biết về Kafka, JSON hay CloudEvent format).
 */
@Slf4j
@Component
@RequiredArgsConstructor
public class KafkaEventConsumer {

    private final ProcessInboundEventUseCase inboundEventUseCase;
    private final CloudEventsParser cloudEventsParser;
    private final EventTranslator eventTranslator;

    /**
     * Lắng nghe các Topic sự kiện từ Kafka.
     * Parse + translate ở tầng Adapter rồi uỷ thác routing cho Inbound Port.
     *
     * @param message        JSON String thô nhận từ Kafka.
     * @param acknowledgment Công cụ commit offset thủ công.
     */
    @KafkaListener(
            topics = {
                "booking-events",
                "finance-events",
                "interaction-events",
                "profile-events",
                "identity-events",
                "dispute-events"
            },
            groupId = "notification-service"
    )
    public void consume(String message, Acknowledgment acknowledgment) {
        log.info("Received raw event message from Kafka: {}", message);

        String eventId = "UNKNOWN";
        try {
            // 1. Parse JSON thô thành CloudEvent (Fail-Fast nếu sai cấu trúc)
            CloudEvent cloudEvent = cloudEventsParser.parse(message);
            eventId = cloudEvent.getId();
            log.info("Successfully parsed CloudEvent. Id: {}, Type: {}", eventId, cloudEvent.getType());

            // 2. Dịch CloudEvent thành danh sách Notification domain (Fail-Fast nếu thiếu template biến)
            List<Notification> notifications = eventTranslator.translate(cloudEvent);
            log.info("Event {} translated into {} notification(s).", eventId, notifications.size());

            // 3. Uỷ thác hoàn toàn routing cho Application Layer qua Inbound Port
            inboundEventUseCase.process(notifications, eventId);

        } catch (IllegalArgumentException e) {
            // Lỗi nghiệp vụ unrecoverable (sai schema, thiếu biến template...)
            log.error("Fail-Fast Alert: Invalid event payload. EventId: {}, Error: {}",
                    eventId, e.getMessage(), e);
        } catch (Exception e) {
            // Lỗi hệ thống ngoài dự kiến
            log.error("Unexpected system error consuming event. EventId: {}", eventId, e);
        } finally {
            // Luôn commit offset để tránh lặp vô tận tin nhắn lỗi
            if (acknowledgment != null) {
                acknowledgment.acknowledge();
            }
        }
    }
}
