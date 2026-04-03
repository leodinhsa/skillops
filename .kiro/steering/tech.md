# Tech Stack

## Language & Runtime
- Go 1.25+
- Module name: `skillops`

## Key Libraries
- `github.com/spf13/cobra` — CLI framework, command/subcommand structure
- `github.com/charmbracelet/bubbletea` — TUI framework (Elm-style model/update/view)
- `github.com/charmbracelet/lipgloss` — Terminal styling and layout
- `github.com/charmbracelet/bubbles` — Reusable TUI components (lists, inputs, etc.)
- `gopkg.in/yaml.v3` — Config file serialization

## Common Commands

```bash
# Build
go build -o skillops .

# Run directly
go run main.go <command>

# Run tests
go test ./...

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

## Config
- Global config: `~/.skillops/config/agentics.yaml`
- Skills directory: `~/.skillops/skills/`
- Config is auto-initialized on every command via `PersistentPreRun` in root command
