# Tech Stack

## Language & Runtime
- Go 1.25+
- Module name: `skillops`

## Key Libraries
- `github.com/spf13/cobra` — CLI framework, command/subcommand structure
- `github.com/charmbracelet/bubbletea` — TUI framework (Elm-style model/update/view)
- `github.com/charmbracelet/lipgloss` — Terminal styling and layout
- `github.com/charmbracelet/bubbles` — Reusable TUI components (lists, inputs, etc.)
- `gopkg.in/yaml.v3` — Config file serialization (global config only)

## Common Commands

```bash
# Build
go build -o skillops .

# Run directly
go run main.go <command>

# Run tests
go test ./...

# Run specific package tests
go test ./internal/skills/...

# Tidy dependencies
go mod tidy

# Snapshot build (test pipeline locally, no publish)
goreleaser release --snapshot --clean

# Release — do NOT run locally, push a tag instead:
# git tag vX.Y.Z && git push origin main --tags
# GitHub Actions handles the rest (see DEPLOY.md)
```

## TUI Pattern (bubbletea)
All interactive TUIs follow the bubbletea `Model` interface:
- `Init() tea.Cmd`
- `Update(tea.Msg) (tea.Model, tea.Cmd)`
- `View() string`

### Clean Exit Rule (critical)
TUI models must have a `quitting bool` field. When quitting:
- Set `m.quitting = true` before returning `tea.Quit`
- In `View()`, return `""` if `m.quitting` is true
- Print final output via `fmt.Println` *after* `p.Run()` returns in the command entry point

This prevents ghost borders/artifacts in the terminal.

### TTY Detection
When TUI interactions are required (conflict resolution, disambiguation):
- Check if running in TTY environment: `term.IsTerminal(int(os.Stdin.Fd()))`
- **TTY**: Launch interactive TUI
- **Non-TTY** (CI, SSH, piped): Fail with descriptive error listing conflicts and suggesting manual config.json edit
- Never silently fail or make assumptions in non-TTY environments

## Config Files

### Global Config (`~/.skillops/config/agentics.yaml`)
- Format: YAML
- Purpose: Maps IDE names to their skill directory paths
- Auto-initialized on every command via `PersistentPreRun` in root command
- Versioned with `config_version` field for migration tracking

### Local Config (`.skillops/config.json`)
- Format: JSON (human-readable, indented)
- Purpose: Project-specific skill configuration (source of truth)
- **Version**: Must be "2" (v1 not supported)
- **Commit to git**: Team members share this file
- Contains: skill identities, registries, custom symlink names

### Global Store (`~/.skillops/skills/`)
- Organized by full identity path: `<host>/<full-path-to-skill>`
- Contains pulled skill repositories and individual skills
- Metadata files: `.so-skill-meta.json` (per skill), `.so-repo-meta.json` (per repo)
- Repo boundary is NOT encoded in filesystem — determined by registry matching

## Skill Identity Format

**Full-path format**: `<host>/<repo-path>/<path-to-skill>`

Examples:
- `github.com/anthropics/skills/skills/logger`
- `gitlab.com/devops-team/ci-helpers/docker-builder`
- `github.com/company/monorepo/backend/services/api/skills/auth` (nested)
- `gitlab.common.datumhq.com/datumhq-consulting-vn/management/datum-skills/software-skills/skills/logger` (multi-level groups)

**Key design decision**: The identity does NOT encode where the repo ends and path-to-skill begins. This boundary is determined by **registry URL prefix matching**.

**Components**:
- **Host**: Git hosting platform (e.g., `github.com`, `gitlab.com`, `gitlab.company.internal`)
- **Short name**: Final component used for symlink (e.g., `logger`)

**Validation rules**:
- Minimum 3 path components (host/something/skill)
- No empty components
- No "." or ".." components (path traversal prevention)
- Always validate with `ParseIdentity` before filesystem operations

## Data Structures

