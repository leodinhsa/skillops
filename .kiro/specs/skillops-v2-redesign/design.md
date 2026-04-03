# Design Document: SkillOps v2 Redesign

## Overview

SkillOps v2 replaces the confusing `agentic` command family with a cleaner mental model built around a local project config file. The core mechanic is unchanged — a global skill store at `~/.skillops/skills/` with symlinks into IDE-specific project directories — but v2 adds a `.skillops/config.json` as the source of truth for which IDEs and skills are active in a project.

The redesign introduces five changes:
1. Trim the default IDE list from 35+ to 9 popular tools
2. Add `internal/config/localconfig.go` for local project state
3. Replace `skillops agentic` with `skillops init`
4. Add `skillops add`, rewrite `skillops remove` (unlink-only)
5. Add `skillops status` and `skillops sync` for visibility and recovery

## Architecture

### Data Flow

```
Global Store (~/.skillops/skills/)
  └── <repo>/
        └── <skill>/SKILL.md   ← pulled by `skillops pull`

Local Config (.skillops/config.json)           ← source of truth
  └── tools:
        └── <tool>: ["repo/skill", ...]        ← managed by init/add/remove

Project Symlinks (derived state)
  └── .<tool>/skills/<short-name> → ~/.skillops/skills/<repo>/<skill>
                                    ↑ created by add/sync, removed by remove/init
```

### Command Dependency Graph

```
skillops pull          → populates global store (unchanged)
skillops init          → reads global config, writes local config, creates/removes dirs+symlinks
skillops add           → reads local config + global store, creates symlinks, updates local config
skillops remove        → reads local config, removes symlinks, updates local config
skillops sync          → reads local config + global store, creates missing symlinks only
skillops status        → reads local config, checks symlinks on disk, renders TUI
```

### State Machines

**`init` state machine:**
```
CHECKLIST → (enter) → CONFIRM → (yes) → APPLY → done
                              → (no)  → CHECKLIST
         → (esc)   → quit
```

**`add` state machine (no args):**
```
SKILL_SELECT → (enter) → TOOL_SELECT → (enter) → CONFIRM → (yes) → APPLY → done
                                                          → (no)  → TOOL_SELECT
            → (esc) → quit
```

**`remove` state machine (no args):**
```
SKILL_SELECT → (enter) → TOOL_SELECT → (enter) → CONFIRM → (yes) → APPLY → done
                                                          → (no)  → TOOL_SELECT
            → (esc) → quit
```

## Components and Interfaces

### `internal/config/localconfig.go` (new file, package `config`)

Struct definitions:

```go
type LocalConfig struct {
    Version string              `json:"version"`
    Tools   map[string][]string `json:"tools"` // tool → []"repo/skill"
}
```

Function signatures:

```go
// LocalConfigPath returns the absolute path to .skillops/config.json in cwd.
func LocalConfigPath() string

// ReadLocalConfig reads and parses .skillops/config.json.
// Returns an error wrapping os.ErrNotExist if the file does not exist.
func ReadLocalConfig() (LocalConfig, error)

// WriteLocalConfig writes cfg to .skillops/config.json, creating the
// .skillops/ directory if needed. Uses JSON with 2-space indentation.
func WriteLocalConfig(cfg LocalConfig) error

// GetActiveTools returns the list of tool names declared in local config.
func GetActiveTools() ([]string, error)

// GetToolSkills returns the full "repo/skill" identities for a given tool.
func GetToolSkills(tool string) ([]string, error)

// AddSkillToTool appends repoSkill ("repo/skill") to the tool's entry.
// No-ops if already present. Creates the tool entry if missing.
func AddSkillToTool(tool, repoSkill string) error

// RemoveSkillFromTool removes repoSkill from the tool's entry.
// No-ops if not present.
func RemoveSkillFromTool(tool, repoSkill string) error

// SetActiveTools replaces the tools map keys with the given list,
// preserving existing skill entries for tools that remain active.
// Tools removed from the list have their entries deleted.
func SetActiveTools(tools []string) error
```

