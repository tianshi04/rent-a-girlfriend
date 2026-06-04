# Agent Instructions

## 1. Agent Behavior
- **Think Before Coding**: State assumptions, ask if unsure, and surface tradeoffs. Don't hide confusion. If a requirement is unclear, stop and ask for clarification before writing any code.
- **Simplicity First**: Write the minimum code required. No speculative features, abstractions, or "flexibility" not requested.
- **Surgical Edits**: Change only what is necessary. Match existing style. Cleanup only unused code created by your changes.
- **Verify Before Returning**: Confirm your output matches the original request. If tests exist, run them. If not, trace the logic mentally.

## 2. Knowledge & Rule Management
- **Knowledge Persistence**:
  - Propose Updates: IF discovering undocumented project knowledge (conventions/constraints/patterns), THEN propose adding to `AGENTS.md`.
  - Report Conflicts: IF `AGENTS.md` contradicts the codebase, THEN report to Developer.
  - CONSTRAINT: NEVER create/modify `AGENTS.md` without explicit Developer approval.
- **Precedence & Scope Isolation**:
  - Precedence: Local `AGENTS.md` rules STRICTLY OVERRIDE root rules for that specific subfolder.
  - Scope Isolation: NEVER add directory-specific rules to global `AGENTS.md`. Local rules MUST stay in local `AGENTS.md` to prevent global pollution.
  - Monorepos: IF a subproject has unique conventions, THEN propose initializing a nested `AGENTS.md`.

## 3. Docstrings & Comments
- **Docstring**: Write for public component when **≥1** applies:
  - Business rule/invariant not inferable from function name + type signature.
  - Non-obvious side effects (emit event, mutate another aggregate's state).
  - Business errors caller must handle (`ErrInsufficientBalance`, `ErrBookingExpired`).
  - Unit of measurement or business meaning unclear from type (`duration` in minutes vs seconds).
  - None of the above → **DO NOT** write docstring.
- **Comment**: Write when code block has **≥1**:
  - Domain-specific business logic.
  - Complex algorithm or non-obvious performance optimization.
  - Workaround for external library bug/limitation (include issue link).
  - Rationale for choosing approach A over B.
  - **Never** comment basic language syntax.
- **DRY**: Do not repeat types already in signature. Do not repeat meaning already clear from function/parameter names.
- **Priority on conflict**: Follow this rule for **new and modified** code. Do not refactor documentation outside current change scope.

## 4. Core Philosophy
- Domain-Driven Design (DDD) & Hexagonal Architecture.
- Event-Driven Microservices with SAGA for distributed transactions.
- Zero cross-service database access.

## 5. Universal Architecture Standards (Scope: `services/**/*`)
- **Hexagonal Architecture**: 
  - Domain (Entities/Aggregates/VO).
  - Application (Use Cases/Handlers).
  - Infrastructure (DB/Broker Adapters).
  - Interfaces (HTTP/gRPC/PubSub).
- **Contracts as SSOT**: Tất cả giao tiếp (gRPC, Async Events) phải được định nghĩa tại thư mục `/contracts` ở root. Đây là Single Source of Truth duy nhất. Không được phép định nghĩa lại message/event trong từng service. Các service phải generate code từ các file proto này.
- **Directory Structure**:
  - `cmd/server/`: Điểm khởi chạy (Entry point).
  - `internal/domain/`: Aggregate, VO, Repository port, Events.
  - `internal/application/`: Commands, Queries, Saga.
  - `internal/infrastructure/`: DB/Broker/Client adapters.
  - `internal/interfaces/`: HTTP/gRPC/PubSub handlers.
  - `deployments/`: K8s manifests và Istio policies riêng của service.
  - `gen/`: Code được generated từ Protobuf/AsyncAPI.
  - `docs/`: Tài liệu kỹ thuật riêng của service.
  - `tests/`: Integration & E2E tests.
- **Principles**:
  - Dependency Direction: Ngoài vào trong (Domain là lõi).
  - Database Isolation: Không truy cập DB service khác.
  - Service Autonomy: Mỗi service phải chứa đủ mọi thứ để có thể tách thành repo riêng và deploy độc lập (Independent Repo Ready).
- **Deployment**:
  - Standalone Dockerfile mỗi service đặt tại root của service. Multi-stage build.
  - Config qua biến môi trường (.env.example). No hard-code.
  - **Port Conventions**: Tất cả các Microservices phải sử dụng cổng kết nối chuẩn: HTTP chạy trên `8080` (`SERVER_PORT`), gRPC chạy trên `50051` (`GRPC_PORT`).
- **Quality & Documentation**:
  - **Self-Documenting**: Mỗi service phải tự quản lý `docs/` riêng. Cập nhật tài liệu ngay khi thay đổi logic nghiệp vụ.
  - **Test-Driven**: Mọi tính năng/logic mới phải có Unit Test (cho Domain/Application) và Integration Test (cho Infrastructure).

## 6. Architecture Decision Records (ADR)
- **Ghi nhận Quyết định (ADR)**: Chủ động tạo/cập nhật ADR khi thống nhất thiết kế kỹ thuật, kiến trúc, công nghệ (CSDL, thư viện, luồng chính...).
- **Vị trí Lưu trữ**:
  - **Cấp Service**: `services/<service-name>/docs/adr/` (ảnh hưởng cục bộ).
  - **Cấp Toàn cục**: `docs/adr/` ở thư mục root (ảnh hưởng toàn hệ thống/liên dịch vụ).
- **Quy tắc Đặt tên**: `XXXX-slug-ten-quyet-dinh.md` (Ví dụ: `0006-use-outbox-pattern.md`). Số `XXXX` tăng dần từ `0001`.
- **Cấu trúc ADR Tiêu chuẩn**:
  - `# ADR XXXX: [Tên quyết định]`
  - `**Trạng thái:** Accepted | Proposed | Rejected`
  - `**Ngày:** YYYY-MM-DD`
  - `## Ngữ cảnh (Context)`: Bối cảnh, ràng buộc kỹ thuật và phương án cân nhắc.
  - `## Quyết định (Decision)`: Giải pháp chọn và Rationale.
  - `## Hệ quả (Consequences)`: Điểm tích cực (Positives) và Đánh đổi (Negatives).
- **Quy trình Thực hiện**:
  - Soạn bản nháp ngay sau khi chốt phương án và đề xuất ghi file.
  - Không sửa ADR đã `Accepted`. Khi thay đổi, cập nhật ADR cũ sang `Superseded by ADR XXXX` và tạo ADR mới.

## 7. DDD & Business Logic Conventions
- **Ubiquitous Language**: Luôn sử dụng thuật ngữ trong `docs/BRD.md` (Kano-Coin, Scenario, Companion, Client, Escrow).
- **Snapshot Policy**: Lưu bản sao thông số (giá, cấu hình, điều khoản) tại thời điểm giao dịch. Không chỉ lưu ID tham chiếu.
- **Naming Standards**:
  - **Invariants**: Chú thích `[INV-XXXX]`.
  - **Commands**: `Verb + Noun` (AcceptBooking).
  - **Events**: `Noun + PastVerb` (BookingAccepted).
- **Domain Errors**: Trả về lỗi nghiệp vụ rõ ràng để map sang HTTP/gRPC code.

## 8. Distributed Communication Patterns
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
