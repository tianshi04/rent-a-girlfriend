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
- **Contracts as SSOT**: All communications (gRPC, Async Events) must be defined in the `/contracts` directory at the root (the Single Source of Truth). Redefining messages/events in individual services is strictly forbidden; services must generate code from these proto files.
- **Directory Structure**:
  - `cmd/server/`: Entry point.
  - `internal/domain/`: Aggregate, VO, Repository port, Events.
  - `internal/application/`: Commands, Queries, Saga.
  - `internal/infrastructure/`: DB/Broker/Client adapters.
  - `internal/interfaces/`: HTTP/gRPC/PubSub handlers.
  - `deployments/`: K8s manifests and Istio policies specific to the service.
  - `gen/`: Generated code from Protobuf/AsyncAPI.
  - `docs/`: Technical documentation specific to the service.
  - `tests/`: Integration & E2E tests.
- **Principles**:
  - Dependency Direction: Outside-in (Domain is the core).
  - Database Isolation: No cross-service database access.
  - Service Autonomy: Each service must contain everything needed to run as an independent repository and deploy standalone (Independent Repo Ready).
- **Deployment**:
  - Standalone Dockerfile for each service at its root. Multi-stage build.
  - Configure via environment variables (.env.example). No hard-coded values.
  - **Port Conventions**: All Microservices must use standard ports: HTTP runs on `8080` (`SERVER_PORT`), gRPC runs on `50051` (`GRPC_PORT`).
- **Quality & Documentation**:
  - **Self-Documenting**: Each service must manage its own `docs/` directory. Update documentation immediately upon business logic changes.
  - **Test-Driven**: Every new feature/logic must have Unit Tests (for Domain/Application) and Integration Tests (for Infrastructure).

## 6. Architecture Decision Records (ADR)
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

## 7. DDD & Business Logic Conventions
- **Ubiquitous Language**: Always use terms defined in `docs/BRD.md` (Kano-Coin, Scenario, Companion, Client, Escrow).
- **Snapshot Policy**: Save a copy of parameters (price, configuration, terms) at transaction time instead of just referencing IDs.
- **Naming Standards**:
  - **Invariants**: Annotate with `[INV-XXXX]`.
  - **Commands**: `Verb + Noun` (AcceptBooking).
  - **Events**: `Noun + PastVerb` (BookingAccepted).
- **Domain Errors**: Return explicit business errors to map to HTTP/gRPC codes.

## 8. Distributed Communication Patterns
- **Service Mesh**: Istio Ambient Mode (Sidecar-less).
  - **L4 (ztunnel)**: Handles mTLS and Service Identity (SPIFFE).
  - **L7 (Waypoint)**: Handles JWT Verification, Routing, and Traffic Policies.
- **Auth Offloading & Identity Propagation**: 
  - NEVER implement JWT verification logic within microservice code. Verification is offloaded to Istio Waypoint.
  - **Header Injection**: After verification, Istio injects JWT claim information into headers for application consumption:
    - `user-id` (from `sub`)
    - `user-email` (from `email`)
    - `user-role` (from `role`)
    - `user-status` (from `status`)
- **Reliable Messaging**: Transactional Outbox when sending events.
- **Safe Consumption**: Check idempotency using `eventId`.
- **Contract Standards**: 
  - **Synchronous**: gRPC for commands, REST for queries.
  - **Asynchronous**: CloudEvents JSON format.
    - **Event Type Naming Rule**: `<domain>.<event-name>.v<version>` (where `<domain>` is lowercase, `<event-name>` is in `kebab-case`, e.g., `booking.booking-accepted.v1`).
    - **Topic Naming Rule**: `<domain>.events` (all lowercase, e.g., `booking.events`).
    - **Casing Standard**: All JSON fields (inside CloudEvents `data` payload and REST APIs) must strictly use **`camelCase`** to align with the Google Protobuf JSON Mapping specification.

