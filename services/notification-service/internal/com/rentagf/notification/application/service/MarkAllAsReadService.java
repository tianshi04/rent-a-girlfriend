package com.rentagf.notification.application.service;

import com.rentagf.notification.application.port.inbound.MarkAllAsReadUseCase;
import com.rentagf.notification.domain.repository.NotificationRepository;
import java.time.Instant;
import java.util.UUID;
import lombok.RequiredArgsConstructor;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

/** Service triển khai Use Case MarkAllAsReadUseCase. */
@Service
@RequiredArgsConstructor
@Transactional
public class MarkAllAsReadService implements MarkAllAsReadUseCase {

  private final NotificationRepository notificationRepository;

  @Override
  public int markAllAsRead(UUID userId) {
    if (userId == null) {
      throw new IllegalArgumentException("userId cannot be null");
    }
    return notificationRepository.markAllAsRead(userId, Instant.now());
  }
}
