package com.rentagf.notification.infrastructure.sse;

import com.rentagf.notification.application.port.outbound.PubSubPort;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;
import org.springframework.web.servlet.mvc.method.annotation.SseEmitter;

import java.util.*;
import java.util.concurrent.*;
import java.util.concurrent.atomic.AtomicInteger;

import static org.junit.jupiter.api.Assertions.*;

@Tag("unit")
class SseConnectionRegistryTest {

    private SseConnectionRegistry registry;
    private MockPubSubPort mockPubSubPort;
    private MockConnectionStatePort mockConnectionStatePort;

    @BeforeEach
    void setUp() {
        mockPubSubPort = new MockPubSubPort();
        mockConnectionStatePort = new MockConnectionStatePort();
        SseLocalRegistry localRegistry = new SseLocalRegistry();
        registry = new SseConnectionRegistry(localRegistry, mockPubSubPort, mockConnectionStatePort);
    }

    @Test
    void shouldSubscribeOnFirstConnection() {
        UUID userId = UUID.randomUUID();
        SseEmitter emitter1 = new SseEmitter();

        registry.register(userId, emitter1);

        // Xác nhận Emitter được lưu trong registry
        List<SseEmitter> emitters = registry.getEmitters(userId);
        assertEquals(1, emitters.size());
        assertTrue(emitters.contains(emitter1));

        // Xác nhận đã gọi subscribe lên Redis đúng 1 lần
        String expectedChannel = "user:" + userId + ":sse";
        assertEquals(1, mockPubSubPort.getSubscribeCount(expectedChannel));
        assertEquals(0, mockPubSubPort.getUnsubscribeCount(expectedChannel));
    }

    @Test
    void shouldNotSubscribeAgainOnMultipleConnections() {
        UUID userId = UUID.randomUUID();
        SseEmitter emitter1 = new SseEmitter();
        SseEmitter emitter2 = new SseEmitter();

        registry.register(userId, emitter1);
        registry.register(userId, emitter2);

        // Xác nhận có 2 kết nối active
        List<SseEmitter> emitters = registry.getEmitters(userId);
        assertEquals(2, emitters.size());
        assertTrue(emitters.contains(emitter1));
        assertTrue(emitters.contains(emitter2));

        // Xác nhận chỉ gọi subscribe đúng 1 lần duy nhất (ở kết nối đầu tiên)
        String expectedChannel = "user:" + userId + ":sse";
        assertEquals(1, mockPubSubPort.getSubscribeCount(expectedChannel));
    }

    @Test
    void shouldNotUnsubscribeWhenConnectionsStillActive() {
        UUID userId = UUID.randomUUID();
        SseEmitter emitter1 = new SseEmitter();
        SseEmitter emitter2 = new SseEmitter();

        registry.register(userId, emitter1);
        registry.register(userId, emitter2);

        // Ngắt 1 kết nối
        registry.unregister(userId, emitter1);

        // Xác nhận vẫn còn 1 kết nối active
        List<SseEmitter> emitters = registry.getEmitters(userId);
        assertEquals(1, emitters.size());
        assertTrue(emitters.contains(emitter2));

        // Xác nhận KHÔNG gọi unsubscribe vì vẫn còn kết nối
        String expectedChannel = "user:" + userId + ":sse";
        assertEquals(0, mockPubSubPort.getUnsubscribeCount(expectedChannel));
    }

    @Test
    void shouldUnsubscribeOnLastConnectionClosed() {
        UUID userId = UUID.randomUUID();
        SseEmitter emitter1 = new SseEmitter();
        SseEmitter emitter2 = new SseEmitter();

        registry.register(userId, emitter1);
        registry.register(userId, emitter2);

        // Ngắt tất cả các kết nối
        registry.unregister(userId, emitter1);
        registry.unregister(userId, emitter2);

        // Xác nhận registry trống rỗng
        List<SseEmitter> emitters = registry.getEmitters(userId);
        assertTrue(emitters.isEmpty());

        // Xác nhận đã gọi unsubscribe chính xác 1 lần
        String expectedChannel = "user:" + userId + ":sse";
        assertEquals(1, mockPubSubPort.getUnsubscribeCount(expectedChannel));
    }

