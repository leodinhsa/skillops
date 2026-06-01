---
name: code-review
description: >-
  Structured approach for reviewing code for logic, maintainability, performance,
  and security across Java, Python, and Golang. Use when Claude is asked to:
  (1) Review a Pull Request (PR), (2) Audit a file or module for bugs,
  (3) Optimize or refactor existing code, (4) Ensure adherence to Clean
  Architecture and SOLID principles, (5) Perform a security audit.
  Trigger on phrases like "review this", "check this code", "any issues?",
  "what do you think of this?", "is this following best practices?",
  "before I commit", or when code is pasted asking for feedback.
  Do NOT trigger for: writing new code from scratch, explaining concepts,
  running or executing code, or tasks already handled by linting tools.
  This skill handles high-level architectural and logic reviews,
  complementing automated static analysis tools.
compatibility: >-
  Works with Claude Code, Cursor, GitHub Copilot, Antigravity, OpenAI Codex,
  and any agent implementing the Agent Skills open standard.
  No runtime dependencies required.
metadata:
  version: "1.0"
#allowed-tools: Read, Grep, Glob ### Enabled for Claude Code
#model: sonnet ### Enabled for Claude Code
---

# Code Review
 
This skill defines the standards and procedures for conducting high-quality
code reviews. It transforms the agent into an expert reviewer focused on
architectural integrity, logical correctness, and security — across Java,
Python, and Golang.
 
---
 
## Mandatory Triggers
 
You MUST use this skill whenever the user:
 
- Points to a specific line or block of code for "improvement",
  "optimization", or "refactoring"
- Questions the "correctness" or "readability" of a file
- Asks "What do you think of this code?" or "Is this following best practices?"
- Submits a Pull Request, a diff, or a file for feedback
- Asks for a security audit or vulnerability check on existing code
 
---
 
## Core Principles
 
- **Focus on High-Level Intent**: Look past the syntax (which should be caught
  by static analysis tools) to the *reasoning* and *structure*
- **Constructive Feedback**: Provide actionable suggestions with clear
  rationales — every finding must have a concrete suggestion
- **Security First**: Always assume edge cases and malicious inputs;
  security findings are never suggestions — always P1 if exploitable
- **Review for the future maintainer**: The author knows the context today;
  the maintainer two years from now does not
- **Distinguish preference from problem**: P1/P2 are objective issues that
  will cause failures; P3 is where judgment and style live — never dress up
  a preference as a correctness issue
 
---
 
## How to Conduct a Review
 
Load all four references before starting any review — they are always required:
 
1. **Preparation**: Follow the [Review Workflow](references/review-workflow.md)
   to ensure a consistent, repeatable approach
2. **Standards Check**:
   - Use the [Review Checklist](references/review-checklist.md) for logic,
     architecture, and maintainability
   - Apply the [Security Review Guidelines](references/security-review.md)
     for critical safety checks — run this on every review, not just when
     security is explicitly requested
3. **Communication**: Load [Feedback Template](assets/feedback-template.md)
   for severity definitions, report structure, comment format, and output standards
 
Do not proceed with any review before all four references are loaded.
 
---
 
## Language and Framework Routing
 
After loading the four core references above, you MUST identify the language
and framework from the code being reviewed, then load the matching reference
file. Loading the language reference is not optional — it contains the
language-specific checklist required to complete the review.
 
```
Detected: Python
  → MUST load: references/lang-python.md
  → MUST load: references/lang-python-feedback-examples.md
 
Detected: Python + FastAPI
  → MUST load: references/lang-python.md
  → MUST load: references/lang-python-fastapi.md
  → MUST load: references/lang-python-feedback-examples.md
 
Detected: Python + Flask
  → MUST load: references/lang-python.md
  → MUST load: references/lang-python-flask.md
  → MUST load: references/lang-python-feedback-examples.md
 
Detected: Java
  → MUST load: references/lang-java.md
 
Detected: Java + Spring Boot
  → MUST load: references/lang-java.md
  → MUST load: references/lang-java-springboot.md
 
Detected: Golang
  → MUST load: references/lang-golang.md
 
Detected: Multiple languages
  → MUST load all matching language files above
```
 
