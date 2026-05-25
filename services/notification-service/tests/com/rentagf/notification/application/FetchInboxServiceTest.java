package com.rentagf.notification.application;

import com.rentagf.notification.application.service.FetchInboxService;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.repository.NotificationRepository;
import com.rentagf.notification.domain.vo.InboxCursor;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.mockito.Mockito;

import java.time.Instant;
import java.util.List;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;

@Tag("unit")
class FetchInboxServiceTest {

    private NotificationRepository notificationRepository;
    private FetchInboxService fetchInboxService;

    @BeforeEach
    void setUp() {
        notificationRepository = Mockito.mock(NotificationRepository.class);
        fetchInboxService = new FetchInboxService(notificationRepository);
    }

    @Test
    void testFetchInboxWithNullCursor() {
        // Arrange
        UUID userId = UUID.randomUUID();
        int limit = 10;
        Mockito.when(notificationRepository.findByUserId(eq(userId), isNull(), isNull(), eq(limit + 1), eq(false)))
                .thenReturn(List.of());

        // Act
        List<Notification> result = fetchInboxService.fetchInbox(userId, null, limit, false);

        // Assert
        assertNotNull(result);
        Mockito.verify(notificationRepository).findByUserId(eq(userId), isNull(), isNull(), eq(limit + 1), eq(false));
    }

    @Test
    void testFetchInboxWithValidCursor() {
        // Arrange
        UUID userId = UUID.randomUUID();
        Instant createdAt = Instant.parse("2026-05-24T10:00:00Z");
        UUID cursorId = UUID.randomUUID();
        InboxCursor cursor = new InboxCursor(createdAt, cursorId);

        int limit = 5;
        Mockito.when(notificationRepository.findByUserId(eq(userId), eq(createdAt), eq(cursorId), eq(limit + 1), eq(true)))
                .thenReturn(List.of());

        // Act
        List<Notification> result = fetchInboxService.fetchInbox(userId, cursor, limit, true);

        // Assert
        assertNotNull(result);
        Mockito.verify(notificationRepository).findByUserId(eq(userId), eq(createdAt), eq(cursorId), eq(limit + 1), eq(true));
    }
}
