package com.rentagf.notification.infrastructure.persistence;

import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.repository.NotificationRepository;
import com.rentagf.notification.domain.vo.DeliveryAttempt;
import com.rentagf.notification.domain.vo.enums.*;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.transaction.annotation.Transactional;

import java.time.Instant;
import java.util.List;
import java.util.Map;
import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;

@SpringBootTest
@Transactional
@Tag("integration")
class NotificationRepositoryTest {

    @Autowired
    private NotificationRepository notificationRepository;

    @Autowired
    private jakarta.persistence.EntityManager entityManager;

    @Test
    void testSaveAndFindNotificationSuccessfully() {
        UUID userId = UUID.randomUUID();
        String eventId = "evt_" + UUID.randomUUID();
        Notification notification = Notification.create(
                userId, eventId, NotificationType.TRANSACTIONAL, NotificationPriority.HIGH,
                Map.of("title", "Hello"), Map.of()
        );

        DeliveryAttempt attempt = notification.createAttempt(DeliveryChannel.EMAIL);

        Notification saved = notificationRepository.save(notification);

        assertNotNull(saved);
        assertEquals(notification.getId(), saved.getId());
        assertEquals(1, saved.getAttempts().size());
        assertEquals(AttemptStatus.PENDING, saved.getAttempts().get(0).getStatus());

        // Find by id
        Notification found = notificationRepository.findById(saved.getId()).orElse(null);
        assertNotNull(found);
        assertEquals(saved.getId(), found.getId());
        assertEquals(1, found.getAttempts().size());
    }

    @Test
    void testUniqueEventIdUserConstraint_duplicateShouldFail() {
        UUID userId = UUID.randomUUID();
        String eventId = "evt_dup_" + UUID.randomUUID();
        Notification notification1 = Notification.create(
                userId, eventId, NotificationType.TRANSACTIONAL, NotificationPriority.HIGH,
                Map.of("title", "Hello 1"), Map.of()
        );
        Notification notificationDuplicate = Notification.create(
                userId, eventId, NotificationType.TRANSACTIONAL, NotificationPriority.HIGH,
                Map.of("title", "Hello Duplicate"), Map.of()
        );

        notificationRepository.save(notification1);
        entityManager.flush();

        // 1. Lưu trùng eventId + userId -> Phải ném Exception
        assertThrows(Exception.class, () -> {
            notificationRepository.save(notificationDuplicate);
            entityManager.flush(); // Ép flush để trigger UNIQUE constraint check ở DB
        });
    }

    @Test
    void testUniqueEventIdUserConstraint_differentUserShouldSuccess() {
        UUID userId1 = UUID.randomUUID();
        UUID userId2 = UUID.randomUUID();
        String eventId = "evt_diff_" + UUID.randomUUID();
        Notification notification1 = Notification.create(
                userId1, eventId, NotificationType.TRANSACTIONAL, NotificationPriority.HIGH,
                Map.of("title", "Hello 1"), Map.of()
        );
        Notification notificationDifferentUser = Notification.create(
                userId2, eventId, NotificationType.TRANSACTIONAL, NotificationPriority.HIGH,
                Map.of("title", "Hello Different User"), Map.of()
        );

        notificationRepository.save(notification1);
        entityManager.flush();

        // 2. Lưu trùng eventId nhưng khác userId -> Phải thành công
        assertDoesNotThrow(() -> {
            notificationRepository.save(notificationDifferentUser);
            entityManager.flush();
        });
    }

    @Test
    void testCursorBasedPagination() {
        UUID userId = UUID.randomUUID();
        // Làm tròn thời gian về giây (SECONDS) để tránh mất độ phân giải nano-giây trong H2 Database
        Instant now = Instant.now().truncatedTo(java.time.temporal.ChronoUnit.SECONDS);
        
        // Tạo 3 notifications với createdAt cách nhau rõ rệt để đảm bảo thứ tự sắp xếp deterministic (n1 cũ nhất, n3 mới nhất)
        Notification n1 = new Notification(
                UUID.randomUUID(), userId, "evt_1_" + UUID.randomUUID(), NotificationType.TRANSACTIONAL,
                NotificationPriority.HIGH, Map.of(), Map.of(), NotificationStatus.PENDING,
                null, now.minusSeconds(10), now.minusSeconds(10), new java.util.ArrayList<>()
        );
        Notification n2 = new Notification(
                UUID.randomUUID(), userId, "evt_2_" + UUID.randomUUID(), NotificationType.TRANSACTIONAL,
                NotificationPriority.HIGH, Map.of(), Map.of(), NotificationStatus.PENDING,
                null, now.minusSeconds(5), now.minusSeconds(5), new java.util.ArrayList<>()
        );
        Notification n3 = new Notification(
                UUID.randomUUID(), userId, "evt_3_" + UUID.randomUUID(), NotificationType.TRANSACTIONAL,
                NotificationPriority.HIGH, Map.of(), Map.of(), NotificationStatus.PENDING,
                null, now, now, new java.util.ArrayList<>()
        );

        notificationRepository.save(n1);
        notificationRepository.save(n2);
        notificationRepository.save(n3);

        // Fetch page 1 (limit = 2)
        List<Notification> page1 = notificationRepository.findByUserId(userId, null, null, 2, false);
        assertEquals(2, page1.size());
        // Do sắp xếp theo createdAt DESC, id DESC: n3 và n2 sẽ lên trước
        assertEquals(n3.getId(), page1.get(0).getId());
        assertEquals(n2.getId(), page1.get(1).getId());

        // Fetch page 2 (dùng cursor là phần tử cuối của page 1)
        Instant cursor = page1.get(1).getCreatedAt();
        UUID cursorId = page1.get(1).getId();

        List<Notification> page2 = notificationRepository.findByUserId(userId, cursor, cursorId, 2, false);
        assertEquals(1, page2.size());
        assertEquals(n1.getId(), page2.get(0).getId());
    }

    @Test
    void testCountUnreadAndMarkAsRead() {
        UUID userId = UUID.randomUUID();
        Notification n1 = Notification.create(userId, "evt_u1_" + UUID.randomUUID(), NotificationType.TRANSACTIONAL, NotificationPriority.HIGH, Map.of(), Map.of());
        Notification n2 = Notification.create(userId, "evt_u2_" + UUID.randomUUID(), NotificationType.TRANSACTIONAL, NotificationPriority.HIGH, Map.of(), Map.of());

        notificationRepository.save(n1);
        notificationRepository.save(n2);

        assertEquals(2, notificationRepository.countUnreadByUserId(userId));

        // Mark n1 as read
        notificationRepository.markAsRead(n1.getId(), Instant.now());

        assertEquals(1, notificationRepository.countUnreadByUserId(userId));

        // Mark all as read
        notificationRepository.markAllAsRead(userId, Instant.now());
        assertEquals(0, notificationRepository.countUnreadByUserId(userId));
    }
}
