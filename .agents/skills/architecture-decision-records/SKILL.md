---
name: architecture-decision-records
description: Records architecture decisions (ADRs). Use when making architectural decisions, choosing between competing approaches, or when you need to record context that future engineers and agents will need to understand the codebase.
source: https://github.com/addyosmani/agent-skills/tree/main/skills/documentation-and-adrs
---

# Architecture Decision Records (ADRs)

## Overview

Document decisions, not just code. The most valuable documentation captures the *why* — the context, constraints, and trade-offs that led to an architectural decision. This context is essential for future humans and agents working in the codebase.

## When to Write an ADR

- Choosing a framework, library, or major dependency
- Designing a data model or database schema
- Selecting an authentication strategy
- Deciding on an API architecture (REST vs. GraphQL vs. gRPC)
- Choosing between build tools, hosting platforms, or infrastructure
- Any decision that would be expensive to reverse

### Storage Location

ADRs must be stored in the following locations depending on their scope:
- **Service level (Local impact):** `services/<service-name>/docs/adr/`
- **Global level (System-wide/cross-service impact):** `docs/adr/`

### Naming Convention

Files must be named using the format `XXXX-slug-decision-name.md` (e.g., `0006-use-outbox-pattern.md`), where `XXXX` is a sequential 4-digit number starting from `0001`.

### ADR Template

Every ADR must follow this standard structure:

```markdown
# ADR XXXX: [Decision Name]

**Status:** Proposed | Accepted | Rejected | Superseded by ADR XXXX | Deprecated
**Date:** YYYY-MM-DD

## Context
Context, technical constraints, and considered alternatives.

## Decision
Selected solution and rationale.

## Consequences
Positive outcomes and trade-offs.
```

### ADR Lifecycle & Updates

1. Draft the ADR immediately after finalizing the approach and propose saving the file.
2. The lifecycle flows as follows:
   `PROPOSED` → `ACCEPTED` or `REJECTED`
3. When a decision is changed or a previous decision is retired:
   - Create a new ADR outlining the new decision.
   - You are **explicitly allowed** to directly edit/update the status of the older ADR file to `Superseded by ADR XXXX` or `Deprecated` to record the lifecycle transition.
