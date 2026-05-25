package com.rentagf.notification.interfaces.event;

import org.yaml.snakeyaml.Yaml;
import java.io.FileInputStream;
import java.io.InputStream;
import java.util.*;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * Engine chịu trách nhiệm parse templates.yaml và biên dịch (interpolate) các tin nhắn tiếng Việt.
 * Áp dụng triệt để nguyên lý Fail-Fast: Ném Exception ngay khi thiếu tham số cần thiết trong payload.
 */
public class TemplateEngine {

    private Map<String, Object> eventsConfig = new HashMap<>();
    private static final Pattern PLACEHOLDER_PATTERN = Pattern.compile("\\{\\{([^}]+)\\}\\}");

    public TemplateEngine(InputStream inputStream) {
        try {
            Yaml yaml = new Yaml();
            Map<String, Object> root = yaml.load(inputStream);
            if (root != null && root.containsKey("events")) {
                this.eventsConfig = (Map<String, Object>) root.get("events");
            }
        } catch (Exception e) {
            throw new IllegalStateException("Failed to load templates from InputStream", e);
        }
    }

    @SuppressWarnings("unchecked")
    public TemplateEngine(String configFilePath) {
        try (InputStream input = new FileInputStream(configFilePath)) {
            Yaml yaml = new Yaml();
            Map<String, Object> root = yaml.load(input);
            if (root != null && root.containsKey("events")) {
                this.eventsConfig = (Map<String, Object>) root.get("events");
            }
        } catch (Exception e) {
            throw new IllegalStateException("Failed to load templates.yaml from " + configFilePath, e);
        }
    }

    /**
     * Lấy cấu hình thô của một event type.
     */
    @SuppressWarnings("unchecked")
    public Map<String, Object> getEventConfig(String eventType) {
        if (!eventsConfig.containsKey(eventType)) {
            throw new IllegalArgumentException("Unknown event type in templates: " + eventType);
        }
        return (Map<String, Object>) eventsConfig.get(eventType);
    }

    public String getRecipientField(String eventType) {
        Map<String, Object> config = getEventConfig(eventType);
        return (String) config.get("recipient_field");
    }

    public String getPriority(String eventType) {
        Map<String, Object> config = getEventConfig(eventType);
        return (String) config.get("priority");
    }

    @SuppressWarnings("unchecked")
    public List<String> getChannels(String eventType) {
        Map<String, Object> config = getEventConfig(eventType);
        return (List<String>) config.get("channels");
    }

    /**
     * Dịch và ráp tham số cho tiêu đề hoặc nội dung thông báo.
     * Mặc định Phase 3 chỉ hỗ trợ tiếng Việt ("vi").
     *
     * @param eventType Loại sự kiện.
     * @param data      Payload của sự kiện chứa các tham số động.
     * @param isTitle   True nếu dịch tiêu đề, False nếu dịch body.
     * @return Chuỗi nội dung đã được ráp tham số đầy đủ.
     */
    @SuppressWarnings("unchecked")
    public String render(String eventType, Map<String, Object> data, boolean isTitle) {
        Map<String, Object> config = getEventConfig(eventType);
        Map<String, Object> template = (Map<String, Object>) config.get("template");
        if (template == null || !template.containsKey("vi")) {
            throw new IllegalStateException("Missing Vietnamese 'vi' template for event: " + eventType);
        }

        Map<String, Object> viTemplate = (Map<String, Object>) template.get("vi");
        String rawText = (String) viTemplate.get(isTitle ? "title" : "body");
        if (rawText == null) {
            throw new IllegalStateException("Missing template text for event: " + eventType + " (isTitle: " + isTitle + ")");
        }

        return interpolate(rawText, data, eventType);
    }

    /**
     * Thực hiện thay thế các placeholder {{var}} bằng giá trị thực tế trong data.
     * Áp dụng FAIL-FAST: Nếu thiếu biến, ném IllegalArgumentException lập tức.
     */
    private String interpolate(String text, Map<String, Object> data, String eventType) {
        Matcher matcher = PLACEHOLDER_PATTERN.matcher(text);
        StringBuilder result = new StringBuilder();

        while (matcher.find()) {
            String variableName = matcher.group(1).trim();
            Object value = data.get(variableName);

            if (value == null) {
                // FAIL-FAST: Ném exception lập tức để bộ phận Ops phát hiện lỗi schema
                throw new IllegalArgumentException(String.format(
                        "Missing required template variable '%s' in event payload for event type '%s'",
                        variableName, eventType
                ));
            }

            matcher.appendReplacement(result, Matcher.quoteReplacement(value.toString()));
        }
        matcher.appendTail(result);
        return result.toString();
    }
}