Key implementation notes:
- `LocalConfigPath()` uses `os.Getwd()` + `filepath.Join(".skillops", "config.json")`
- `WriteLocalConfig` creates `.skillops/` with `os.MkdirAll` before writing
- All functions that read config must handle `os.IsNotExist` gracefully
- `LocalConfig.Version` is always written as `"1"`

### `internal/config/config.go` (modified)

`defaultAgentics` is trimmed to exactly 9 entries. The removed entries are commented out (not deleted) so they can be re-enabled without code loss:

```go
var defaultAgentics = map[string]string{
    "claude-code":    ".claude/skills",
    "cursor":         ".cursor/skills",
    "windsurf":       ".windsurf/skills",
    "kiro":           ".kiro/skills",
    "gemini-cli":     ".gemini/skills",
    "goose":          ".goose/skills",
    "github-copilot": ".github/skills",
    "opencode":       ".agents/skills",
    "antigravity":    ".agent/skills",
}
// Commented out (not deleted): universal, augment, openclaw, cline, codebuddy, ...
```

`EnsureConfig()` behavior is unchanged — it only adds missing keys, never removes existing ones.

### `cmd/init.go` (new)

Cobra command registered with `GroupID: "project"`. No flags.

Calls `tui.RunInit()` which drives the `checklistModel` (updated in `tui.go`).

**Updated `checklistModel` in `internal/tui/tui.go`:**

The existing `checklistModel` is repurposed for `init`. Key changes:
- Pre-check logic reads from `ReadLocalConfig()` instead of checking for directory existence
- `applyChanges()` is rewritten to:
  1. Call `SetActiveTools(selectedTools)` to update local config
  2. For newly added tools: `os.MkdirAll(filepath.Join(cwd, toolSkillsPath), 0755)`
  3. For removed tools: enumerate symlinks in the skills dir and remove each; do NOT remove the root IDE dir
- The confirm screen shows `+` / `-` diffs before applying
- Destructive removals (symlink cleanup) are batched and shown in the confirm screen; no per-item confirm needed since the summary screen already serves as confirmation

### `cmd/add.go` (new)

Cobra command registered with `GroupID: "project"`. Optional positional arg `[skill]`.

Flags:
- `--all` — link into all active tools
- `--tool string` — comma-separated list of tools to target

**TUI flow (no args):**

Screen 1 — skill selection: reuses the existing `model` from `tui.go` but in read-only checklist mode (no path editing, no filter-by-agentic). Shows all skills from `skills.Discover()`, grouped by repo.

Screen 2 — tool selection: a new `toolSelectModel` (checklist of active tools from `GetActiveTools()`).

Screen 3 — confirm: shows `+ <short-name> → <tool>` for each planned symlink.

**Non-TUI paths:**
- `skillops add <skill> --all`: resolve skill from global store, link into all active tools, no TUI
- `skillops add <skill> --tool kiro,cursor`: link into specified tools only, no TUI

**Conflict detection logic:**

Before creating any symlink at `<cwd>/<toolPath>/<shortName>`:
1. Check if a file/symlink already exists at that path via `os.Lstat`
2. If it exists and is a symlink, read its target via `os.Readlink`
3. Extract the repo from the existing target path
4. If the existing target's repo differs from the new skill's repo → conflict: print warning and skip
5. If the existing target matches the new skill's path → already linked, skip silently

### `cmd/remove.go` (rewrite)

Same flag surface as `add`: optional `[skill]` arg, `--all`, `--tool`.

**Key behavioral difference from current `remove`:** never touches `~/.skillops/skills/`. Only removes symlinks and updates local config.

**TUI flow (no args):**

Screen 1 — skill selection: shows only skills currently in local config (not all global skills).

Screen 2 — tool selection: shows only tools that have the selected skill linked.

