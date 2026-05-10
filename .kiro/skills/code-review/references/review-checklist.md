# Code Review Checklist — Common

Use this checklist during every manual code review to ensure high standards
of logic, architecture, and maintainability. This file covers language-agnostic
items that apply to all stacks.

For language-specific checks, load the appropriate reference file after this one:
- Python → `lang-python.md`
- Python + FastAPI → `lang-python.md` + `lang-python-fastapi.md`
- Java → `lang-java.md`
- Java + Spring Boot → `lang-java.md` + `lang-java-spring.md`
- Golang → `lang-golang.md`

---

## 1. Logic & Correctness

- [ ] **Algorithm**: Is the algorithm correct? Does it handle edge cases
  (empty collections, null/nil/None values, zero, negative numbers)?
- [ ] **Looping**: Are there any potential infinite loops? Are exit conditions
  clearly defined?
- [ ] **Data structures**: Is the choice of collection type appropriate for
  the access pattern (e.g., a set for membership checks, not a list)?
- [ ] **Side effects**: Does the function change state in unexpected or
  undocumented ways?
- [ ] **Boundary conditions**: Are comparisons using the correct operators
  (`<` vs `<=`, `>` vs `>=`)? Off-by-one errors?

---

## 2. Architecture & Design

- [ ] **Single Responsibility**: Does each function and class do one thing?
  Would a future maintainer be able to describe its purpose in one sentence?
- [ ] **Layering**: Does the code respect architectural boundaries?
  (e.g., domain layer does not import infrastructure, business logic does
  not live in the API handler)
- [ ] **Coupling**: Is the code too tightly coupled to a specific
  implementation rather than an abstraction or interface?
- [ ] **DRY (Don't Repeat Yourself)**: Is there duplicated logic that should
  be extracted into a shared function or module?
- [ ] **Open/Closed**: Can the behavior be extended without modifying
  existing, tested code?

---

## 3. Readability & Maintainability

- [ ] **Naming**: Are variable, function, and class names descriptive and
  unambiguous? Can their purpose be understood without reading the
  implementation?
- [ ] **Complexity**: Is any function too long or cognitively complex?
  A function that requires significant mental effort to trace is a
  refactoring candidate regardless of length.
- [ ] **Comments**: Do comments explain *why*, not *what*? Code should be
  self-explanatory; comments should explain intent, constraints, or
  non-obvious decisions.
- [ ] **Documentation**: Are public interfaces documented with the
  format required by the project's style guide?
  *(See language-specific reference for the required format.)*

---

## 4. Error Handling

- [ ] **Specificity**: Are specific error or exception types caught rather
  than the broadest possible base type?
- [ ] **Fail Fast**: Is validation happening early — at the entry point of
  the function or request handler — before side effects occur?
- [ ] **Error Messages**: Are error messages helpful to the caller and
  non-leaky (they do not expose internal paths, queries, or stack traces)?
- [ ] **Re-raise or handle**: When an error is caught, is it either fully
  handled or explicitly re-raised with sufficient context? Silent swallowing
  is never acceptable.

---

## 5. Performance

- [ ] **I/O in loops**: Are there unnecessary database queries, API calls,
  or file reads inside loops (N+1 problem)?
- [ ] **Resource cleanup**: Are external resources (connections, file handles,
  streams) always released — both on success and on error?
- [ ] **Async correctness**: If the codebase uses async I/O, is blocking work
  kept off the async thread/event loop?
  *(See language-specific reference for implementation details.)*
- [ ] **Unnecessary work**: Is computation being repeated that could be
  cached or moved outside a loop?

---

## The Golden Rule

If an issue is about **formatting** or **type annotations**, it should have
been caught by the project's static analysis tools (linters and type
checkers). Do not raise findings for issues that automated tooling already
enforces — focus manual review effort on what tools cannot catch.