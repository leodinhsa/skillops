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
│   │   ├── localconfig.go   # Local project config R/W (.skillops/config.json v2)
│   │   ├── registry.go      # Registry matching logic (MatchRegistry, MatchesRegistry)
│   │   └── settings.go      # Registry settings R/W (settings.yaml, fallback only)
│   ├── git/
│   │   └── git.go           # Clone, pull, URL normalization, ParseRepoURL
│   ├── skills/
│   │   ├── skills.go        # Skill discovery (SKILL.md detection), ParsedIdentity, ParseIdentity
│   │   ├── metadata.go      # Skill/Repo metadata R/W (.so-skill-meta.json, .so-repo-meta.json)
│   │   └── extract.go       # PullSkillFromURL (shared by pull --skill and sync auto-pull)
│   ├── symlink/
│   │   └── symlink.go       # Create/remove/check symlinks, find linked agentics
│   ├── tui/
│   │   ├── styles.go        # Shared lipgloss styles and color palette
│   │   ├── tui.go           # Main interactive TUI (init checklist, checklistModel)
│   │   ├── add.go           # Add TUI (skill select → tool select → conflict detect → confirm)
│   │   ├── remove.go        # Remove TUI (skill select → tool select → disambiguate → confirm)
│   │   ├── conflict.go      # Conflict resolution TUI (custom symlink name input)
│   │   ├── list.go          # List TUI view
│   │   └── init.go          # Init TUI entry point
│   └── utils/
│       └── utils.go         # Shared helpers (ValidateName, CopyDir, path validation)
│
└── plan/                    # Idea/planning docs (not shipped)
```

## Conventions

- Each `cmd/` file registers itself via `init()` calling `rootCmd.AddCommand(...)`
- Commands are grouped with `GroupID`: `"project"` or `"skill"`
- All shared TUI styles live in `internal/tui/styles.go` — never define one-off styles in command files
- **Skill identity format**: `<host>/<repo-path>/<path-to-skill>` (e.g., `github.com/anthropics/skills/skills/logger`)
- **Short name**: Final component of identity path used for symlink (e.g., `logger`)
- **Custom symlink names**: Stored in `config.symlink_names` to resolve conflicts
- **Repo boundary**: Determined by registry URL prefix matching, NOT by parsing the identity string
- A skill is valid only if it contains a `SKILL.md` file
- Path safety: always validate identities with `ParseIdentity` before constructing file paths; never `os.RemoveAll` on root or cwd
- Destructive/bulk actions require a confirmation TUI step before execution

## Global Store Structure

```
~/.skillops/
├── config/
│   ├── agentics.yaml              # Global IDE registry (name → relative path)
│   └── settings.yaml              # Global registries (fallback, optional)
│
└── skills/                        # Global store (organized by full identity path)
    ├── github.com/
    │   ├── anthropics/
    │   │   └── skills/                          # ← Repo root
    │   │       ├── .so-repo-meta.json
    │   │       ├── .git/
    │   │       └── skills/
    │   │           ├── logger/
    │   │           │   ├── SKILL.md
    │   │           │   └── .so-skill-meta.json
    │   │           └── auth/
    │   │               ├── SKILL.md
    │   │               └── .so-skill-meta.json
    │   └── company-private/
    │       └── enterprise-skills/               # ← Repo root
    │           └── api/
    │               └── rate-limiter/
    │                   ├── SKILL.md
    │                   └── .so-skill-meta.json
    ├── gitlab.com/
    │   └── devops-team/
    │       └── ci-helpers/                      # ← Repo root
    │           └── docker-builder/
    │               ├── SKILL.md
    │               └── .so-skill-meta.json
    ├── gitlab.common.datumhq.com/
    │   └── datumhq-consulting-vn/
    │       └── management/
    │           └── datum-skills/
    │               └── software-skills/         # ← Repo root (multi-level groups)
    │                   ├── .so-repo-meta.json
    │                   └── skills/
    │                       └── logger/
    │                           ├── SKILL.md
    │                           └── .so-skill-meta.json
    └── bitbucket.org/
        └── frontend-guild/
            └── react-skills/                    # ← Repo root
                └── components/
                    └── form-handler/
                        ├── SKILL.md
                        └── .so-skill-meta.json
