package com.rentagf.notification.application.port.outbound;

import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.vo.enums.DeliveryChannel;

/**
 * Strategy interface cho việc gửi tin ngoại vi. Tất cả các provider adapters ở tầng infrastructure
 * (Email, FCM, SSE) bắt buộc implement.
 */
public interface NotificationSender {

  /**
   * Thực thi gửi thông báo ngoại vi.
   *
   * @param notification Aggregate root chứa đầy đủ thông tin gửi
   * @return SendResult giàu ngữ cảnh kết quả gửi
   */
  SendResult send(Notification notification);

  /** Trả về kênh phân phối tương ứng của Strategy. */
  DeliveryChannel getChannel();
}
