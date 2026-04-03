# Deploy Guide

Step-by-step process to release a new version of skillops.

## Prerequisites

- Push access to `github.com/leodinhsa/skillops`
- `HOMEBREW_TAP_GITHUB_TOKEN` secret set on the repo (needs `repo` scope on `homebrew-skillops`) — one-time setup
- `goreleaser` installed locally only if you need snapshot builds (`brew install goreleaser`)

## Branch policy

Always release from `main`. Tagging from a feature branch is technically possible (the workflow triggers on any `v*` tag regardless of branch), but it means shipping unreviewed code and creates a messy tag history.

Correct flow:
```
feature branch → PR → merge into main → tag on main → push tag → CI deploys
```

## Release steps

### 1. Verify the build is clean

```bash
go build -o /dev/null ./...
go test ./...
```

### 2. Bump the version constant

Edit `internal/config/config.go`:

```go
Version = "v0.2.0"  // update to new version
```

### 3. Merge to main, then commit and tag

Make sure all changes are on `main` before tagging. Never tag from a feature branch.

```bash
git checkout main
git pull origin main
git add -A
git commit -m "chore: release v0.2.0"
git tag v0.2.0
git push origin main --tags
```

### 4. GitHub Actions handles the release automatically

Pushing the tag triggers `.github/workflows/release.yml`, which runs goreleaser with:
- `GITHUB_TOKEN` — provided automatically by GitHub Actions
- `HOMEBREW_TAP_GITHUB_TOKEN` — must be set as a repo secret in `github.com/leodinhsa/skillops` → Settings → Secrets

Goreleaser will:
- Build binaries for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`
- Create a GitHub release with the binaries and `checksums.txt`
- Auto-update the Homebrew tap (`homebrew-skillops`) with the new formula

You can monitor the run at: `https://github.com/leodinhsa/skillops/actions`

### 5. Verify the release

```bash
# Check GitHub release was created
open https://github.com/leodinhsa/skillops/releases

# Verify Homebrew tap was updated
open https://github.com/leodinhsa/homebrew-skillops
```

### 6. Smoke test via Homebrew

```bash
brew update
brew upgrade skillops
skillops version
```

---

## Snapshot build (no release)

To test the full build pipeline without publishing:

```bash
goreleaser release --snapshot --clean
```

Binaries are written to `dist/`. Nothing is pushed.

---

## Troubleshooting

**`HOMEBREW_TAP_GITHUB_TOKEN` not set** — goreleaser will fail at the brew step. The binaries are still built and the GitHub release is created; you can manually update the tap formula afterward.

**Tag already exists** — delete it locally and remotely before re-running:
```bash
git tag -d v0.2.0
git push origin :refs/tags/v0.2.0
```

**Formula `sha256` mismatch** — goreleaser computes the sha256 automatically from the release tarball. Never edit it manually in `Formula/skillops.rb`; that file is overwritten by goreleaser on each release.
