package com.rentagf.notification.interfaces.event;

import io.cloudevents.CloudEvent;
import io.cloudevents.core.format.EventFormat;
import io.cloudevents.core.provider.EventFormatProvider;
import io.cloudevents.jackson.JsonFormat;
import java.nio.charset.StandardCharsets;
import org.springframework.stereotype.Component;

/** Tiện ích phân tích cú pháp chuỗi JSON thành đối tượng CloudEvent v1.0. */
@Component
public class CloudEventsParser {

  private final EventFormat eventFormat;

  public CloudEventsParser() {
    this.eventFormat = EventFormatProvider.getInstance().resolveFormat(JsonFormat.CONTENT_TYPE);
  }

  /**
   * Parse JSON string thành đối tượng CloudEvent. Ném IllegalArgumentException nếu JSON lỗi hoặc
   * thiếu các trường bắt buộc.
   */
  public CloudEvent parse(String jsonString) {
    if (jsonString == null || jsonString.trim().isEmpty()) {
      throw new IllegalArgumentException("JSON string must not be empty");
    }

    try {
      byte[] bytes = jsonString.getBytes(StandardCharsets.UTF_8);
      CloudEvent event = eventFormat.deserialize(bytes);
      validate(event);
      return event;
    } catch (Exception e) {
      throw new IllegalArgumentException("Failed to parse CloudEvent JSON: " + e.getMessage(), e);
    }
  }

  private void validate(CloudEvent event) {
    if (event.getSpecVersion() == null || event.getSpecVersion().toString().trim().isEmpty()) {
      throw new IllegalArgumentException("Missing required CloudEvent field: specversion");
    }
    if (event.getType() == null || event.getType().trim().isEmpty()) {
      throw new IllegalArgumentException("Missing required CloudEvent field: type");
    }
    if (event.getSource() == null || event.getSource().toString().trim().isEmpty()) {
      throw new IllegalArgumentException("Missing required CloudEvent field: source");
    }
    if (event.getId() == null || event.getId().trim().isEmpty()) {
      throw new IllegalArgumentException("Missing required CloudEvent field: id");
    }
    if (event.getData() == null) {
      throw new IllegalArgumentException("Missing required CloudEvent field: data");
    }
  }
}
