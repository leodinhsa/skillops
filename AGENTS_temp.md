# SkillOps: Project Memory & Development Guidelines

This document is the primary knowledge base for the `skillops` project. It captures architecture, design philosophy, and critical development rules to ensure consistency and prevent regressions.

---

## Project Overview

`skillops` is a CLI tool for managing AI agent "skills" (modular capabilities/scripts) across multiple Agentic IDEs (Claude Code, Cursor, Windsurf, Kiro, etc.).

- **Core problem**: Each IDE stores skills in a different directory. Syncing a shared skill repo across IDEs manually is tedious and error-prone.
- **Solution**: A central global store (`~/.skillops/skills/`) holds all pulled skill repos. Symlinks map individual skills into the correct IDE-specific paths within a project. A local `.skillops/config.json` tracks which IDEs and skills are active per project.

---

## Architecture

### Directory layout

```
~/.skillops/
  config/
    agentics.yaml      ← global IDE registry (name → relative path)
    settings.yaml      ← registry sources for auto-pull
  skills/
    <repo-name>/
      <skill-name>/
        SKILL.md       ← presence of this file marks a valid skill

<project>/
  .skillops/
    config.json        ← local project config (source of truth, commit to git)
  .<ide>/
    skills/
      <skill-name> → ~/.skillops/skills/<repo>/<skill>   (symlink)
```

### Source files

```
cmd/                   ← Cobra commands (one file per command)
  root.go              ← root command, PersistentPreRun config init, custom help
  init.go              ← skillops init
  add.go               ← skillops add
  remove.go            ← skillops remove
  status.go            ← skillops status
  sync.go              ← skillops sync
  pull.go              ← skillops pull
  list.go              ← skillops list
  update.go            ← skillops update
  config.go            ← skillops config subcommands
  version.go           ← skillops version

internal/
  config/
    config.go          ← global config R/W, defaultAgentics, EnsureConfig, migration
    localconfig.go     ← local project config R/W (.skillops/config.json)
    settings.go        ← registry settings R/W (settings.yaml)
  git/
    git.go             ← Clone, pull, URL normalization
  skills/
    skills.go          ← skill discovery (SKILL.md detection), metadata R/W
    extract.go         ← PullSkillFromURL (shared by pull --skill and sync auto-pull)
  symlink/
    symlink.go         ← create/remove/check symlinks, find linked agentics
  tui/
    styles.go          ← shared lipgloss styles and color palette
    tui.go             ← main interactive TUI (init checklist, checklistModel)
    add.go             ← add TUI (skill select → tool select → confirm)
    remove.go          ← remove TUI (skill select → tool select → confirm)
    list.go            ← list TUI view
    init.go            ← init TUI entry point
  utils/
    utils.go           ← shared helpers (ValidateName, CopyDir, etc.)
```

### Data flow

```
Global store (~/.skillops/skills/)
  └── pulled by: skillops pull

Local config (.skillops/config.json)           ← source of truth
  └── managed by: init / add / remove

Project symlinks (derived state)
  └── created by: add / sync
  └── removed by: remove / init (deselect)
```

### Local config schema

```json
{
  "version": "1",
  "tools": {
    "claude-code": ["repo-a/auth-agent", "repo-a/logging-agent"],
    "kiro": ["repo-a/auth-agent"]
  }
}
```

Skills are stored as `"repo/skill"` full identity. The short name (symlink filename) is derived at runtime as the portion after `/`.

---

## TUI Design System

All TUIs follow the bubbletea `Model` interface (`Init`, `Update`, `View`) and use styles from `internal/tui/styles.go`.

### Color palette

| Role | Hex | Usage |
|---|---|---|
| Primary (Pink) | `#F92672` | Titles, focus accents |
| Secondary (Purple) | `#AE81FF` | Borders, selection backgrounds |
| Success (Green) | `#A6E22E` | Success text, emojis |
| Muted | `#6272A4` | Help text, secondary info |

### Clean exit rule (critical)

To avoid ghost borders in the terminal:

1. TUI models must have a `quitting bool` field.
2. Set `m.quitting = true` before returning `tea.Quit`.
3. In `View()`, return `""` if `m.quitting` is true.
4. Print final output via `fmt.Println` *after* `p.Run()` returns in the command entry point.

---

## Interaction patterns

### Confirmation first

Any destructive or bulk action must show a confirm screen before applying:
- Checklist changes in `init` → summary of `+` added / `-` removed tools
- `remove` → summary of symlinks to be deleted

### Path safety

- Never `os.RemoveAll` on root directories (`/`, `~`, cwd).
- Always validate paths are within `<cwd>/<toolRootDir>/skills/` before any removal.
- Always validate names with `utils.ValidateName` before constructing file paths.

### Missing local config message

When `.skillops/config.json` is not found, commands (`add`, `remove`, `status`, `sync`) print:

```
No local config found.

If you're upgrading from v1, run:
  skillops init   — declare which IDEs this project uses
  skillops sync   — restore your skill links
```

---

## Config versioning & migration

`config_version` field in `agentics.yaml` tracks breaking changes to the default IDE list.

- **v1 → v2**: default list trimmed from 35+ to 9 IDEs. `EnsureConfig()` detects `config_version < 2`, prunes legacy entries from `legacyAgentics`, adds new defaults, bumps version to `2`.
- `EnsureConfig()` only adds missing keys — never removes user-added entries.

---

## Conventions

- Each `cmd/` file registers itself via `init()` calling `rootCmd.AddCommand(...)`.
- Commands are grouped with `GroupID`: `"project"` or `"skill"`.
- Skill identity format: `repo_name/skill_name` (e.g. `my-repo/logger`).
- A skill is valid only if it contains a `SKILL.md` file.
- Conflict detection: if a symlink short name already exists from a different repo → warn and skip, never silently overwrite.

---

*Last updated: 2026-04-03*
