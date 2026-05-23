package com.rentagf.notification.interfaces.http;

import com.rentagf.notification.application.port.outbound.PubSubPort;
import com.rentagf.notification.interfaces.http.filter.MockAuthFilter;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.data.redis.connection.RedisConnectionFactory;
import org.springframework.data.redis.core.StringRedisTemplate;
import org.springframework.data.redis.listener.RedisMessageListenerContainer;
import org.springframework.http.MediaType;
import org.springframework.test.context.ActiveProfiles;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.MvcResult;

import java.util.UUID;

import static org.junit.jupiter.api.Assertions.*;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@SpringBootTest
@AutoConfigureMockMvc
@ActiveProfiles("local") // Kích hoạt MockAuthFilter chỉ chạy ở profile local/dev
@Tag("integration")
class SseControllerIntegrationTest {

    @Autowired
    private MockMvc mockMvc;

    @MockBean
    private PubSubPort pubSubPort; // Mock để tháo gỡ Port gọi ra của Registry


    @Test
    void shouldEstablishSseConnectionWithMockAuth() throws Exception {
        // Gửi request GET /stream mà KHÔNG truyền header 'user-id'
        // MockAuthFilter sẽ tự động chèn header 'user-id' mặc định
        MvcResult result = mockMvc.perform(get("/v1/notifications/stream")
                        .accept(MediaType.TEXT_EVENT_STREAM_VALUE))
                .andExpect(status().isOk())
                .andExpect(content().contentTypeCompatibleWith(MediaType.TEXT_EVENT_STREAM))
                .andReturn();

        String content = result.getResponse().getContentAsString();
        
        // Xác nhận gói tin bắt tay đầu tiên ":connected" (Spring SseEmitter gửi dưới dạng comment)
        assertTrue(content.contains(":connected"));
    }

    @Test
    void shouldEstablishSseConnectionWithCustomUserId() throws Exception {
        UUID customUserId = UUID.randomUUID();

        // Gửi request có truyền header 'user-id' tùy chỉnh
        MvcResult result = mockMvc.perform(get("/v1/notifications/stream")
                        .header("user-id", customUserId.toString())
                        .accept(MediaType.TEXT_EVENT_STREAM_VALUE))
                .andExpect(status().isOk())
                .andExpect(content().contentTypeCompatibleWith(MediaType.TEXT_EVENT_STREAM))
                .andReturn();

        String content = result.getResponse().getContentAsString();
        assertTrue(content.contains(":connected"));
    }
}
