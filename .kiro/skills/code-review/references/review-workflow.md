# Review Workflow: Step-by-Step Guide

Follow this sequential process to perform effective and time-efficient code reviews.

---

## Step 1: Context Gathering

1. **Read the PR Description**: Understand the "Why" and the "How."
2. **Run Automated Checks**: Ensure CI (linters, type checkers, test suite)
   passes. **Never** review code that fails automated checks — those failures
   must be resolved first.
3. **Review the Implementation Plan**: Compare the PR against the approved plan.

---

## Step 2: High-Level Architecture

1. Check if the changes belong to the right layers (e.g., no database logic
   in the domain layer, no business logic in the infrastructure layer).
2. Verify whether the new functionality breaks existing patterns or introduces
   excessive technical debt.

---

## Step 3: Logic & Detailed Review

1. Verify the core logic of new functions.
2. Walk through the "Happy Path" and all significant "Error Paths."
3. Apply the [Review Checklist](review-checklist.md).

---

## Step 4: Security Inspection

1. Check for authorization gaps and missing input validation.
2. Apply the [Security Review Guidelines](security-review.md) — run on every
   review, not only when security is explicitly requested.

---

## Step 5: Providing Feedback

1. Leave clear, actionable comments with current code and suggested fix.
2. Categorize every comment using the severity labels defined in
   [Feedback Template](feedback-template.md):
   - `[P1-Blocker]` — must fix before merge
   - `[P2-Should Fix]` — fix in this PR or create a ticket immediately
   - `[P3-Suggestion]` — encouraged, author's discretion
   - `[P4-Nit]` — minor, never blocks merge
   - `[Question]` — clarification needed before raising a finding
3. Provide concrete code examples for every finding (see
   [Feedback Template](feedback-template.md) for comment structure).

---

## Step 6: Final Decision

- **Approve**: Code is ready for merge. Proceed to Step 7 if post-approval
  documentation is required.
- **Request Changes**: Blockers (`[P1-Blocker]`) or significant logic errors
  found — author must address before re-review.
- **Comment**: Only minor suggestions or nits; merge can proceed at
  author's discretion.

---

## Step 7: Post-Approval Documentation *(Optional — project-specific)*

This step applies only if your project uses an Implementation Plan workflow.
Skip if not applicable.

1. **Create Detailed Implementation Plan**: Use the Implementation Plan skill
   (if available in your IDE) to document final architectural decisions and
   review outcomes.
2. **Save the Plan**: Write the plan to the location defined by your project
   conventions (e.g., `docs/implementation-plan/YYYY-MM-DD-plan.md`).

> This step is optional and depends on your project's documentation workflow.
> It is not required for the review itself to be complete.