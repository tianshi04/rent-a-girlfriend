package com.rentagf.notification.interfaces.event;

import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import java.time.Instant;
import java.util.Map;
import static org.junit.jupiter.api.Assertions.*;

@Tag("unit")
public class CloudEventsParserTest {

    private final CloudEventsParser parser = new CloudEventsParser();

    @Test
    public void testParseValidCloudEventSuccessfully() {
        String json = """
            {
              "specversion": "1.0",
              "type": "com.rentagf.booking.BookingRequested.v1",
              "source": "/services/booking",
              "id": "a5d89f81-81f1-4db5-9e67-d86161726a45",
              "time": "2026-05-23T10:00:00Z",
              "datacontenttype": "application/json",
              "data": {
                "bookingId": "booking-123",
                "clientId": "c1111111-1111-1111-1111-111111111111",
                "companionId": "d2222222-2222-2222-2222-222222222222"
              }
            }
            """;

        CloudEvent event = parser.parse(json);

        assertNotNull(event);
        assertEquals("1.0", event.getSpecversion());
        assertEquals("com.rentagf.booking.BookingRequested.v1", event.getType());
        assertEquals("/services/booking", event.getSource());
        assertEquals("a5d89f81-81f1-4db5-9e67-d86161726a45", event.getId());
        assertEquals(Instant.parse("2026-05-23T10:00:00Z"), event.getTime());
        assertEquals("application/json", event.getDatacontenttype());
        
        Map<String, Object> data = event.getData();
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
        String missingFieldsJson = """
            {
              "specversion": "1.0",
              "type": "com.rentagf.booking.BookingRequested.v1"
            }
            """;
        assertThrows(IllegalArgumentException.class, () -> parser.parse(missingFieldsJson));
    }
}
