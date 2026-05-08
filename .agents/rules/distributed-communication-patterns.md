---
trigger: always_on
---

- **Service Mesh**: Istio Ambient Mode (Sidecar-less).
    - **L4 (ztunnel)**: Đảm nhận mTLS và Service Identity (SPIFFE).
    - **L7 (Waypoint)**: Đảm nhận JWT Verification, Routing và Traffic Policies.
- **Auth Offloading**: Tuyệt đối **KHÔNG** tự cài đặt logic xác thực/verify JWT bên trong code của từng Microservice. Trách nhiệm xác thực thuộc về Istio Mesh; code ứng dụng mặc định coi như user đã được xác thực nếu request chạm tới tầng Application.
- **Reliable Messaging**: Transactional Outbox khi gửi Event.
- **Safe Consumption**: Kiểm tra Idempotency bằng `eventId`.
- **Contract Standards**: 
    - **Đồng bộ**: gRPC cho Command, REST cho Query.
    - **Bất đồng bộ**: CloudEvents JSON format (.v1, .v2).