---
trigger: always_on
---

- **Service Mesh**: Istio Ambient Mode (Sidecar-less).
    - **L4 (ztunnel)**: Đảm nhận mTLS và Service Identity (SPIFFE).
    - **L7 (Waypoint)**: Đảm nhận JWT Verification, Routing và Traffic Policies.
- **Auth Offloading & Identity Propagation**: 
    - Tuyệt đối **KHÔNG** tự cài đặt logic xác thực JWT bên trong code của từng Microservice. Trách nhiệm xác thực thuộc về Istio Waypoint.
    - **Header Injection**: Sau khi verify, Istio sẽ inject thông tin từ JWT Claim vào Header để Application sử dụng:
        - `user-id` (từ `sub`)
        - `user-email` (từ `email`)
        - `user-role` (từ `role`)
        - `user-status` (từ `status`)
- **Reliable Messaging**: Transactional Outbox khi gửi Event.
- **Safe Consumption**: Kiểm tra Idempotency bằng `eventId`.
- **Contract Standards**: 
    - **Đồng bộ**: gRPC cho Command, REST cho Query.
    - **Bất đồng bộ**: CloudEvents JSON format (.v1, .v2).