```

**Note:** Repo root is determined by registry URL, not by filesystem markers.

## Local Config Schema (V2)

```json
{
  "version": "2",
  "registries": [
    {
      "url": "https://github.com/anthropics/skills",
      "name": "Anthropic Public Skills",
      "priority": 1
    },
    {
      "url": "git@github.com:company-private/enterprise-skills",
      "name": "Company Private Skills",
      "priority": 2
    }
  ],
  "tools": {
    "kiro": [
      "github.com/anthropics/skills/skills/logger",
      "github.com/anthropics/skills/skills/auth",
      "github.com/company-private/enterprise-skills/api/rate-limiter"
    ],
    "cursor": [
      "github.com/anthropics/skills/skills/logger"
    ]
  },
  "symlink_names": {
    "github.com/company-a/utils/tools/logger": "logger-utils",
    "github.com/company-b/helpers/services/logger": "logger-services"
  }
}
```

**Critical**: Config v1 is NOT supported. Version must be "2".
**Registry URL**: Points to the exact repository (not owner-scoped). This enables unambiguous prefix matching.

## Project Symlink Structure

```
my-project/
├── .skillops/
│   └── config.json                # Local config v2 (commit to git)
│
├── .kiro/
│   └── skills/                    # Flat symlink structure
│       ├── logger -> ~/.skillops/skills/github.com/anthropics/skills/skills/logger
│       ├── auth -> ~/.skillops/skills/github.com/anthropics/skills/skills/auth
│       ├── rate-limiter -> ~/.skillops/skills/github.com/company-private/enterprise-skills/api/rate-limiter
│       ├── logger-utils -> ~/.skillops/skills/github.com/company-a/utils/tools/logger
│       └── logger-services -> ~/.skillops/skills/github.com/company-b/helpers/services/logger
│
└── .cursor/
    └── skills/
        └── logger -> ~/.skillops/skills/github.com/anthropics/skills/skills/logger
```

## Key Data Structures

### ParsedIdentity
```go
type ParsedIdentity struct {
    Full        string   // gitlab.common.datumhq.com/datumhq-consulting-vn/management/datum-skills/software-skills/skills/logger
    Host        string   // gitlab.common.datumhq.com
    Path        string   // datumhq-consulting-vn/management/datum-skills/software-skills/skills/logger
    ShortName   string   // logger (final component, used for symlink)
}
```

**Note**: No `Owner` or `Repo` field. Repo boundary is determined by registry matching.

### Registry
```go
type Registry struct {
    URL      string   // https://github.com/anthropics/skills (full repo clone URL, no trailing slash)
    Name     string   // Anthropic Public Skills
    Priority int      // Lower number = higher priority
}
```

### SkillMetadata (.so-skill-meta.json)
```json
{
  "repo_url": "https://github.com/anthropics/skills",
  "path_in_repo": "skills/logger",
  "pulled_at": "2026-05-06T10:30:00Z",
  "commit_hash": "abc123def456"
}
```

### RepoMetadata (.so-repo-meta.json)
```json
{
  "repo_url": "https://github.com/anthropics/skills",
  "pulled_at": "2026-05-06T10:30:00Z",
  "commit_hash": "abc123def456"
}
```

## Data Flow

```
Global store (~/.skillops/skills/<host>/<full-path-to-skill>)
  └── populated by: skillops pull
  └── organized by: full identity path structure
  └── contains: .so-skill-meta.json or .so-repo-meta.json

Local config (.skillops/config.json v2)        ← source of truth
  └── managed by: init / add / remove
  └── contains: skill identities, registries, custom symlink names
  └── committed to git for team sharing

Project symlinks (derived state, flat structure)
  └── created by: add / sync
  └── removed by: remove / init (deselect)
  └── uses: short name or custom name from config.symlink_names
```

## Conflict Resolution

When multiple skills have the same short name:
- **TTY environment**: Launch interactive TUI for custom name input
- **Non-TTY environment**: Fail with descriptive error listing conflicts and suggesting manual config.json edit
- Store custom names in `config.symlink_names` map
- Never silently overwrite

## Registry Matching

- Registry URL is a full repo clone URL (e.g., `https://github.com/anthropics/skills`)
- Normalize URL: strip protocol, replace `:` with `/` for SSH → get prefix
- Match: skill identity starts with normalized registry prefix
- Path in repo = identity minus matched prefix
- Sort by priority (lower number = higher priority)
- Auto-populate registries when adding skills (read from skill metadata)
- Sync uses registries to auto-pull missing skills

## Path Validation

- Minimum 3 path components (host/something/skill)
- No empty components, no "." or "..", no path traversal
- Use `ParseIdentity` for validation before any filesystem operations
- Never `os.RemoveAll` on root directories (`/`, `~`, cwd)
