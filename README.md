# 🌸 RENT-A-GIRLFRIEND MICROSERVICES

[![Architecture: Microservices](https://img.shields.io/badge/Architecture-Microservices-blueviolet)](/docs/01_Architecture_Overview/01_system_architecture.md)
[![Design: DDD](https://img.shields.io/badge/Design-Domain--Driven--Design-blue)](/docs/01_Architecture_Overview/02_domain_and_contexts.md)

Nền tảng kết nối dịch vụ đồng hành theo kịch bản (**Scenario-based companion service**) thế hệ mới. Dự án được thiết kế với mục tiêu xử lý các luồng nghiệp vụ phức tạp liên quan đến tài chính (Kano-Coin), đặt lịch thời gian thực và quản lý tương tác người dùng an toàn.

---

## 🏗️ KIẾN TRÚC HỆ THỐNG (SYSTEM ARCHITECTURE)

Hệ thống tuân thủ nghiêm ngặt mô hình **Event-Driven Microservices** kết hợp với **Domain-Driven Design (DDD)** Tactical Modeling.

- **Communication**: Hybrid (REST/gRPC cho Sync, Message Broker cho Async).
- **Consistency**: Eventual Consistency thông qua **SAGA Pattern** (Orchestration & Choreography).
- **Persistence**: Polyglot Persistence (Mỗi service sở hữu cơ sở dữ liệu riêng biệt để đảm bảo tính đóng gói).
- **External**: Tích hợp VNPay (Payment), Google OAuth (Identity), Cloud Storage (Media).

---

## 🗺️ BẢN ĐỒ MICROSERVICES

Hệ thống bao gồm 7 Bounded Contexts độc lập:

| Service | Phân loại | Trách nhiệm chính |
| :--- | :--- | :--- |
| **Booking** | Core | State Machine của cuộc hẹn, Orchestrator cho SAGA. |
| **Finance** | Core | Ví Kano-Coin, Escrow, Payout, tích hợp VNPay. |
| **Profile** | Core | Quản lý Scenario, Media, Metadata của Companion. |
| **Interaction** | Supporting | Chat realtime, Profanity filter, Review & Rating. |
| **Dispute** | Supporting | Xử lý khiếu nại, Refund/Payout logic. |
| **Identity** | Generic | Auth, RBAC, Account status control. |
| **Notification** | Generic | SSE, FCM, Email delivery system. |

---

## 📚 HƯỚNG DẪN ĐỌC TÀI LIỆU (DOCUMENTATION INDEX)

Tài liệu được cấu trúc theo phương pháp **Progressive Disclosure** giúp tiếp cận từ tổng quan đến chi tiết.

### 1. 🏗️ Tổng quan & Nền tảng (The Big Picture)
- [Hệ thống Kiến trúc tổng thể](/docs/01_Architecture_Overview/01_system_architecture.md): Sơ đồ topology và các nguyên tắc cốt lõi.
- [Phân tách Bounded Contexts](/docs/01_Architecture_Overview/02_domain_and_contexts.md): Cách chúng tôi chia nhỏ hệ thống.
- [Luồng nghiệp vụ lõi](/docs/01_Architecture_Overview/03_core_business_flows.md): Booking loop, Payment flow, Dispute flow.
- [Service Template](/docs/01_Architecture_Overview/04_service_template.md): Chuẩn hoá cấu trúc thư mục và phân lớp code.

### 2. 🧩 Chi tiết từng Bounded Context (The Components)
Mỗi context có đặc tả chi tiết về **Aggregate, Command, Event, Invariant**:
- [Booking Context](/docs/02_Bounded_Contexts/01_booking_context.md) | [Finance Context](/docs/02_Bounded_Contexts/02_finance_context.md) | [Profile Context](/docs/02_Bounded_Contexts/03_profile_context.md) | [Interaction Context](/docs/02_Bounded_Contexts/04_interaction_context.md) | [Dispute Context](/docs/02_Bounded_Contexts/05_dispute_context.md) | [Identity Context](/docs/02_Bounded_Contexts/06_identity_context.md) | [Notification Context](/docs/02_Bounded_Contexts/07_notification_context.md)

### 3. 🔄 Giao tiếp & Giao dịch (The Glue)
- [API & Event Bus](/docs/03_Integration_and_Comms/01_api_and_event_bus.md): Chuẩn hóa Protocol và Message format.
- [SAGA Workflows](/docs/04_Distributed_Transactions/02_saga_workflows.md): Cách chúng tôi xử lý Distributed Transactions.

---
