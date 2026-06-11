package com.rentagf.notification.application.port.outbound;

import lombok.Builder;
import lombok.Getter;

/**
 * Kết quả gửi tin ngoại vi phong phú. Chứa đầy đủ ngữ cảnh để domain logic quyết định retry hay báo
 * hỏng.
 */
@Getter
@Builder
public class SendResult {
  private final boolean success;
  private final String messageId; // ID tin nhắn từ nhà cung cấp (FCM, SendGrid...) để đối soát
  private final String errorCode; // Mã lỗi hệ thống (ví dụ: FCM_TOKEN_INVALID, SMTP_TIMEOUT)
  private final String errorMessage; // Chi tiết thông báo lỗi phục vụ debug
  private final boolean recoverable; // Lỗi tạm thời có thể cứu vãn (true) hay lỗi vĩnh viễn (false)

  public static SendResult success(String messageId) {
    return SendResult.builder().success(true).messageId(messageId).recoverable(false).build();
  }

  public static SendResult fail(String errorCode, String errorMessage, boolean recoverable) {
    return SendResult.builder()
        .success(false)
        .errorCode(errorCode)
        .errorMessage(errorMessage)
        .recoverable(recoverable)
        .build();
  }
}
