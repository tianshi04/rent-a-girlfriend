# Protobuf Naming Conventions

Strict adherence to these naming standards ensures cross-language compatibility and readability.

## 1. Packages
- **Format:** `lower_snake_case`.
- **Versioning:** Always include a version suffix (e.g., `v1`, `v2beta1`).
- **Example:** `package my.service.v1;`
- **Pathing:** Directory structure MUST match the package name.

## 2. Messages
- **Format:** `PascalCase`.
- **Example:** `message CreateUserRequest { ... }`

## 3. Fields
- **Format:** `lower_snake_case`.
- **Repeated Fields:** Use plural names.
- **Example:** `repeated string email_addresses = 1;`
- **Avoid:** Language keywords (e.g., `internal`, `for`, `from`, `class`).

## 4. Enums
- **Type Name:** `PascalCase`.
- **Value Name:** `UPPER_SNAKE_CASE`.
- **Prefixing:** MUST prefix values with the Enum type name to prevent collisions in languages like C++.
- **Zero Value:** MUST start with `0` as an `UNSPECIFIED` default.
- **Example:**
  ```proto
  enum Corpus {
    CORPUS_UNSPECIFIED = 0;
    CORPUS_UNIVERSAL = 1;
    CORPUS_WEB = 2;
  }
  ```

## 5. Services & RPCs
- **Service Name:** `PascalCase` with `Service` suffix (e.g., `UserService`).
- **RPC Name:** `PascalCase` (e.g., `GetUser`).
