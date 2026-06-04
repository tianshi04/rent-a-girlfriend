# Knowledge & Rule Management

## 1. Knowledge Persistence
- Propose Updates: IF discovering undocumented project knowledge (conventions/constraints/patterns), THEN propose adding to `AGENTS.md`.
- Report Conflicts: IF `AGENTS.md` contradicts the codebase, THEN report to Developer.
- CONSTRAINT: NEVER create/modify `AGENTS.md` without explicit Developer approval.

## 2. Precedence & Nested AGENTS.md
- Precedence: Local `AGENTS.md` rules STRICTLY OVERRIDE root rules for that specific subfolder.
- Scope Isolation: NEVER add directory-specific rules to global `AGENTS.md`. Local rules MUST stay in local `AGENTS.md` to prevent global pollution.
- Monorepos: IF a subproject has unique conventions, THEN propose initializing a nested `AGENTS.md`.
