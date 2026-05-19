# 🧪 TEST DOCUMENTATION

## Cấu trúc Test

```
src/test/java/com/rentagf/notification/
├── domain/                    # Unit tests cho Domain Layer
│   ├── NotificationTest.java          # Aggregate behaviors
│   ├── NotificationInvariantsTest.java # [INV-N01], [INV-N02], [INV-N03]
│   ├── StateMachineTest.java          # Transition guards
│   ├── DeliveryAttemptTest.java       # Attempt lifecycle
│   └── FailureClassificationTest.java # Recoverable vs Unrecoverable
├── application/               # Unit tests cho Use Cases
│   ├── TemplateEngineTest.java
│   ├── RoutingEngineTest.java
│   └── SendNotificationUseCaseTest.java
├── interfaces/                # Unit tests cho Adapters
│   ├── rest/
│   │   ├── InboxControllerTest.java
│   │   └── MarkReadControllerTest.java
│   └── sse/
│       └── SseConnectionManagerTest.java
└── integration/               # Integration tests
    ├── InboxApiIntegrationTest.java
    ├── SseRedisIntegrationTest.java
    └── ConsumerToSseIntegrationTest.java
```

## Quy ước

- **Domain tests**: Pure unit tests, KHÔNG cần Spring context. Chạy nhanh.
- **Application tests**: Mock outbound ports, kiểm tra orchestration logic.
- **Integration tests**: Dùng `@SpringBootTest` + Testcontainers cho DB/Redis/Kafka thực.
- **Naming**: `<MethodUnderTest>_<Scenario>_<ExpectedBehavior>`

## Chạy test

```bash
# Tất cả tests
./gradlew test

# Chỉ unit tests (nhanh)
./gradlew test --tests "com.rentagf.notification.domain.*"

# Chỉ integration tests
./gradlew test --tests "com.rentagf.notification.integration.*"
```
