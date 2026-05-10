# Feedback Template

This file defines the severity system, report structure, and comment standards
for all code reviews. Load this file before writing any review output.

---

## Severity Labels

Every finding MUST be labeled. Use exactly these labels — no variations.

| Label | Meaning | Action required |
|-------|---------|-----------------|
| `[P1-Blocker]` | Will cause failure, crash, data loss, or security breach | Must be fixed before merge. No exceptions. |
| `[P2-Should Fix]` | Will cause problems under stress, edge cases, or over time | Fix in this PR, or create a ticket immediately if deferred |
| `[P3-Suggestion]` | Improves clarity, maintainability, or performance | Encouraged but not mandatory. Author's discretion. |
| `[P4-Nit]` | Minor style or naming preference | Never blocks merge. Include sparingly. |
| `[Question]` | Clarification needed on intent or design | Not an issue — ask before assuming a problem exists |

**Rules for labeling:**
- A finding without a label is incomplete — always assign one
- Never label something `[P1-Blocker]` because of style preference — P1 means it will break or be exploited
- `[Question]` is not a severity — use it when intent is genuinely unclear before raising a finding
- When in doubt between P2 and P3, ask: will this cause a real problem in production? Yes → P2. No → P3.

---

## Comment Structure

Every finding — regardless of severity — MUST follow this structure:

```
[LABEL] Short title (noun phrase, not a sentence)
Location: file/path:line_number

Current code:
  [paste the exact problematic code — do not paraphrase]

Problem:
  [one to two sentences explaining what is wrong and why it matters]

Suggested fix:
  [paste the corrected or improved code — be specific]

Reason:
  [one sentence explaining why the suggestion is better]
```

**Why current code AND suggested fix are both required:**
- Current code: anchors the finding to a specific, unambiguous location — reviewer and author are looking at the same thing
- Suggested fix: makes the comment immediately actionable — author should be able to apply it without guessing what you meant
- A comment without suggested fix is an observation, not a review comment

**For [Question]:**
```
[Question] Short title
Location: file/path:line_number

Current code:
  [the code in question]

Question:
  [what you need clarified before you can assess this as a finding or not]
```

**For [P4-Nit] — abbreviated form allowed:**
```
[P4-Nit] Short title
Location: file/path:line_number
Suggestion: [one line — no need for current/suggested code blocks if trivial]
```

---

## Full Review Report Template

Use this template for every Full Review. Do not omit sections — write
"None found" if a section has no content.

```
CODE REVIEW — [filename, PR title, or module name]
Mode: FULL
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

VERDICT: [APPROVED | APPROVED WITH COMMENTS | CHANGES REQUIRED]
Blocking issues (P1): [N]
Should-fix issues (P2): [N]

SUMMARY
[2–3 sentences. Overall quality assessment — what is working well and
what is the primary concern. Write this as if explaining to someone who
has not read the code.]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
FINDINGS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[Apply comment structure above for each finding.
Order findings by severity: P1 first, then P2, P3, P4, Questions last.]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
POSITIVES
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[1–3 things done well. This section is not optional — if the code has
no positives worth noting, that itself signals a deeper problem worth
raising in the summary. Skipping positives signals a poor review
process, not a poor codebase.]
```

---

## Fast Review Report Template

Use this template when Fast Review mode is triggered.

```
CODE REVIEW — [filename or PR title]
Mode: FAST (Tier 1 Correctness + Tier 2 Security only)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

VERDICT: [APPROVED | CHANGES REQUIRED]
Blocking issues (P1): [N]

[P1 findings only, using comment structure above]
[If none: "No blocking issues found in Tier 1 and Tier 2."]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

---

## Tone and Communication Rules

- **Objective over subjective**: The subject is the code, not the author.
  Write "this method catches `Exception` broadly" not "you caught Exception broadly"
- **Specific over vague**: Name the exact file, line, and problem.
  "The catch block at line 34 swallows the exception silently" not "error handling could be better"
- **Constructive for suggestions**: P3 and P4 comments should feel like
  professional advice from a colleague, not criticism
- **Questions before assumptions**: If intent is unclear, use `[Question]`
  rather than assuming the code is wrong