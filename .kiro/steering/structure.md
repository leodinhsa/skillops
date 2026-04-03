# Project Structure

```
skillops/
├── main.go                  # Entry point, calls cmd.Execute()
├── go.mod / go.sum
├── .goreleaser.yaml         # Release config
├── DEPLOY.md                # Release process documentation
│
├── cmd/                     # Cobra commands (one file per command)
│   ├── root.go              # Root command, PersistentPreRun config init, custom help styling
│   ├── init.go              # skillops init — declare IDEs for this project
│   ├── add.go               # skillops add — link skills into IDEs
│   ├── remove.go            # skillops remove — unlink skills
│   ├── status.go            # skillops status — show linked skills
│   ├── sync.go              # skillops sync — restore symlinks from local config
│   ├── pull.go              # skillops pull — pull a skill repo from GitHub
│   ├── list.go              # skillops list — list downloaded skills (launches TUI)
│   ├── update.go            # skillops update — update pulled skill repos
│   ├── config.go            # skillops config — config management subcommands
│   └── version.go           # skillops version — print version
│
├── internal/
│   ├── config/
│   │   ├── config.go        # Global config R/W, defaultAgentics, EnsureConfig, migration
│   │   ├── localconfig.go   # Local project config R/W (.skillops/config.json)
│   │   └── settings.go      # Registry settings R/W (settings.yaml)
│   ├── git/
│   │   └── git.go           # Clone, pull, URL normalization helpers
│   ├── skills/
│   │   ├── skills.go        # Skill discovery (SKILL.md detection), metadata R/W
│   │   └── extract.go       # PullSkillFromURL (shared by pull --skill and sync auto-pull)
│   ├── symlink/
│   │   └── symlink.go       # Create/remove/check symlinks, find linked agentics
│   ├── tui/
│   │   ├── styles.go        # Shared lipgloss styles and color palette
│   │   ├── tui.go           # Main interactive TUI (init checklist, checklistModel)
│   │   ├── add.go           # Add TUI (skill select → tool select → confirm)
│   │   ├── remove.go        # Remove TUI (skill select → tool select → confirm)
│   │   ├── list.go          # List TUI view
│   │   └── init.go          # Init TUI entry point
│   └── utils/
│       └── utils.go         # Shared helpers (ValidateName, CopyDir, etc.)
│
└── plan/                    # Idea/planning docs (not shipped)
```

## Conventions

- Each `cmd/` file registers itself via `init()` calling `rootCmd.AddCommand(...)`
- Commands are grouped with `GroupID`: `"project"` or `"skill"`
- All shared TUI styles live in `internal/tui/styles.go` — never define one-off styles in command files
- Skill identity format: `repo_name/skill_name` (e.g., `my-repo/logger`)
- A skill is valid only if it contains a `SKILL.md` file
- Path safety: always validate names with `utils.ValidateName` before constructing file paths; never `os.RemoveAll` on root or cwd
- Destructive/bulk actions require a confirmation TUI step before execution

## Local config schema

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
