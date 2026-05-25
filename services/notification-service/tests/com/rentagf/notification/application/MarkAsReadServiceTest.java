package com.rentagf.notification.application;

import com.rentagf.notification.application.service.MarkAsReadService;
import com.rentagf.notification.domain.errors.NotificationNotFoundException;
import com.rentagf.notification.domain.repository.NotificationRepository;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.mockito.Mockito;

import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;

@Tag("unit")
class MarkAsReadServiceTest {

    private NotificationRepository notificationRepository;
    private MarkAsReadService markAsReadService;

    @BeforeEach
    void setUp() {
        notificationRepository = Mockito.mock(NotificationRepository.class);
        markAsReadService = new MarkAsReadService(notificationRepository);
    }

    @Test
    void testMarkAsReadSuccess() {
        // Arrange
        UUID notificationId = UUID.randomUUID();
        UUID userId = UUID.randomUUID();
        Mockito.when(notificationRepository.markAsRead(eq(notificationId), eq(userId), any())).thenReturn(true);

        // Act & Assert
        assertDoesNotThrow(() -> markAsReadService.markAsRead(notificationId, userId));
        Mockito.verify(notificationRepository).markAsRead(eq(notificationId), eq(userId), any());
    }

    @Test
    void testMarkAsReadThrowsNotificationNotFoundExceptionWhenRepositoryReturnsFalse() {
        // Arrange (BOLA or Non-existent)
        UUID notificationId = UUID.randomUUID();
        UUID userId = UUID.randomUUID();
        Mockito.when(notificationRepository.markAsRead(eq(notificationId), eq(userId), any())).thenReturn(false);

        // Act & Assert
        assertThrows(NotificationNotFoundException.class, () -> markAsReadService.markAsRead(notificationId, userId));
        Mockito.verify(notificationRepository).markAsRead(eq(notificationId), eq(userId), any());
    }

    @Test
    void testMarkAsReadWithNullArgumentsThrowsIllegalArgumentException() {
        UUID validId = UUID.randomUUID();
        assertThrows(IllegalArgumentException.class, () -> markAsReadService.markAsRead(null, validId));
        assertThrows(IllegalArgumentException.class, () -> markAsReadService.markAsRead(validId, null));
    }
}
