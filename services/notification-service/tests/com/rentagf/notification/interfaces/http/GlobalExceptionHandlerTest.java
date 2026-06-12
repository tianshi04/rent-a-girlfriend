package com.rentagf.notification.interfaces.http;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import com.rentagf.notification.domain.errors.DuplicateEventException;
import com.rentagf.notification.domain.errors.NotificationAlreadyCompletedException;
import com.rentagf.notification.domain.errors.NotificationNotFoundException;
import com.rentagf.notification.domain.errors.RetryLimitExceededException;
import java.util.UUID;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.test.context.ContextConfiguration;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@WebMvcTest(controllers = GlobalExceptionHandlerTest.TestController.class)
@ContextConfiguration(
    classes = {GlobalExceptionHandlerTest.TestController.class, GlobalExceptionHandler.class})
@Tag("integration")
class GlobalExceptionHandlerTest {

  @Autowired private MockMvc mockMvc;

  @Test
  void testHandleNotFoundException() throws Exception {
    UUID notifId = UUID.randomUUID();
    mockMvc
        .perform(get("/test/not-found/" + notifId))
        .andExpect(status().isNotFound())
        .andExpect(jsonPath("$.error.code").value("NOTIFICATION_NOT_FOUND"))
        .andExpect(jsonPath("$.error.message").value("Notification " + notifId + " not found"))
        .andExpect(jsonPath("$.error.details").isArray());
  }

  @Test
  void testHandleDuplicateEventException() throws Exception {
    mockMvc
        .perform(get("/test/duplicate"))
        .andExpect(status().isConflict())
        .andExpect(jsonPath("$.error.code").value("DUPLICATE_EVENT"))
        .andExpect(
            jsonPath("$.error.message")
                .value("Event evt_123 for user usr_456 has already been processed"))
        .andExpect(jsonPath("$.error.details").isArray());
  }

  @Test
  void testHandleAlreadyCompletedException() throws Exception {
    mockMvc
        .perform(get("/test/already-completed"))
        .andExpect(status().isConflict())
        .andExpect(jsonPath("$.error.code").value("NOTIFICATION_ALREADY_COMPLETED"))
        .andExpect(
            jsonPath("$.error.message")
                .value("Notification notif_789 is already completed. No further attempts allowed"))
        .andExpect(jsonPath("$.error.details").isArray());
  }

  @Test
  void testHandleRetryLimitExceededException() throws Exception {
    mockMvc
        .perform(get("/test/retry-exceeded"))
        .andExpect(status().isUnprocessableEntity())
        .andExpect(jsonPath("$.error.code").value("RETRY_LIMIT_EXCEEDED"))
        .andExpect(
            jsonPath("$.error.message")
                .value("Notification notif_789 has exceeded maximum retry attempts (3)"))
        .andExpect(jsonPath("$.error.details").isArray());
  }

  @Test
  void testHandleUnexpectedException() throws Exception {
    mockMvc
        .perform(get("/test/unexpected"))
        .andExpect(status().isInternalServerError())
        .andExpect(jsonPath("$.error.code").value("INTERNAL_ERROR"))
        .andExpect(jsonPath("$.error.message").value("An unexpected error occurred"))
        .andExpect(jsonPath("$.error.details").isArray());
  }

  @RestController
  static class TestController {

    @GetMapping("/test/not-found/{id}")
    public void notFound(@org.springframework.web.bind.annotation.PathVariable UUID id) {
      throw new NotificationNotFoundException(id.toString());
    }

    @GetMapping("/test/duplicate")
    public void duplicate() {
      throw new DuplicateEventException("evt_123", "usr_456");
    }

    @GetMapping("/test/already-completed")
    public void alreadyCompleted() {
      throw new NotificationAlreadyCompletedException("notif_789");
    }

    @GetMapping("/test/retry-exceeded")
    public void retryExceeded() {
      throw new RetryLimitExceededException("notif_789");
    }

    @GetMapping("/test/unexpected")
    public void unexpected() {
      throw new RuntimeException("Unexpected db error");
    }
  }
}
