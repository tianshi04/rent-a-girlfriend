package com.rentagf.notification.interfaces.http;

import com.rentagf.notification.application.port.inbound.FetchInboxUseCase;
import com.rentagf.notification.application.port.inbound.MarkAllAsReadUseCase;
import com.rentagf.notification.application.port.inbound.MarkAsReadUseCase;
import com.rentagf.notification.application.port.inbound.SendNotificationUseCase;
import com.rentagf.notification.application.port.inbound.TriggerNotificationUseCase;
import com.rentagf.notification.domain.aggregate.Notification;
import com.rentagf.notification.domain.errors.NotificationNotFoundException;
import com.rentagf.notification.domain.vo.enums.NotificationPriority;
import com.rentagf.notification.domain.vo.enums.NotificationStatus;
import com.rentagf.notification.domain.vo.enums.NotificationType;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.mockito.Mockito;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.http.MediaType;
import org.springframework.test.context.ContextConfiguration;
import org.springframework.test.context.bean.override.mockito.MockitoBean;
import org.springframework.test.web.servlet.MockMvc;

import java.time.Instant;
import java.util.List;
import java.util.Map;
import java.util.UUID;

import static org.mockito.ArgumentMatchers.*;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.patch;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@WebMvcTest(controllers = NotificationController.class)
@ContextConfiguration(classes = {
    NotificationController.class,
    GlobalExceptionHandler.class
})
@Tag("integration")
class NotificationControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @MockitoBean
    private FetchInboxUseCase fetchInboxUseCase;

    @MockitoBean
    private MarkAsReadUseCase markAsReadUseCase;

    @MockitoBean
    private MarkAllAsReadUseCase markAllAsReadUseCase;

    @MockitoBean
    private TriggerNotificationUseCase triggerNotificationUseCase;

    @MockitoBean
    private SendNotificationUseCase sendNotificationUseCase;

    @Test
    void testFetchInboxSuccessfully() throws Exception {
        UUID userId = UUID.randomUUID();
        Notification n = new Notification(
            UUID.randomUUID(), userId, "evt_123", NotificationType.TRANSACTIONAL,
            NotificationPriority.HIGH, Map.of("title", "Hello"), Map.of(),
            NotificationStatus.PENDING, null, Instant.now(), Instant.now(), List.of()
        );

        Mockito.when(fetchInboxUseCase.fetchInbox(eq(userId), any(), eq(20), eq(false)))
            .thenReturn(List.of(n));

        mockMvc.perform(get("/v1/notifications")
                .header("user-id", userId.toString())
                .param("limit", "20")
                .param("unread_only", "false"))
            .andExpect(status().isOk())
            .andExpect(jsonPath("$.data").isArray())
            .andExpect(jsonPath("$.data[0].eventId").value("evt_123"))
            .andExpect(jsonPath("$.paging.hasMore").value(false))
            .andExpect(jsonPath("$.paging.nextCursor").value((String) null));
    }

    @Test
    void testFetchInboxMissingUserIdHeader_shouldReturnBadRequest() throws Exception {
        mockMvc.perform(get("/v1/notifications"))
            .andExpect(status().isBadRequest());
    }

    @Test
    void testFetchInboxInvalidUserIdHeader_shouldReturnBadRequest() throws Exception {
        mockMvc.perform(get("/v1/notifications")
                .header("user-id", "not-a-uuid"))
            .andExpect(status().isBadRequest())
            .andExpect(jsonPath("$.error.code").value("INVALID_PARAMETER"))
            .andExpect(jsonPath("$.error.message").value("Invalid 'user-id' header format. Must be a valid UUID"));
    }

    @Test
    void testFetchInboxInvalidLimit_shouldReturnBadRequest() throws Exception {
        UUID userId = UUID.randomUUID();
        mockMvc.perform(get("/v1/notifications")
                .header("user-id", userId.toString())
                .param("limit", "0"))
            .andExpect(status().isBadRequest())
            .andExpect(jsonPath("$.error.code").value("INVALID_PARAMETER"))
            .andExpect(jsonPath("$.error.message").value("Limit must be between 1 and 100"));
    }

    @Test
    void testMarkAsReadSuccessfully() throws Exception {
        UUID userId = UUID.randomUUID();
        UUID notifId = UUID.randomUUID();

        Mockito.doNothing().when(markAsReadUseCase).markAsRead(notifId, userId);

        mockMvc.perform(patch("/v1/notifications/" + notifId + "/read")
                .header("user-id", userId.toString()))
            .andExpect(status().isNoContent());
    }

    @Test
    void testMarkAsReadNotFoundOrBola_shouldReturnNotFound() throws Exception {
        UUID userId = UUID.randomUUID();
        UUID notifId = UUID.randomUUID();

        Mockito.doThrow(new NotificationNotFoundException(notifId.toString()))
            .when(markAsReadUseCase).markAsRead(notifId, userId);

        mockMvc.perform(patch("/v1/notifications/" + notifId + "/read")
                .header("user-id", userId.toString()))
            .andExpect(status().isNotFound())
            .andExpect(jsonPath("$.error.code").value("NOTIFICATION_NOT_FOUND"));
    }

    @Test
    void testMarkAllAsReadSuccessfully() throws Exception {
        UUID userId = UUID.randomUUID();

        Mockito.when(markAllAsReadUseCase.markAllAsRead(userId)).thenReturn(5);

        mockMvc.perform(patch("/v1/notifications/read-all")
                .header("user-id", userId.toString()))
            .andExpect(status().isOk())
            .andExpect(jsonPath("$.affectedRows").value(5));
    }
}
