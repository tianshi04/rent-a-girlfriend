package com.rentagf.notification.interfaces.event;

import static org.junit.jupiter.api.Assertions.*;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import io.cloudevents.CloudEvent;
import java.net.URI;
import java.time.OffsetDateTime;
import java.util.Map;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

@Tag("unit")
public class CloudEventsParserTest {

  private final CloudEventsParser parser = new CloudEventsParser();
  private final ObjectMapper objectMapper = new ObjectMapper();

  @Test
  public void testParseValidCloudEventSuccessfully() throws Exception {
    String json =
        """
            {
              "specversion": "1.0",
              "type": "booking.booking-requested.v1",
              "source": "/services/booking",
              "id": "a5d89f81-81f1-4db5-9e67-d86161726a45",
              "time": "2026-05-23T10:00:00Z",
              "datacontenttype": "application/json",
              "correlationid": "corr-test-123",
              "data": {
                "bookingId": "booking-123",
                "clientId": "c1111111-1111-1111-1111-111111111111",
                "companionId": "d2222222-2222-2222-2222-222222222222"
              }
            }
            """;

    CloudEvent event = parser.parse(json);

    assertNotNull(event);
    assertEquals("1.0", event.getSpecVersion().toString());
    assertEquals("booking.booking-requested.v1", event.getType());
    assertEquals(URI.create("/services/booking"), event.getSource());
    assertEquals("a5d89f81-81f1-4db5-9e67-d86161726a45", event.getId());
    assertEquals(OffsetDateTime.parse("2026-05-23T10:00:00Z"), event.getTime());
    assertEquals("application/json", event.getDataContentType());
    assertEquals("corr-test-123", event.getExtension("correlationid"));

    assertNotNull(event.getData());
    Map<String, Object> data =
        objectMapper.readValue(
            event.getData().toBytes(), new TypeReference<Map<String, Object>>() {});
    assertNotNull(data);
    assertEquals("booking-123", data.get("bookingId"));
    assertEquals("c1111111-1111-1111-1111-111111111111", data.get("clientId"));
    assertEquals("d2222222-2222-2222-2222-222222222222", data.get("companionId"));
  }

  @Test
  public void testParseInvalidJsonThrowsIllegalArgumentException() {
    String invalidJson = "{ invalid json }";
    assertThrows(IllegalArgumentException.class, () -> parser.parse(invalidJson));
  }

  @Test
  public void testParseMissingRequiredFieldsThrowsIllegalArgumentException() {
    String missingFieldsJson =
        """
            {
              "specversion": "1.0",
              "type": "booking.booking-requested.v1"
            }
            """;
    assertThrows(IllegalArgumentException.class, () -> parser.parse(missingFieldsJson));
  }
}
