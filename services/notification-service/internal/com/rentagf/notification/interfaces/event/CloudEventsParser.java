package com.rentagf.notification.interfaces.event;

import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.springframework.stereotype.Component;

/**
 * Tiện ích phân tích cú pháp chuỗi JSON thành đối tượng CloudEvent v1.0.
 */
@Component
public class CloudEventsParser {

    private final ObjectMapper objectMapper;

    public CloudEventsParser() {
        this.objectMapper = new ObjectMapper()
                .registerModule(new JavaTimeModule())
                .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
    }

    public CloudEventsParser(ObjectMapper objectMapper) {
        this.objectMapper = objectMapper;
    }

    /**
     * Parse JSON string thành đối tượng CloudEvent.
     * Ném IllegalArgumentException nếu JSON lỗi hoặc thiếu các trường bắt buộc.
     */
    public CloudEvent parse(String jsonString) {
        if (jsonString == null || jsonString.trim().isEmpty()) {
            throw new IllegalArgumentException("JSON string must not be empty");
        }

        try {
            CloudEvent event = objectMapper.readValue(jsonString, CloudEvent.class);
            validate(event);
            return event;
        } catch (Exception e) {
            throw new IllegalArgumentException("Failed to parse CloudEvent JSON: " + e.getMessage(), e);
        }
    }

    private void validate(CloudEvent event) {
        if (event.getSpecversion() == null || event.getSpecversion().trim().isEmpty()) {
            throw new IllegalArgumentException("Missing required CloudEvent field: specversion");
        }
        if (event.getType() == null || event.getType().trim().isEmpty()) {
            throw new IllegalArgumentException("Missing required CloudEvent field: type");
        }
        if (event.getSource() == null || event.getSource().trim().isEmpty()) {
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
