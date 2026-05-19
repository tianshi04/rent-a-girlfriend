---
name: protobuf-standards
description: Enforces industry-standard Protobuf conventions and design patterns. Use when drafting, reviewing, or refactoring .proto files to ensure consistent naming, robust backward compatibility, and optimal message structure based on Google and Buf guidelines.
---

# Protobuf Standards

## Overview

This skill ensures that Protobuf definitions follow industry standards for maintainability, readability, and backward compatibility. It incorporates guidelines from both Google's official documentation and Buf's style guide.

## Golden Rules

1. **Naming:** `PascalCase` for Messages/Services, `lower_snake_case` for fields.
2. **Architecture:** Use the **1-1-1 Rule** (one entity per file) and dedicated Request/Response messages.
3. **Compatibility:** NEVER reuse tags; use `reserved` for all deletions.
4. **Enums:** Start with `0` as `UNSPECIFIED` and prefix values with the Enum type name.
5. **Efficiency:** Use tags 1-15 for the most frequently used fields to save bytes.

## Guidelines Index

Detailed standards are categorized for efficient context retrieval:

- **[Naming & Syntax](references/naming.md):** Rules for packages, messages, fields, and enums.
- **[Architectural Patterns](references/architecture.md):** 1-1-1 Rule, Request/Response patterns, and service design.
- **[Evolution & Compatibility](references/evolution.md):** Backward compatibility, field management, and persistence pitfalls.

## Assets

- **[Boilerplate Template](assets/boilerplate.proto):** A comprehensive example applying all standards.
