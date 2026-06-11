package com.rentagf.notification.application.port.inbound;

import java.util.UUID;

/** Inbound Port (UseCase) đại diện cho nghiệp vụ đăng ký nhận thông báo thời gian thực. */
public interface NotificationSubscriptionUseCase {
  /**
   * Đăng ký nhận thông báo thời gian thực cho User.
   *
   * @param userId UUID của User nhận tin
   */
  void subscribe(UUID userId);
}
