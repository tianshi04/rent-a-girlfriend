# Protobuf Architectural Patterns

Guidelines for designing scalable and modular Protobuf APIs.

## 1. The 1-1-1 Rule
- **One** top-level entity (Message, Enum, or Service) per **One** `.proto` file, corresponding to **One** build target.
- **Benefit:** Minimizes transitive dependencies and build times.

## 2. Request/Response Pattern
- **Rule:** Every RPC MUST have its own dedicated Request and Response message.
- **Avoid:** Sharing messages across RPCs, as their requirements often diverge over time.
- **Avoid:** `google.protobuf.Empty`. Use a custom empty message (e.g., `message DeleteUserResponse {}`) to allow for future backward-compatible additions.

## 3. Service Design
- Services should be focused on a single responsibility.
- Favor specific RPCs over "god-like" generic methods.

## 4. Data Modeling
- **Booleans vs. Enums:** Prefer Enums for fields that could eventually have more than two states (e.g., `Status` instead of `is_active`).
- **Storage vs. API:** Maintain separate message definitions for public APIs and internal long-term storage (databases).

## 5. Well-Known Types
- Use `google.protobuf.Timestamp`, `google.protobuf.Duration`, and `google.protobuf.FieldMask` where appropriate instead of custom definitions.
- Avoid `google.protobuf.Any`; prefer explicit `oneof` or extensions for type safety.
