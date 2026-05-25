package com.rentagf.notification.infrastructure.adapter;

import com.rentagf.notification.application.event.NotificationReadyEvent;
import com.rentagf.notification.domain.vo.enums.DeliveryChannel;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;
import org.mockito.Mockito;
import org.springframework.context.ApplicationEventPublisher;

import java.time.Duration;
import java.util.UUID;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

class InMemoryRetrySchedulerTest {

    private ApplicationEventPublisher eventPublisher;
    private InMemoryRetryScheduler retryScheduler;

    @BeforeEach
    void setUp() {
        eventPublisher = mock(ApplicationEventPublisher.class);
        retryScheduler = new InMemoryRetryScheduler(eventPublisher);
    }

    @Test
    void testScheduleRetry_FiresEventAfterDelay() throws InterruptedException {
        // Given
        UUID notificationId = UUID.randomUUID();
        DeliveryChannel channel = DeliveryChannel.FCM;
        Duration delay = Duration.ofMillis(100);

        CountDownLatch latch = new CountDownLatch(1);

        // Capture event when publisher is called, then trigger latch
        doAnswer(invocation -> {
            latch.countDown();
            return null;
        }).when(eventPublisher).publishEvent(any(NotificationReadyEvent.class));

        // When
        retryScheduler.scheduleRetry(notificationId, channel, delay);

        // Then: Event should NOT be published immediately
        verify(eventPublisher, never()).publishEvent(any(NotificationReadyEvent.class));

        // Wait for scheduler to trigger (with margin)
        boolean completed = latch.await(250, TimeUnit.MILLISECONDS);

        // Assert
        assertTrue(completed, "Retry event was not fired within expected time");

        ArgumentCaptor<NotificationReadyEvent> eventCaptor = ArgumentCaptor.forClass(NotificationReadyEvent.class);
        verify(eventPublisher, times(1)).publishEvent(eventCaptor.capture());

        NotificationReadyEvent firedEvent = eventCaptor.getValue();
        assertEquals(notificationId, firedEvent.getNotificationId());
        assertEquals(channel, firedEvent.getChannel());
    }

    @Test
    void testScheduleMultipleRetries_IndependentExecution() throws InterruptedException {
        // Given
        UUID id1 = UUID.randomUUID();
        UUID id2 = UUID.randomUUID();
        CountDownLatch latch = new CountDownLatch(2);

        doAnswer(invocation -> {
            latch.countDown();
            return null;
        }).when(eventPublisher).publishEvent(any(NotificationReadyEvent.class));

        // When
        retryScheduler.scheduleRetry(id1, DeliveryChannel.EMAIL, Duration.ofMillis(50));
        retryScheduler.scheduleRetry(id2, DeliveryChannel.FCM, Duration.ofMillis(100));

        // Wait
        boolean completed = latch.await(300, TimeUnit.MILLISECONDS);

        // Then
        assertTrue(completed, "Not all retry events were fired");
        verify(eventPublisher, times(2)).publishEvent(any(NotificationReadyEvent.class));
    }
}