Screen 3 — confirm: shows `- <short-name> from <tool>` for each planned removal.

**Idempotence:** if the symlink doesn't exist on disk, `RemoveSymlink` already returns nil (no-op). `RemoveSkillFromTool` is also a no-op if the entry isn't present.

### `cmd/status.go` (new)

Cobra command registered with `GroupID: "project"`. No flags. No interactive TUI — renders a static lipgloss panel and exits.

**Rendering logic:**

```
1. ReadLocalConfig() — exit with message if not found
2. GetActiveTools() → sorted list of tools
3. For each tool:
   a. GetToolSkills(tool) → []"repo/skill"
   b. GetAgenticPath(tool) → relative skills path
   c. For each "repo/skill":
      - shortName = portion after "/"
      - symlinkPath = filepath.Join(cwd, toolPath, shortName)
      - os.Lstat(symlinkPath) → exists? → ◉ : ○
   d. If no skills → "— no skills linked"
4. Footer: count active tools, count symlinks where Lstat succeeded
```

**Output format** (rendered with lipgloss, not plain text):

```
╭─────────────────────────────────────────╮
│           PROJECT STATUS                │
│  /path/to/my-project                    │
├─────────────────────────────────────────┤
│  claude-code                            │
│    ◉ auth-agent        (repo-a)         │
│    ○ logging-agent     not linked       │
│                                         │
│  kiro                                   │
│    — no skills linked                   │
│                                         │
│  2 tools active • 1 skill linked        │
╰─────────────────────────────────────────╯
```

`◉` = symlink exists on disk  
`○` = in local config but symlink missing (needs `sync`)  
`—` = tool active but no skills in local config

The full `~/.skillops/skills/...` path is never shown. Only `(repo-name)` is shown as context.

### `internal/skills/extract.go` (new file, package `skills`)

Shared logic extracted from `cmd/pull.go` so both `pull --skill` and `sync` registry auto-pull use the same code path.

```go
// PullSkillFromURL clones repoURL into a temp dir, finds the skill folder
// matching skillName (using the same 3-rule discovery as Discover()), copies
// it to destSkillDir, saves metadata.json, and cleans up the temp dir.
//
// Discovery rules (in order):
//   1. Root skill: SKILL.md at repo root → skill folder = repo root
//   2. Container skill: skills/<skillName>/SKILL.md → skill folder = skills/<skillName>
//   3. Direct subfolder: <skillName>/SKILL.md → skill folder = <skillName>
//
// destSkillDir is the final path where the skill folder will be placed,
// e.g. ~/.skillops/skills/<repoName>/<skillName>.
// metadata.json is written to filepath.Dir(destSkillDir) with {url, skill_name}.
func PullSkillFromURL(repoURL, skillName, destSkillDir string) error
```

`cmd/pull.go` is refactored to call `PullSkillFromURL` instead of inlining the logic.

---

### `cmd/sync.go` (new)

Cobra command registered with `GroupID: "project"`. No flags.

#### Full Sync Flowchart