### ParsedIdentity
```go
type ParsedIdentity struct {
    Full        string   // gitlab.common.datumhq.com/datumhq-consulting-vn/management/datum-skills/software-skills/skills/logger
    Host        string   // gitlab.common.datumhq.com
    Path        string   // datumhq-consulting-vn/management/datum-skills/software-skills/skills/logger (everything after host)
    ShortName   string   // logger (final component, used for symlink)
}
```

**Note**: There is no `Owner` or `Repo` field. The boundary between repo-path and path-to-skill is determined by registry matching at runtime, not by parsing.

### Registry
```go
type Registry struct {
    URL      string   // https://github.com/anthropics/skills (full repo clone URL, no trailing slash)
    Name     string   // Anthropic Public Skills
    Priority int      // Lower number = higher priority
}
```

### SkillMetadata (`.so-skill-meta.json`)
```json
{
  "repo_url": "https://github.com/anthropics/skills",
  "path_in_repo": "skills/logger",
  "pulled_at": "2026-05-06T10:30:00Z",
  "commit_hash": "abc123def456"
}
```

### RepoMetadata (`.so-repo-meta.json`)
```json
{
  "repo_url": "https://github.com/anthropics/skills",
  "pulled_at": "2026-05-06T10:30:00Z",
  "commit_hash": "abc123def456"
}
```

## Path Safety Rules

**Critical**: Always validate paths before filesystem operations

1. **Never** `os.RemoveAll` on root directories (`/`, `~`, cwd)
2. **Always** validate paths are within `<cwd>/<toolRootDir>/skills/` before removal
3. **Always** validate identity components (no empty, ".", "..", path traversal)
4. **Always** use `utils.ValidateName` before constructing file paths
5. **Always** use `ParseIdentity` for validation before any filesystem operations

## Symlink Structure

- **Global store**: Nested structure matching full identity path (`~/.skillops/skills/<host>/<full-path>`)
- **Project symlinks**: Flat structure in IDE directories (`.kiro/skills/logger`)
- **Symlink names**: Use short name (default) or custom name from `config.symlink_names`
- **Conflict resolution**: When multiple skills have same short name, require custom names

## Registry Matching

- Registry URL is a full repo clone URL (e.g., `https://github.com/anthropics/skills`)
- Normalize URL: strip protocol (`https://`, `git@`), replace `:` with `/` for SSH → get prefix
- Match: skill identity starts with normalized registry prefix
- Path in repo = identity minus matched prefix (strip leading `/`)
- Sort by priority (lower number = higher priority)
- Auto-populate registries when adding skills (read from skill metadata)
- Sync uses registries to auto-pull missing skills
- No fallback to metadata when registry matching fails (explicit error)

## Error Handling Patterns

### Descriptive Errors
- Always include full skill identity in error messages
- Suggest recovery actions (e.g., "add registry to config")
- List all conflicts when multiple issues exist

### Non-TTY Errors
- Detect non-TTY environment before launching TUI
- Provide actionable error messages with manual resolution steps
- Show example config.json snippets for manual fixes

### Validation Errors
- Validate early (before filesystem operations)
- Clear error messages for invalid identities
- Prevent partial state (no partial symlinks or corrupted config)

## Testing Conventions

- Unit tests: `*_test.go` files alongside implementation
- Test file naming: `<package>_test.go`
- Integration tests: `cmd/*_test.go`
- TUI testing: Manual testing (bubbletea models not easily unit-testable)
- Test coverage: Focus on parsing, validation, and data flow logic

## Development Workflow

1. **Read spec first**: Check requirements.md and tasks.md before implementing
2. **Validate inputs**: Use ParseIdentity and path validators before filesystem ops
3. **Handle TTY/non-TTY**: Always check environment before launching TUI
4. **Test edge cases**: Nested paths, multi-level groups, conflicts, missing metadata
5. **Update docs**: Keep steering files and README in sync with implementation
