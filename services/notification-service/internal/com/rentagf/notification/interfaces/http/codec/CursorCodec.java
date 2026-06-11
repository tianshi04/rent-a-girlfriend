package com.rentagf.notification.interfaces.http.codec;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import com.rentagf.notification.domain.errors.InvalidCursorException;
import com.rentagf.notification.domain.vo.InboxCursor;
import java.nio.charset.StandardCharsets;
import java.util.Base64;

/**
 * Component mã hóa và giải mã InboxCursor. Đặt tại interfaces layer để tránh làm rò rỉ thư viện
 * Jackson (ObjectMapper) vào Domain core.
 */
public final class CursorCodec {

  private static final ObjectMapper OBJECT_MAPPER =
      new ObjectMapper().registerModule(new JavaTimeModule());

  private CursorCodec() {
    // Utility class
  }

  /** Mã hóa InboxCursor thành chuỗi Base64 URL-safe. */
  public static String encode(InboxCursor cursor) {
    if (cursor == null) {
      return null;
    }
    try {
      String json = OBJECT_MAPPER.writeValueAsString(cursor);
      return Base64.getUrlEncoder()
          .withoutPadding()
          .encodeToString(json.getBytes(StandardCharsets.UTF_8));
    } catch (Exception e) {
      throw new InvalidCursorException("Failed to encode cursor", e);
    }
  }

  /** Giải mã chuỗi Base64 URL-safe thành đối tượng InboxCursor. */
  public static InboxCursor decode(String rawCursor) {
    if (rawCursor == null || rawCursor.trim().isEmpty()) {
      return null;
    }
    try {
      byte[] decodedBytes = Base64.getUrlDecoder().decode(rawCursor.trim());
      String json = new String(decodedBytes, StandardCharsets.UTF_8);
      return OBJECT_MAPPER.readValue(json, InboxCursor.class);
    } catch (IllegalArgumentException e) {
      throw new InvalidCursorException("Invalid Base64 format for cursor", e);
    } catch (Exception e) {
      throw new InvalidCursorException("Failed to parse cursor JSON", e);
    }
  }
}