```
skillops sync
│
├─ Read .skillops/config.json
│   └─ NOT FOUND → exit: "No local config found. Run 'skillops init' first"
│
├─ Read ~/.skillops/config/settings.yaml
│   ├─ NOT FOUND → registries = []  (silent, no error)
│   └─ MALFORMED → registries = [], warn to stderr
│
├─ FOR EACH tool in local config
│   │
│   ├─ toolPath = GetAgenticPath(tool)
│   ├─ os.MkdirAll(<cwd>/<toolPath>)
│   │
│   └─ FOR EACH "repo/skill" in tool's skill list
│       │
│       ├─ shortName       = portion after  "/"   (e.g. "auth-agent")
│       ├─ repoName        = portion before "/"   (e.g. "repo-a")
│       ├─ globalSkillPath = ~/.skillops/skills/<repoName>/<skillName>
│       ├─ symlinkPath     = <cwd>/<toolPath>/<shortName>
│       │
│       ├─ [A] os.Stat(globalSkillPath) exists?
│       │       │
│       │       ├─ YES ──────────────────────────────────────────► [D]
│       │       │
│       │       └─ NO
│       │           │
│       │           ├─ [B] len(registries) > 0 ?
│       │           │       │
│       │           │       ├─ NO → record WARN:
│       │           │       │       "skill '<repo>/<skill>' not found locally,
│       │           │       │        run 'skillops pull'"
│       │           │       │       → continue (next skill)
│       │           │       │
│       │           │       └─ YES
│       │           │           │
│       │           │           └─ [C] FOR EACH registry (in config order)
│       │           │               │
│       │           │               ├─ cloneURL = registry.URL + "/" + repoName
│       │           │               ├─ call PullSkillFromURL(cloneURL, skillName,
│       │           │               │                        globalSkillPath)
│       │           │               │
│       │           │               ├─ SUCCESS → record "auto-pulled from <name>"
│       │           │               │            → break ──────────────────► [D]
│       │           │               │
│       │           │               └─ FAIL → try next registry
│       │           │                   └─ no more registries?
│       │           │                       → record WARN:
│       │           │                         "skill '<repo>/<skill>' not found
│       │           │                          in any configured registry"
│       │           │                         → continue (next skill)
│       │
│       └─ [D] Create symlink
│               │
│               ├─ os.Lstat(symlinkPath) exists? → YES → skip (idempotent)
│               │
│               └─ os.Symlink(globalSkillPath, symlinkPath)
│                   ├─ SUCCESS → record "linked"
│                   └─ FAIL   → record ERROR, continue
│
└─ Render TUI summary panel
    ├─ ✓  X symlinks created
    ├─ ↓  Y auto-pulled from registry   (omit line if Y = 0)
    └─ ⚠  Z warnings                   (omit line if Z = 0)
```

#### Registry display name

`registry.Name` if non-empty, otherwise `"registry-<index+1>"` (e.g. `"registry-1"`).

#### Key invariants

- Sync is **purely additive** — it never removes symlinks or modifies existing ones.
- `PullSkillFromURL` is the **single code path** for extracting a skill from a remote repo, shared with `pull --skill`. No logic is duplicated.
- Registry auto-pull saves `metadata.json` identically to `pull --skill`, enabling `skillops update` to work on auto-pulled skills.

### `internal/config/settings.go` (new file, package `config`)

Struct definitions:

```go
type Registry struct {
    URL  string `yaml:"url"`
    Name string `yaml:"name,omitempty"`
}

type Settings struct {
    Registries []Registry `yaml:"registries"`
}
```

Function signatures:

```go
// SettingsPath returns filepath.Join(ConfigDir, "settings.yaml").
func SettingsPath() string

// ReadSettings reads ~/.skillops/config/settings.yaml.
// If the file is absent, returns empty Settings{} with no error.
// If the file is malformed, logs a warning to stderr and returns empty Settings{}.
func ReadSettings() (Settings, error)

// WriteSettings writes s to ~/.skillops/config/settings.yaml.
func WriteSettings(s Settings) error
```

`EnsureConfig()` in `config.go` is updated to call `ensureSettings()`, which writes `settings.yaml` with an empty registries list if the file does not exist.

`settings.yaml` lives in `~/.skillops/config/` and is never committed to git.

---

### `cmd/status.go` — registry section

When `ReadSettings()` returns a non-empty registries list, the status panel appends a `Registries:` section:

```
│  Registries: company-internal                   │
```

- Only the `name` field is shown. If `name` is empty, display `"registry-<index+1>"` (e.g. `"registry-1"`).
- Registry URLs are never displayed.
- The section is omitted entirely when no registries are configured.

---

### `cmd/agentic.go` (deleted)

