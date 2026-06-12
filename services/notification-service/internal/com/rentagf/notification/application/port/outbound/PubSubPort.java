package com.rentagf.notification.application.port.outbound;

/**
 * Outbound Port cho việc truyền thông điệp liên-máy (Inter-Pod Communication) sử dụng mô hình
 * Publish/Subscribe.
 */
public interface PubSubPort {

  /**
   * Publish một tin nhắn tới kênh cụ thể.
   *
   * @param channel Tên kênh cần gửi
   * @param message Nội dung tin nhắn (JSON payload)
   */
  void publish(String channel, String message);

  /**
   * Subscribe động vào một kênh cụ thể để lắng nghe tin nhắn.
   *
   * @param channel Tên kênh cần lắng nghe
   */
  void subscribe(String channel);

  /**
   * Unsubscribe khỏi một kênh cụ thể để giải phóng tài nguyên.
   *
   * @param channel Tên kênh cần hủy lắng nghe
   */
  void unsubscribe(String channel);
}
