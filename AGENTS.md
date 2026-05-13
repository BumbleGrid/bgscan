# Agent instructions

1. **Comments** — Do not add comments unless the user explicitly asks for them. The only exception is documenting methods on an `interface` type (contract / behavior for implementers and callers).

2. **Compile-time interface checks** — Do not add assertions such as `var _ SomeInterface = (*Concrete)(nil)`. Rely on normal assignment, tests, or the compiler at use sites instead.

3. **Names** — Avoid single-letter identifiers (`i`, `n`, `c`, `s`, …), except for receiver variables. Prefer short but readable names that state role or content (`idx`, `count`, `client`, `spec`, …).
