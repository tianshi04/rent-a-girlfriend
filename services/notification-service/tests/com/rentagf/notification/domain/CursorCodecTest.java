package com.rentagf.notification.domain;

import com.rentagf.notification.domain.errors.InvalidCursorException;
import com.rentagf.notification.interfaces.http.codec.CursorCodec;
import com.rentagf.notification.domain.vo.InboxCursor;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

import java.time.Instant;
import java.util.Base64;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;

@Tag("unit")
class CursorCodecTest {

    @Test
    void testEncodeAndDecodeSuccessfully() {
        // Arrange
        Instant createdAt = Instant.parse("2026-05-24T10:00:00Z");
        UUID id = UUID.randomUUID();
        InboxCursor originalCursor = new InboxCursor(createdAt, id);

        // Act
        String encoded = CursorCodec.encode(originalCursor);
        assertNotNull(encoded);
        assertFalse(encoded.contains("=")); // URL-safe without padding should not contain '='

        InboxCursor decodedCursor = CursorCodec.decode(encoded);

        // Assert
        assertNotNull(decodedCursor);
        assertEquals(originalCursor.createdAt(), decodedCursor.createdAt());
        assertEquals(originalCursor.id(), decodedCursor.id());
    }

    @Test
    void testDecodeNullOrEmptyReturnsNull() {
        assertNull(CursorCodec.decode(null));
        assertNull(CursorCodec.decode(""));
        assertNull(CursorCodec.decode("   "));
    }

    @Test
    void testDecodeInvalidBase64ThrowsInvalidCursorException() {
        assertThrows(InvalidCursorException.class, () -> CursorCodec.decode("not-a-base64-string!@#"));
    }

    @Test
    void testDecodeInvalidJsonThrowsInvalidCursorException() {
        // Encode a random string that is valid base64 but not valid JSON
        String rawString = "Hello, World!";
        String encoded = Base64.getUrlEncoder().withoutPadding().encodeToString(rawString.getBytes());

        assertThrows(InvalidCursorException.class, () -> CursorCodec.decode(encoded));
    }

    @Test
    void testDecodeMissingFieldsThrowsInvalidCursorException() {
        // Encode JSON missing required fields
        String invalidJson = "{\"someOtherField\":\"value\"}";
        String encoded = Base64.getUrlEncoder().withoutPadding().encodeToString(invalidJson.getBytes());

        assertThrows(InvalidCursorException.class, () -> CursorCodec.decode(encoded));
    }
}
