---
trigger: always_on
---

- **Docstring vs. Comment**: 
    - **Docstring**: Interface contract explaining WHAT a public component (API, class, function, message) does and HOW to use it. Must document parameters, return values, business exceptions (`Raises`), and business constraints (`Invariants`). Write in concise imperative mood.
    - **Comment**: Implementation notes explaining WHY complex or non-obvious logic was written. Use only for domain specifics, complex algorithms, or workarounds. Never explain basic language syntax.
- **No Redundancy (DRY)**: 
    - **No Type Duplication**: Never specify types in docstrings if the language supports Type Hints or signatures. Only explain business meaning or units of measurement (e.g., currency, time units like milliseconds/seconds).
    - **Self-Documenting Code**: Prioritize expressive naming. Do not write docstrings that redundantly repeat clear function or variable names.
    - **No Internal Implementation Details**: Docstrings must not describe internal execution steps. Use inline comments next to complex code blocks instead.
- **Synchronization**: Always update docstrings and comments in the same commit when changing function signatures or implementation logic. Code reviews must verify documentation accuracy similarly to source code.
