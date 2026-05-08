---
trigger: always_on
---

- **Reliable Messaging**: Transactional Outbox khi gửi Event.
- **Safe Consumption**: Kiểm tra Idempotency bằng `eventId`.
- **Contract Standards**: 
    - **Đồng bộ**: gRPC cho Command, REST cho Query.
    - **Bất đồng bộ**: CloudEvents JSON format (.v1, .v2).