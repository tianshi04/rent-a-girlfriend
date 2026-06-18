package com.rentagf.notification.interfaces.event;

import static org.junit.jupiter.api.Assertions.*;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.vo.enums.NotificationPriority;
import com.rentagf.notification.domain.vo.enums.NotificationType;
import com.rentagf.notification.interfaces.event.resolver.BookingCancelledResolver;
import com.rentagf.notification.interfaces.event.resolver.DisputeResolvedResolver;
import com.rentagf.notification.interfaces.event.resolver.SimpleRecipientResolver;
import io.cloudevents.CloudEvent;
import io.cloudevents.core.builder.CloudEventBuilder;
import java.io.File;
import java.net.URI;
import java.time.OffsetDateTime;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

@Tag("unit")
public class EventTranslatorTest {

  private EventTranslator translator;
  private ObjectMapper objectMapper;

  @BeforeEach
  public void setUp() throws Exception {
    // Tải templates.yaml thật từ thư mục config của dự án để chạy test chính xác
    File templateFile = new File("config/templates.yaml");
    if (!templateFile.exists()) {
      // Fallback nếu chạy test từ thư mục con hoặc IDE cấu hình working dir khác
      templateFile = new File("services/notification-service/config/templates.yaml");
    }

    assertTrue(templateFile.exists(), "templates.yaml file must exist for testing");

    objectMapper = new ObjectMapper();
    SimpleRecipientResolver simpleFallbackResolver = new SimpleRecipientResolver();

    List<com.rentagf.notification.interfaces.event.resolver.RecipientResolver> resolverList =
        List.of(
            new BookingCancelledResolver(objectMapper), new DisputeResolvedResolver(objectMapper));

    RecipientResolverRegistry recipientResolverRegistry =
        new RecipientResolverRegistry(resolverList, simpleFallbackResolver);
    TemplateEngine templateEngine = new TemplateEngine(templateFile.getAbsolutePath());

    translator = new EventTranslator(templateEngine, recipientResolverRegistry, objectMapper);
  }

  private CloudEvent createDummyEvent(
      String type, String source, String id, Map<String, Object> data) throws Exception {
    byte[] dataBytes = objectMapper.writeValueAsBytes(data);
    return CloudEventBuilder.v1()
        .withId(id)
        .withType(type)
        .withSource(URI.create(source))
        .withTime(OffsetDateTime.now())
        .withDataContentType("application/json")
        .withData(dataBytes)
        .build();
  }

  @Test
  public void testTranslateKanoCoinDepositedSuccessfully() throws Exception {
    UUID userId = UUID.randomUUID();
    String eventId = UUID.randomUUID().toString();

    CloudEvent event =
        createDummyEvent(
            "finance.kano-coin-deposited.v1",
            "/services/finance",
            eventId,
            Map.of("userId", userId.toString(), "amount", 500, "transactionId", "tx-999"));

    List<Notification> notifications = translator.translate(event);

    assertEquals(1, notifications.size());
    Notification notification = notifications.getFirst();

    assertEquals(userId, notification.getUserId());
    assertEquals(eventId, notification.getEventId());
    assertEquals(NotificationType.TRANSACTIONAL, notification.getType());
    assertEquals(NotificationPriority.MEDIUM, notification.getPriority());

    // Kiểm tra placeholder interpolation tiếng Việt
    Map<String, Object> payload = notification.getPayload();
    assertEquals("Nạp tiền thành công 💰", payload.get("title"));
    assertEquals("Bạn vừa nạp thành công 500 Kano-Coin.", payload.get("body"));
  }

  @Test
  public void testTranslateBookingCancelledByClientResolvesDynamicRecipient() throws Exception {
    UUID clientId = UUID.randomUUID();
    UUID companionId = UUID.randomUUID();
    String eventId = UUID.randomUUID().toString();

    CloudEvent event =
        createDummyEvent(
            "booking.booking-cancelled.v1",
            "/services/booking",
            eventId,
            Map.of(
                "bookingId",
                "b-888",
                "clientId",
                clientId.toString(),
                "companionId",
                companionId.toString(),
                "actorRole",
                "CLIENT"));

    // Recipient của BookingCancelled phải là đối phương -> actorRole: CLIENT thì người nhận là
    // companionId
    List<Notification> notifications = translator.translate(event);

    assertEquals(1, notifications.size());
    Notification notification = notifications.getFirst();

    assertEquals(companionId, notification.getUserId()); // Companion nhận tin
    assertEquals("Lịch hẹn bị hủy ⚠️", notification.getPayload().get("title"));
    assertEquals("Lịch hẹn #b-888 đã bị hủy bởi CLIENT.", notification.getPayload().get("body"));
  }

  @Test
  public void testTranslateDisputeResolvedSendsToBothParties() throws Exception {
    UUID reporterId = UUID.randomUUID();
    UUID accusedId = UUID.randomUUID();
    String eventId = UUID.randomUUID().toString();

    CloudEvent event =
        createDummyEvent(
            "dispute.dispute-resolved.v1",
            "/services/dispute",
            eventId,
            Map.of(
                "disputeId", "d-777",
                "bookingId", "b-555",
                "reporterId", reporterId.toString(),
                "accusedId", accusedId.toString(),
                "resolution", "Hoàn trả 50% Kano-Coin"));

    // DisputeResolved phải sinh ra 2 notifications riêng biệt cho cả hai bên
    List<Notification> notifications = translator.translate(event);

    assertEquals(2, notifications.size());

    boolean hasReporter = false;
    boolean hasAccused = false;

    for (Notification notification : notifications) {
      assertEquals(eventId, notification.getEventId());
      assertEquals("Kết quả khiếu nại 📋", notification.getPayload().get("title"));
      assertEquals(
          "Khiếu nại cho booking đã được giải quyết: Hoàn trả 50% Kano-Coin.",
          notification.getPayload().get("body"));

      if (notification.getUserId().equals(reporterId)) {
        hasReporter = true;
      } else if (notification.getUserId().equals(accusedId)) {
        hasAccused = true;
      }
    }

    assertTrue(hasReporter, "Should send notification to Reporter");
    assertTrue(hasAccused, "Should send notification to Accused");
  }

  @Test
  public void testTranslateUnknownEventTypeThrowsIllegalArgumentException() throws Exception {
    CloudEvent unknownEvent =
        createDummyEvent(
            "unknown.event.v1",
            "/services/unknown",
            UUID.randomUUID().toString(),
            Map.of("userId", UUID.randomUUID().toString()));

    assertThrows(IllegalArgumentException.class, () -> translator.translate(unknownEvent));
  }
}
