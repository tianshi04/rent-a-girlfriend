package com.rentagf.notification.interfaces.event;

import lombok.AllArgsConstructor;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;
import java.time.Instant;
import java.util.Map;

/**
 * Cấu trúc đối tượng đóng gói CloudEvent v1.0 Envelope.
 */
@Getter
@Setter
@NoArgsConstructor
@AllArgsConstructor
public class CloudEvent {
    private String specversion;
    private String type;
    private String source;
    private String id;
    private Instant time;
    private String datacontenttype;
    private Map<String, Object> data;
}
