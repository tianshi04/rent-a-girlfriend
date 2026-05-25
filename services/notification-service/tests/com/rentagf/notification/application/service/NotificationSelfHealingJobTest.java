package com.rentagf.notification.application.service;

import com.rentagf.notification.application.event.NotificationReadyEvent;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.repository.NotificationRepository;
import com.rentagf.notification.domain.vo.enums.DeliveryChannel;
import com.rentagf.notification.domain.vo.enums.NotificationPriority;
import com.rentagf.notification.domain.vo.enums.NotificationStatus;
import com.rentagf.notification.domain.vo.enums.NotificationType;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;
import org.springframework.context.ApplicationEventPublisher;

import java.time.Instant;
import java.util.List;
import java.util.Map;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.mockito.Mockito.*;

class NotificationSelfHealingJobTest {

    private NotificationRepository repository;
    private ApplicationEventPublisher eventPublisher;
    private NotificationSelfHealingJob selfHealingJob;

    @BeforeEach
    void setUp() {
        repository = mock(NotificationRepository.class);
        eventPublisher = mock(ApplicationEventPublisher.class);
        selfHealingJob = new NotificationSelfHealingJob(repository, eventPublisher);
    }

    @Test
    void testPeriodicRecovery_NoStuckNotifications_NoEventFired() {
        // Given
        when(repository.findAllByStatusAndCreatedAtBefore(eq(NotificationStatus.PROCESSING), any(Instant.class)))
                .thenReturn(List.of());

        // When
        selfHealingJob.scheduleRecovery();

        // Then
        verify(repository, times(1)).findAllByStatusAndCreatedAtBefore(eq(NotificationStatus.PROCESSING), any(Instant.class));
        verifyNoInteractions(eventPublisher);
    }

    @Test
    void testPeriodicRecovery_StuckNotificationWithNoAttempts_FallbackToPolicyChannel() {
        // Given
        UUID userId = UUID.randomUUID();
        // Cấu hình policy có FCM
        Notification stuckNotification = Notification.create(
                userId, "evt-stuck-1", NotificationType.TRANSACTIONAL,
                NotificationPriority.HIGH, Map.of("title", "Stuck"), Map.of("channels", List.of("EMAIL", "FCM"))
        );

        when(repository.findAllByStatusAndCreatedAtBefore(eq(NotificationStatus.PROCESSING), any(Instant.class)))
                .thenReturn(List.of(stuckNotification));

        // When
        selfHealingJob.scheduleRecovery();

        // Then
        verify(repository, times(1)).findAllByStatusAndCreatedAtBefore(eq(NotificationStatus.PROCESSING), any(Instant.class));

        // EMAIL là kênh đầu tiên cấu hình -> phải fallback lấy EMAIL
        ArgumentCaptor<NotificationReadyEvent> eventCaptor = ArgumentCaptor.forClass(NotificationReadyEvent.class);
        verify(eventPublisher, times(1)).publishEvent(eventCaptor.capture());

        NotificationReadyEvent firedEvent = eventCaptor.getValue();
        assertEquals(stuckNotification.getId(), firedEvent.getNotificationId());
        assertEquals(DeliveryChannel.EMAIL, firedEvent.getChannel());
    }

    @Test
    void testPeriodicRecovery_StuckNotificationWithLastAttempt_ReDispatchesViaLastChannel() {
        // Given
        UUID userId = UUID.randomUUID();
        Notification stuckNotification = Notification.create(
                userId, "evt-stuck-2", NotificationType.TRANSACTIONAL,
                NotificationPriority.HIGH, Map.of("title", "Stuck"), Map.of("channels", List.of("SSE", "FCM"))
        );

        // Tạo nỗ lực gửi trước đó bằng FCM
        stuckNotification.createAttempt(DeliveryChannel.FCM);

        when(repository.findAllByStatusAndCreatedAtBefore(eq(NotificationStatus.PROCESSING), any(Instant.class)))
                .thenReturn(List.of(stuckNotification));

        // When
        selfHealingJob.scheduleRecovery();

        // Then
        verify(repository, times(1)).findAllByStatusAndCreatedAtBefore(eq(NotificationStatus.PROCESSING), any(Instant.class));

        // Lấy attempt cuối cùng -> FCM
        ArgumentCaptor<NotificationReadyEvent> eventCaptor = ArgumentCaptor.forClass(NotificationReadyEvent.class);
        verify(eventPublisher, times(1)).publishEvent(eventCaptor.capture());

        NotificationReadyEvent firedEvent = eventCaptor.getValue();
        assertEquals(stuckNotification.getId(), firedEvent.getNotificationId());
        assertEquals(DeliveryChannel.FCM, firedEvent.getChannel());
    }

    @Test
    void testStartupRecovery_InvokesExecutionCorrectly() {
        // Given
        when(repository.findAllByStatusAndCreatedAtBefore(eq(NotificationStatus.PROCESSING), any(Instant.class)))
                .thenReturn(List.of());

        // When
        selfHealingJob.onApplicationReady();

        // Then
        verify(repository, times(1)).findAllByStatusAndCreatedAtBefore(eq(NotificationStatus.PROCESSING), any(Instant.class));
        verifyNoInteractions(eventPublisher);
    }
}
