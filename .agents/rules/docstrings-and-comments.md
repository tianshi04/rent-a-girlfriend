---
trigger: always_on
---

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