The file is deleted. All subcommands (`agentic`, `agentic manage`, `agentic remove-skill`, `agentic remove-skills`) are removed. Cobra will return its default "unknown command" error if a user runs `skillops agentic`.

## Data Models

### LocalConfig

```go
type LocalConfig struct {
    Version string              `json:"version"` // always "1"
    Tools   map[string][]string `json:"tools"`   // tool → []"repo/skill"
}
```

Example on-disk representation:

```json
{
  "version": "1",
  "tools": {
    "claude-code": ["repo-a/auth-agent", "repo-a/logging-agent"],
    "kiro": ["repo-a/auth-agent"]
  }
}
```

### Skill Identity

Skills are stored in local config as `"repo/skill"` (full identity). The short name (symlink filename) is always derived at runtime as `strings.SplitN(identity, "/", 2)[1]`.

This means:
- `"my-repo/auth-agent"` → symlink filename `auth-agent`
- `"other-repo/auth-agent"` → same filename → **conflict** if both are added to the same tool

### Symlink Layout

```
<cwd>/
  .skillops/
    config.json                    ← local config
  .kiro/
    skills/
      auth-agent → ~/.skillops/skills/my-repo/auth-agent
  .claude/
    skills/
      auth-agent → ~/.skillops/skills/my-repo/auth-agent
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system — essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: EnsureConfig preserves existing entries

*For any* existing global config containing arbitrary tool entries, running `EnsureConfig()` should result in a config that contains all the original entries plus any missing defaults — no original entry should be removed or overwritten.

**Validates: Requirements 1.4**

---

### Property 2: LocalConfig round-trip

*For any* `LocalConfig` value with valid tool names and `"repo/skill"` identities, calling `WriteLocalConfig` followed by `ReadLocalConfig` should produce a value equal to the original.

**Validates: Requirements 2.2, 2.3**

---

### Property 3: init apply sets exactly the selected tools

*For any* set of tool names selected in the init checklist, after `applyChanges` completes, `GetActiveTools()` should return exactly that set (no more, no less).

**Validates: Requirements 3.3, 3.5**

---

### Property 4: init creates skills directories for selected tools

*For any* tool that is newly selected in init, after `applyChanges` completes, the tool's skills directory (e.g., `.kiro/skills/`) should exist on disk.

**Validates: Requirements 3.4, 4.7**

---

### Property 5: init does not delete root IDE directories

*For any* tool that is deselected in init, if the tool's root directory (e.g., `.kiro/`) existed before `applyChanges`, it should still exist after `applyChanges` completes.

**Validates: Requirements 3.6**

---

### Property 6: init is idempotent

*For any* selection of tools, running `applyChanges` twice with the same selection should produce the same local config and the same set of symlinks as running it once.

**Validates: Requirements 3.7**

---

### Property 7: add targets exactly the specified tools

*For any* skill identity and any subset of active tools (via `--all` or `--tool`), after `add` completes, `GetToolSkills(tool)` should contain the skill identity for every targeted tool and should not contain it for any non-targeted tool that did not already have it.

**Validates: Requirements 4.4, 4.5**

---

### Property 8: add uses short name as symlink filename

*For any* `"repo/skill"` identity, the symlink created on disk should have a filename equal to the portion after `"/"` (the short name), not the full identity string.

**Validates: Requirements 4.6**

---

### Property 9: add round-trip — symlink and local config are consistent

*For any* skill added to a tool, after `add` completes: (a) a symlink exists at `<cwd>/<toolPath>/<shortName>`, and (b) `GetToolSkills(tool)` contains the full `"repo/skill"` identity.

**Validates: Requirements 4.8**

---

### Property 10: add does not overwrite conflicting symlinks

*For any* tool that already has a symlink with a given short name pointing to a different repo, attempting to add a skill with the same short name from a different repo should leave the existing symlink unchanged and should not add the new identity to local config for that tool.

**Validates: Requirements 4.9**

---

### Property 11: remove does not touch the global store

*For any* skill removed from any tool, the skill's directory in `~/.skillops/skills/` should still exist and be unchanged after `remove` completes.

**Validates: Requirements 5.1**

---

### Property 12: remove targets exactly the specified tools

*For any* skill identity and any subset of active tools (via `--all` or `--tool`), after `remove` completes, `GetToolSkills(tool)` should not contain the skill identity for any targeted tool, and should still contain it for any non-targeted tool that had it.

**Validates: Requirements 5.4, 5.5**

---

### Property 13: remove round-trip — symlink and local config are consistent

*For any* skill removed from a tool, after `remove` completes: (a) no symlink exists at `<cwd>/<toolPath>/<shortName>`, and (b) `GetToolSkills(tool)` does not contain the full `"repo/skill"` identity.

**Validates: Requirements 5.7**

---

### Property 14: status renders correct symlink indicators without exposing global paths

*For any* local config and any set of symlinks on disk, the status output for each skill should show `◉` if and only if the symlink exists at the expected path, `○` otherwise — and the rendered string should not contain the `~/.skillops/skills/` prefix for any skill.

**Validates: Requirements 6.3, 6.5**

---

### Property 15: status renders correct structure for all tools

*For any* local config, the status output should include a section for every active tool, and the footer should show the correct count of active tools and linked skills.

**Validates: Requirements 6.2, 6.6**

---

### Property 16: sync creates all missing symlinks

*For any* local config where all declared skills exist in the global store, after `sync` completes, every declared symlink should exist on disk.

**Validates: Requirements 7.1, 7.2**

---

### Property 17: sync emits warnings for missing global skills

*For any* skill declared in local config that does not exist in the global store, `sync` should produce a warning message containing the full `"repo/skill"` identity.

**Validates: Requirements 7.3**

---

### Property 18: sync does not remove existing symlinks

*For any* symlink that exists on disk before `sync` runs, that symlink should still exist after `sync` completes.

**Validates: Requirements 7.4**

---

### Property 19: quitting TUI models return empty string from View()

*For any* TUI model where `quitting` is set to `true`, calling `View()` should return `""`.

**Validates: Requirements 9.2**

---

### Property 20: sync auto-pulls from registry when skill missing locally

*For any* skill absent from the global store but present in a configured registry, after `sync` completes, the skill directory should exist in `~/.skillops/skills/` and the symlink should be created at the expected project path.

**Validates: Requirements 11.8, 11.9**

---

### Property 21: sync tries registries in order and stops at first success

*For any* configuration where registry-1 fails and registry-2 succeeds, after `sync` completes, exactly one clone attempt should have succeeded from registry-2.

**Validates: Requirements 11.10**

---

### Property 22: sync with no registries configured behaves identically to current sync behavior

*For any* local config and empty/absent `settings.yaml`, warnings emitted for missing skills should be identical to pre-registry behavior.

**Validates: Requirements 11.5, 11.12**

---

## Error Handling

| Scenario | Command | Behavior |
|---|---|---|
| `.skillops/config.json` not found | `add`, `remove`, `status`, `sync` | Exit with `"No local config found. Run 'skillops init' first"` |
| No skills in global store | `add` (TUI mode) | Exit with `"No skills found. Use 'skillops pull' to download skill repositories."` |
| Short name conflict on add | `add` | Print warning per conflicting tool, skip that tool, continue with others |
| Skill not in global store | `sync` | Print warning per missing skill, continue with others |
| Tool not in global config | `add --tool`, `remove --tool` | Exit with `"unknown tool: <name>"` |
| Symlink target is not a symlink | `remove` | Skip with warning (do not attempt `os.Remove` on non-symlink) |
| Skills directory creation fails | `init`, `add`, `sync` | Return error, abort operation |
| `os.RemoveAll` safety check | `init` (deselect) | Validate path is within `<cwd>/<toolRootDir>/skills/` before any removal |
| Registry URL unreachable | `sync` | Try next registry; if all fail, warn `"skill not found in any configured registry"` |
| Registry reachable but repo not found | `sync` | Try next registry; if all fail, warn |
| `settings.yaml` malformed | all commands | Log warning to stderr, treat registries as empty list, continue |

All errors are printed to `os.Stderr`. Commands exit with code 1 on fatal errors.

## Testing Strategy

### Unit Tests

Unit tests cover specific examples and edge cases:

- `TestLocalConfigPath` — verifies path ends with `.skillops/config.json`
- `TestReadLocalConfig_NotFound` — verifies `os.IsNotExist` wrapping
- `TestDefaultAgentics_Count` — verifies exactly 9 entries
- `TestDefaultAgentics_Paths` — verifies each tool maps to the correct path
- `TestStatusRender_NoSkillsLinked` — verifies `"— no skills linked"` text
- `TestStatusRender_NoLocalConfig` — verifies correct error message
- `TestSyncRender_NoLocalConfig` — verifies correct error message
- `TestConflictDetection_SameRepo` — verifies same-repo re-add is a no-op (not a conflict)
- `TestRemoveSymlink_Idempotent` — verifies no error when symlink doesn't exist (edge case from 5.6)

### Property-Based Tests

Property-based tests use [pgregory.net/rapid](https://github.com/pgregory/rapid) (pure Go, no external dependencies beyond the module).

Each test runs a minimum of 100 iterations.

Tag format in comments: `// Feature: skillops-v2-redesign, Property N: <property_text>`

