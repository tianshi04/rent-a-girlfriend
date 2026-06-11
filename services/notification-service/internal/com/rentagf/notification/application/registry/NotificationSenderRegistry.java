package com.rentagf.notification.application.registry;

import com.rentagf.notification.application.port.outbound.EmailPort;
import com.rentagf.notification.application.port.outbound.FcmPort;
import com.rentagf.notification.application.port.outbound.NotificationSender;
import com.rentagf.notification.application.port.outbound.SsePort;
import com.rentagf.notification.domain.vo.enums.DeliveryChannel;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.function.Function;
import java.util.stream.Collectors;
import org.springframework.stereotype.Component;

/**
 * Registry quản lý các NotificationSender Strategy. Tự động thu thập tất cả các bean implement
 * NotificationSender.
 */
@Component
public class NotificationSenderRegistry {

  private final Map<DeliveryChannel, NotificationSender> senders;

  public NotificationSenderRegistry(List<NotificationSender> senderList) {
    this.senders =
        senderList.stream()
            .collect(
                Collectors.toMap(
                    sender -> {
                      DeliveryChannel channel = sender.getChannel();
                      if (channel != null) {
                        return channel;
                      }
                      // Fallback cho Mockito mock beans trước khi được stub trong môi trường test
                      if (sender instanceof EmailPort) {
                        return DeliveryChannel.EMAIL;
                      }
                      if (sender instanceof FcmPort) {
                        return DeliveryChannel.FCM;
                      }
                      if (sender instanceof SsePort) {
                        return DeliveryChannel.SSE;
                      }
                      throw new IllegalArgumentException("Unknown sender channel");
                    },
                    Function.identity()));
  }

  /**
   * Lấy Strategy tương ứng cho kênh gửi.
   *
   * @param channel Kênh gửi (SSE, FCM, EMAIL)
   * @return Optional chứa NotificationSender tương ứng
   */
  public Optional<NotificationSender> getSender(DeliveryChannel channel) {
    return Optional.ofNullable(senders.get(channel));
  }
}
