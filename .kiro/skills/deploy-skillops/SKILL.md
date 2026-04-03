---
name: deploy-skillops
description: >
  Guides the full release process for the skillops CLI project: bumping the version constant,
  writing release notes, tagging, and pushing to trigger the GitHub Actions + goreleaser pipeline
  that publishes to GitHub Releases and the Homebrew tap.
  Use this skill whenever the user wants to release, deploy, publish, cut a new version, or tag
  a new release of skillops — even if they just say "let's deploy" or "time to release".
---

# Deploy skillops

This skill handles the full release flow for the `skillops` project.

## How releases work

Pushing a git tag matching `v*` triggers `.github/workflows/release.yml`, which runs goreleaser.
Goreleaser builds binaries for all platforms, creates a GitHub Release, and auto-updates the
Homebrew tap (`homebrew-skillops`). No manual goreleaser invocation is needed.

The version constant in `internal/config/config.go` (`Version = "..."`) must match the git tag
exactly. This is enforced before tagging.

---

## Release workflow

### Step 1 — Read current state

Before suggesting a tag, check:
1. The current `Version` value in `internal/config/config.go`
2. The latest git tag: `git tag --sort=-version:refname | head -5`
3. Recent commits since last tag: `git log <last-tag>..HEAD --oneline`

### Step 2 — Propose a tag and ask the user to confirm

Based on the commits since the last tag, suggest the next version following semver:
- Patch bump (`vX.Y.Z+1`) — bug fixes, small improvements, no new commands
- Minor bump (`vX.Y+1.0`) — new features, new commands, new flags
- Major bump (`vX+1.0.0`) — breaking changes, major redesign

Present the suggestion clearly and ask the user to confirm or choose a different tag.
Do NOT proceed until the user explicitly confirms the tag.

Example prompt to user:
```
Current version: v1.0.0
Latest tag: v1.0.0
Commits since last tag: 7 commits (new `update` command, bug fixes)

Suggested tag: v1.1.0 (minor bump — new feature)

Other options: v1.0.1 (patch) | v2.0.0 (major)

Which tag should we use?
```

### Step 3 — Collect release notes

Ask the user: "What changed in this release? I'll use the git log as a starting point — feel free to add, remove, or reword."

Show the git log since last tag as a draft:
```bash
git log <last-tag>..HEAD --oneline
```

Help the user shape this into clean release notes grouped by category:
- New features
- Bug fixes  
- Improvements
- Breaking changes (if any)

Do NOT proceed to step 4 until the user has approved the release notes.

### Step 4 — Check version/tag alignment

Before touching git, verify the `Version` constant matches the chosen tag.

Read `internal/config/config.go` and check the line:
```go
Version = "vX.Y.Z"
```

If it does NOT match the chosen tag:
1. Update it to match
2. Show the user the change
3. Confirm before continuing

This check is mandatory. A mismatch means the binary reports the wrong version.

### Step 5 — Ensure on main, then commit, tag, push

First verify the current branch is `main` and up to date:

```bash
git checkout main
git pull origin main
```

If the user is on a feature branch, stop and ask them to merge to main via PR first.
Never tag from a feature branch — the workflow triggers on any `v*` tag regardless of branch,
but releasing from a feature branch ships unreviewed code and creates a messy tag history.

Once on main:

```bash
git add internal/config/config.go
git commit -m "chore: release <tag>"
git tag <tag>
git push origin main --tags
```

Explain to the user: "GitHub Actions will now run goreleaser automatically. You can monitor it at:
https://github.com/leodinhsa/skillops/actions"

### Step 6 — Verify

After pushing, remind the user to verify:
```bash
# Watch the Actions run
open https://github.com/leodinhsa/skillops/actions

# Once done, check the release
open https://github.com/leodinhsa/skillops/releases

# Check the Homebrew tap was updated
open https://github.com/leodinhsa/homebrew-skillops

# Smoke test
brew update && brew upgrade skillops && skillops version
```

---

## Important rules

- Always tag from `main` — never from a feature branch
- Never tag before the user confirms the tag name
- Never push before the user approves the release notes
- Never push if `Version` in `config.go` doesn't match the tag — fix it first
- Never run `goreleaser` locally for a real release — GitHub Actions handles it
- If the user wants to test the build without releasing: `goreleaser release --snapshot --clean`

## Troubleshooting

**Tag already exists** — delete and recreate:
```bash
git tag -d <tag>
git push origin :refs/tags/<tag>
```

**`HOMEBREW_TAP_GITHUB_TOKEN` not set as repo secret** — goreleaser will fail at the brew step.
Binaries and GitHub Release are still created. Fix: add the secret at
`github.com/leodinhsa/skillops` → Settings → Secrets → `HOMEBREW_TAP_GITHUB_TOKEN`
(needs `repo` scope on the `homebrew-skillops` repo).

**Formula sha256 mismatch** — never edit `Formula/skillops.rb` manually. Goreleaser overwrites it.
