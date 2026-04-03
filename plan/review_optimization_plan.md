# Proposed Improvements for SkillOps

This document outlines recommended fixes and optimizations for the `skillops` project. 
> [!IMPORTANT]
> Per user instructions, these changes are proposed for review and will **NOT** be implemented directly in the codebase at this time to maintain stability.

## User Review Required

> [!WARNING]
> **Lack of Automated Tests**: The project currently has zero unit tests. It is highly recommended to implement tests for core logic before production release.

## Proposed Changes

### 1. Build & Optimization (Binary Size)
Summary: Reduce binary size by stripping debug symbols and optimizing the build process.

#### [MODIFY] .goreleaser.yaml
- Add `-ldflags="-s -w"` to the `builds` section.
- This will reduce binary size from **8.1MB** to approximately **5.6MB** (~30% reduction).

### 2. Clipboard Support (Cross-Platform)
Summary: Replace the hardcoded `pbcopy` call with a cross-platform library to support Linux and other environments.

#### [MODIFY] internal/tui/list.go
- Use `github.com/atotto/clipboard` for the copy-to-clipboard functionality instead of calling `exec.Command("pbcopy")`.
- This ensures the "Copy Skill Name" feature works everywhere `skillops` is supported.

### 3. TUI UX Improvement (Error Reporting)
Summary: Enhance TUI stability by capturing and displaying errors within the UI instead of printing to stderr.

#### [MODIFY] internal/tui/tui.go
- Move error messages to a model field that can be rendered within the TUI `View`. This prevents UI flickering and ensures a clean terminal state.

### 4. Testing Strategy [NEW]
Summary: Establish a testing baseline.

#### [NEW] internal/symlink/symlink_test.go
- Implement unit tests for `EnsureSymlink`, `RemoveSymlink`, and `IsSymlinkEnabled` using temporary directories.

#### [NEW] internal/skills/skills_test.go
- Test skill discovery logic with various folder structures (Root, Subfolder, Container rules).

## Verification Plan

### Automated Tests
- Run `go test ./...` after implementing the proposed tests.

### Manual Verification
- Verify binary size reduction using `ls -lh`.
- Verify TUI rendering on various terminal sizes.
- Test clipboard functionality on different OS (Linux/macOS).