**If language or framework cannot be detected, or no reference file exists
for the detected language/framework: STOP. Ask the user to confirm the
language and framework before proceeding. Do not begin the review until
the correct reference file is confirmed and loaded.**
 
---
 
## Review Modes
 
### Full Review (default)
Used when: new feature implementation, PR from another developer,
pre-merge gate, or user requests a thorough review.
 
Runs all tiers in order. Produces a complete Review Report as defined
in [Feedback Template](references/feedback-template.md).
 
### Fast Review
Used when: user says "quick", "fast review", "just check for blockers",
"hotfix review", or context makes urgency clear.
 
Runs Tier 1 (Correctness) and Tier 2 (Security) only.
Produces a shortened report — verdict and P1 findings only, no suggestions.
 
---
 
## Review Tiers
 
The checklist detail for each tier lives in the reference files above.
The tiers below define scope and severity — use them to classify every finding.
 
### Tier 1 — Correctness `[P1: BLOCKING]`
Will this code do what it is supposed to do in all cases?
Logic errors, missing null/error checks, wrong conditions, incorrect state
transitions, unhandled edge cases.
 
### Tier 2 — Security `[P1 if exploitable · P2 if theoretical]`
Can this code be exploited or does it expose what it should not?
Hardcoded credentials, injection risks, missing input validation,
auth checks in the wrong layer, sensitive data in logs.
 
### Tier 3 — Reliability `[P2: SHOULD FIX]`
Will this code hold up in production over time?
Resource leaks, silent failures, race conditions, dependency failures
not handled, timeouts missing on external calls.
 
### Tier 4 — Maintainability `[P3: SUGGESTION]`
Will this code be easy to understand and change in the future?
Functions doing too much, magic numbers, duplicate logic, names that
require reading the implementation to understand intent.
 
### Tier 5 — Performance `[P2 if user-facing · P3 otherwise]`
Will this code perform acceptably under expected load?
N+1 queries, unnecessary computation in loops, missing caching
opportunities, avoidable memory allocations.
 
### Tier 6 — Style & Conventions `[P4: INFO]`
Does the code follow the project's coding standards?
Check the project context file (see Integration below) for
project-specific conventions. Only flag what automated tools do not
already catch. Never block a merge for P4 alone.
 
---
 
## Integration with Standards
 
This skill is designed to work alongside other tooling and configuration
in the project:
 
- **Automated static analysis**: Linters, type checkers, and security
  scanners should pass before manual review begins. This skill focuses
  on what those tools miss: design patterns, business logic correctness,
  architectural decisions, and subtle concurrency issues.
 
- **Project context file**: Every Agentic IDE provides a way to define
  project-specific conventions, known exceptions, and team standards.
  Always check this file before starting a review — its contents override
  the general best practices in this skill.
  > **Claude Code**: project context file is `CLAUDE.md` in the project root.
  > **Cursor**: project context file is `.cursor/rules/` or `.cursorrules`.
  > **GitHub Copilot**: project context file is `.github/copilot-instructions.md`.
  > **Other IDEs**: refer to your IDE documentation for the equivalent file.
 
- **Implementation Plan**: Use this skill after implementation is complete
  as a quality gate before merge. Document final architectural decisions
  and any P1/P2 findings that were resolved during review.
 
- **Manual review focus**: Design patterns, business logic correctness,
  subtle race conditions, and architectural coherence — the layer above
  what automated tooling can catch.
 
---
 
## Hard Rules
 
- Never modify a file — read only; if you see an obvious fix, describe it,
  do not apply it
- Never approve when P1 issues remain unaddressed
- Never write a finding without a concrete suggestion
- Never flag issues that the project's static analysis tools already enforce
- Never mark something P1 because it is stylistically displeasing —
  P1 means it will break or be exploited
- Never proceed with a review if the language reference file has not been
  loaded — return to Language and Framework Routing and resolve first
 