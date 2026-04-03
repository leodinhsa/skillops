# Deploy Guide

Step-by-step process to release a new version of skillops.

## Prerequisites

- `goreleaser` installed (`brew install goreleaser`)
- `GITHUB_TOKEN` env var set (needs `repo` scope)
- `HOMEBREW_TAP_GITHUB_TOKEN` env var set (needs `repo` scope on `homebrew-skillops` repo)
- Push access to `github.com/leodinhsa/skillops`
- Push access to `github.com/leodinhsa/homebrew-skillops`

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

### 3. Commit and tag

```bash
git add -A
git commit -m "chore: release v0.2.0"
git tag v0.2.0
git push origin main --tags
```

### 4. Run goreleaser

```bash
export GITHUB_TOKEN=<your-token>
export HOMEBREW_TAP_GITHUB_TOKEN=<your-tap-token>

goreleaser release --clean
```

This will:
- Build binaries for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`
- Create a GitHub release with the binaries and `checksums.txt`
- Auto-update the Homebrew tap (`homebrew-skillops`) with the new formula

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