| Property | Test Name | Generator |
|---|---|---|
| 1 | `TestProp_EnsureConfigPreservesEntries` | random map[string]string of tool→path entries |
| 2 | `TestProp_LocalConfigRoundTrip` | random LocalConfig with valid tool names and repo/skill strings |
| 3 | `TestProp_InitApplySetsExactTools` | random subset of known tools |
| 4 | `TestProp_InitCreatesSkillsDirs` | random tool selection |
| 5 | `TestProp_InitDoesNotDeleteRootDirs` | random tool deselection |
| 6 | `TestProp_InitIdempotent` | random tool selection, apply twice |
| 7 | `TestProp_AddTargetsExactTools` | random active tools, random target subset |
| 8 | `TestProp_AddUsesShortName` | random "repo/skill" strings |
| 9 | `TestProp_AddRoundTrip` | random skill + tool |
| 10 | `TestProp_AddNoConflictOverwrite` | pre-existing symlink from different repo |
| 11 | `TestProp_RemoveDoesNotTouchGlobalStore` | random skill + tool |
| 12 | `TestProp_RemoveTargetsExactTools` | random active tools, random target subset |
| 13 | `TestProp_RemoveRoundTrip` | random skill + tool |
| 14 | `TestProp_StatusIndicatorsAndNoPaths` | random LocalConfig + random symlink presence |
| 15 | `TestProp_StatusStructure` | random LocalConfig |
| 16 | `TestProp_SyncCreatesMissingSymlinks` | random LocalConfig with all skills present |
| 17 | `TestProp_SyncWarnsForMissingSkills` | random LocalConfig with some skills absent |
| 18 | `TestProp_SyncDoesNotRemoveExisting` | random pre-existing symlinks |
| 19 | `TestProp_QuittingViewReturnsEmpty` | all TUI model types |
| 20 | `TestProp_SyncAutoPullsFromRegistry` | skill absent from global store, present in fake registry |
| 21 | `TestProp_SyncTriesRegistriesInOrder` | registry-1 fails, registry-2 succeeds |
| 22 | `TestProp_SyncNoRegistriesUnchangedBehavior` | empty/absent settings.yaml |

Property tests for filesystem operations use `t.TempDir()` to isolate state. Global config paths and `SkillsDir` are overridden via test helpers that temporarily set the package-level vars.
