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
- Polyglot Microservices: Each service chooses its own language and framework, unified by shared Protobuf contracts and the architectural principles below.

## 5. Monorepo Layout
- `contracts/`: Protobuf definitions (SSOT for all APIs and events).
- `services/`: Individual microservices, each independently deployable.
- `docs/`: System-wide documentation (architecture, bounded contexts, BRD, ADRs).
- `infra/`: Infrastructure configs (K8s, Istio, Kafka, local Docker Compose).
- `.github/`: CI/CD workflows (per-service CI, proto-lint, GHCR publish).
- `third_party/`: External proto imports (googleapis).

## 6. Service Architecture Standards
- **Hexagonal Architecture** (all services, regardless of language):
  - Domain: Entities, Aggregates, Value Objects, Domain Events, Repository Ports.
  - Application: Use Cases, Command/Query Handlers, Saga Coordinators.
  - Infrastructure: Database Adapters, Broker Adapters, External Service Clients.
  - Interfaces: HTTP Controllers, gRPC Servicers, PubSub Listeners.
- **Directory Structure**: Each service organizes code into the layers above using idiomatic conventions of its language. Refer to each service's own README or codebase for language-specific layout.
- **Contracts as SSOT**: All gRPC services and async events MUST be defined in `/contracts`. Redefining messages in individual services is strictly forbidden; services must generate code from these proto files.
- **Principles**:
  - Dependency Direction: Outside-in (Domain has zero external dependencies).
  - Database Isolation: No cross-service database access.
  - Service Autonomy: Each service is independently deployable (Independent Repo Ready).

## 7. Contract & Communication Standards

### Synchronous Protocols
- **Internal (Inter-service)**: Strictly gRPC for both commands and queries.
- **External (Client-to-Service)**: REST APIs (HTTP/JSON) via gRPC-Gateway or controllers.

### Asynchronous Protocols
- **Format**: CloudEvents JSON via Kafka (Structured Content Mode).
- **Event Type**: `<domain>.<event-name>.v<version>` (kebab-case, e.g., `booking.booking-accepted.v1`).
- **Topic**: `<domain>.events` (e.g., `booking.events`).
- **Message Key**: Each service must publish events with the primary Aggregate Root ID (e.g., `bookingId`, `userId`) as the Kafka message key to ensure partition-level ordering.
- **Extensions**: Custom extension attributes (like `correlationid`) must be serialized in all-lowercase alphanumeric format at the root level of the JSON envelope (sibling to `specversion`, `id`, `data`, etc.) per the CloudEvents specification.
- **Reliable Messaging**: Transactional Outbox pattern when sending events.
- **Safe Consumption**: Idempotency check to guarantee exactly-once processing.
- See: `docs/03_Integration_and_Comms/` and `docs/04_Distributed_Transactions/`

### JSON Casing
- All JSON fields (CloudEvents `data` payloads, REST APIs) must use **camelCase** per Google Protobuf JSON Mapping specification. Custom extension attributes at the envelope root level must remain all-lowercase.

### Service Mesh & Auth
- **Istio Ambient Mode** (Sidecar-less): L4 mTLS via ztunnel, L7 JWT/Routing via Waypoint.
- **Auth Offloading**: NEVER implement JWT verification in microservice code. Verification is offloaded to Istio Waypoint.
- **Header Injection** (by Istio after JWT verification):
  - `user-id` (from `sub`), `user-email` (from `email`), `user-role` (from `role`), `user-status` (from `status`).

## 8. Deployment & Infrastructure
- Standalone Dockerfile per service (multi-stage build).
- Configuration via environment variables (`.env.example`). No hard-coded values.
- **Port Conventions**: HTTP on `8080` (`SERVER_PORT`), gRPC on `50051` (`GRPC_PORT`).

## 9. Quality & Documentation
- **Self-Documenting**: Each service manages its own `docs/` directory. Update documentation immediately upon business logic changes.
- **Test-Driven**: Every new feature must include:
  - Unit Tests (Domain and Application layers).
  - Integration Tests (Infrastructure layer).

## 10. Architecture Decision Records (ADR)
- **Architecture Decision Records (ADRs)**: Proactively create/update ADRs when aligning on technical design, architecture, or technology stack (database, libraries, main flows, etc.).
- **Storage Location**:
  - **Service level**: `services/<service-name>/docs/adr/` (local impact).
  - **Global level**: `docs/adr/` at the root directory (system-wide/cross-service impact).
- **Naming Convention**: `XXXX-slug-decision-name.md` (e.g., `0006-use-outbox-pattern.md`). `XXXX` is a sequential number starting from `0001`.
- **Standard ADR Structure**:
  - `# ADR XXXX: [Decision Name]`
  - `**Status:** Accepted | Proposed | Rejected`
  - `**Date:** YYYY-MM-DD`
  - `## Context`: Context, technical constraints, and considered alternatives.
  - `## Decision`: Selected solution and rationale.
  - `## Consequences`: Positive outcomes and trade-offs.
- **Execution Process**:
  - Draft the ADR immediately after finalizing the approach and propose saving the file.
  - Never modify an `Accepted` ADR. When changes occur, mark the old ADR as `Superseded by ADR XXXX` and create a new ADR.

## 11. DDD & Business Logic Conventions
- **Ubiquitous Language**: Always use terms defined in `docs/BRD.md` (Kano-Coin, Scenario, Companion, Client, Escrow).
- **Snapshot Policy**: Save a copy of parameters (price, configuration, terms) at transaction time instead of just referencing IDs.
- **Naming Standards**:
  - **Invariants**: Annotate with `[INV-XXXX]`.
  - **Commands**: `Verb + Noun` (AcceptBooking).
  - **Events**: `Noun + PastVerb` (BookingAccepted).
- **Domain Errors**: Return explicit business errors to map to HTTP/gRPC codes.