    @Test
    void shouldBeThreadSafeUnderHeavyConcurrency() throws InterruptedException {
        UUID userId = UUID.randomUUID();
        int threadCount = 100;
        ExecutorService executor = Executors.newFixedThreadPool(10);
        CountDownLatch latch = new CountDownLatch(1);
        CountDownLatch doneLatch = new CountDownLatch(threadCount);

        List<SseEmitter> testEmitters = new CopyOnWriteArrayList<>();
        for (int i = 0; i < threadCount; i++) {
            testEmitters.add(new SseEmitter());
        }

        // 100 luồng đồng thời gọi register
        for (int i = 0; i < threadCount; i++) {
            final SseEmitter emitter = testEmitters.get(i);
            executor.submit(() -> {
                try {
                    latch.await(); // Chờ tín hiệu xuất phát đồng thời
                    registry.register(userId, emitter);
                } catch (InterruptedException e) {
                    Thread.currentThread().interrupt();
                } finally {
                    doneLatch.countDown();
                }
            });
        }

        latch.countDown(); // Kích nổ 100 luồng chạy đồng thời
        assertTrue(doneLatch.await(5, TimeUnit.SECONDS));

        // Xác nhận toàn bộ 100 emitters được lưu thread-safe, không bị mất mát dữ liệu
        List<SseEmitter> emitters = registry.getEmitters(userId);
        assertEquals(threadCount, emitters.size());

        // Chỉ được subscribe đúng 1 lần duy nhất lên Redis
        String expectedChannel = "user:" + userId + ":sse";
        assertEquals(1, mockPubSubPort.getSubscribeCount(expectedChannel));

        // Đồng thời dọn dẹp (unregister) 100 luồng
        CountDownLatch unregLatch = new CountDownLatch(1);
        CountDownLatch unregDoneLatch = new CountDownLatch(threadCount);
        for (int i = 0; i < threadCount; i++) {
            final SseEmitter emitter = testEmitters.get(i);
            executor.submit(() -> {
                try {
                    unregLatch.await();
                    registry.unregister(userId, emitter);
                } catch (InterruptedException e) {
                    Thread.currentThread().interrupt();
                } finally {
                    unregDoneLatch.countDown();
                }
            });
        }

        unregLatch.countDown();
        assertTrue(unregDoneLatch.await(5, TimeUnit.SECONDS));

        // Xác nhận registry trống
        assertTrue(registry.getEmitters(userId).isEmpty());

        // Chỉ được gọi unsubscribe đúng 1 lần duy nhất
        assertEquals(1, mockPubSubPort.getUnsubscribeCount(expectedChannel));

        executor.shutdown();
    }

    // --- Mock PubSubPort Implementation ---
    private static class MockPubSubPort implements PubSubPort {
        private final Map<String, AtomicInteger> subscribeCounts = new ConcurrentHashMap<>();
        private final Map<String, AtomicInteger> unsubscribeCounts = new ConcurrentHashMap<>();

        @Override
        public void publish(String channel, String message) {
            // Không test publish ở đây
        }

        @Override
        public void subscribe(String channel) {
            subscribeCounts.computeIfAbsent(channel, k -> new AtomicInteger(0)).incrementAndGet();
        }

        @Override
        public void unsubscribe(String channel) {
            unsubscribeCounts.computeIfAbsent(channel, k -> new AtomicInteger(0)).incrementAndGet();
        }

        public int getSubscribeCount(String channel) {
            AtomicInteger count = subscribeCounts.get(channel);
            return count != null ? count.get() : 0;
        }

        public int getUnsubscribeCount(String channel) {
            AtomicInteger count = unsubscribeCounts.get(channel);
            return count != null ? count.get() : 0;
        }
    }

    // --- Mock ConnectionStatePort Implementation ---
    private static class MockConnectionStatePort implements com.rentagf.notification.application.port.outbound.ConnectionStatePort {
        private final Map<UUID, Boolean> states = new ConcurrentHashMap<>();
        private final Map<UUID, Long> ttls = new ConcurrentHashMap<>();

        @Override
        public void setOnline(UUID userId, long ttlSeconds) {
            states.put(userId, true);
            ttls.put(userId, ttlSeconds);
        }

        @Override
        public void setOffline(UUID userId) {
            states.remove(userId);
            ttls.remove(userId);
        }

        @Override
        public boolean isOnline(UUID userId) {
            return states.getOrDefault(userId, false);
        }

        @Override
        public void refreshHeartbeat(UUID userId, long ttlSeconds) {
            if (states.containsKey(userId)) {
                ttls.put(userId, ttlSeconds);
            }
        }
    }
}
