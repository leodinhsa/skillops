# Requirements Document

## Introduction

SkillOps v2 is a redesign of the `skillops` CLI tool that improves UX by reducing friction, clarifying the mental model, and making commands consistent. The core value proposition remains unchanged: one skill, many IDEs, always up to date — via a global store at `~/.skillops/skills/` and symlinks into IDE-specific project folders.

The v2 redesign introduces a local project config (`.skillops/config.json`) as the source of truth for which IDEs and skills are active in a project, replaces the confusing `agentic` command family with clearer `init`/`add`/`remove` commands, and adds `status` and `sync` commands for visibility and recovery.

## Glossary

- **CLI**: The `skillops` command-line interface tool
- **Global Store**: The `~/.skillops/skills/` directory containing all pulled skill repos
- **Local Config**: The `.skillops/config.json` file at the project root, committed to git
- **Skill**: A directory containing a `SKILL.md` file, discovered from pulled repos in the global store
- **Skill Identity**: Full format `repo/skill` (e.g., `repo-a/auth-agent`) used in local config to avoid name conflicts
- **Short Name**: The skill name portion only (e.g., `auth-agent`), used as the symlink filename on disk
- **Tool**: An agentic IDE environment (e.g., `kiro`, `claude-code`, `cursor`) with a known relative skills path
- **Active Tool**: A tool declared in the local config for the current project
- **Symlink**: A filesystem symbolic link mapping a global skill into a project's tool-specific skills directory
- **Global Config**: `~/.skillops/config/agentics.yaml` mapping tool names to relative skills paths
- **TUI**: Terminal User Interface built with bubbletea and lipgloss

---

## Requirements

### Requirement 1: Trim Default IDE List

**User Story:** As a developer, I want the default IDE list to contain only popular tools, so that the TUI is not cluttered with obscure entries.

#### Acceptance Criteria

1. THE CLI SHALL include exactly 9 tools in `defaultAgentics`: `claude-code`, `cursor`, `windsurf`, `kiro`, `gemini-cli`, `goose`, `github-copilot`, `opencode`, and `antigravity`.
2. THE CLI SHALL map each default tool to its canonical skills path: `claude-code → .claude/skills`, `cursor → .cursor/skills`, `windsurf → .windsurf/skills`, `kiro → .kiro/skills`, `gemini-cli → .gemini/skills`, `goose → .goose/skills`, `github-copilot → .github/skills`, `opencode → .agents/skills`, `antigravity → .agent/skills`.
3. THE CLI SHALL comment out (not delete) previously supported tools in `internal/config/config.go` so they can be re-enabled without code loss.
4. WHEN `EnsureConfig()` runs on a machine with an existing config, THE CLI SHALL preserve all existing tool entries and only add missing default entries.

---

### Requirement 2: Local Project Config

**User Story:** As a developer, I want a project-level config file that records which IDEs and skills are active, so that my team can restore the same setup after cloning the repo.

#### Acceptance Criteria

1. THE CLI SHALL store local project config at `.skillops/config.json` relative to the current working directory.
2. THE Local_Config SHALL use the following JSON schema with version `"1"`:
   ```json
   {
     "version": "1",
     "tools": {
       "<tool-name>": ["<repo>/<skill>", ...]
     }
   }
   ```
3. THE Local_Config SHALL store skill identities in full `repo/skill` format to prevent name conflicts across repos.
4. THE CLI SHALL implement local config functions in `internal/config/localconfig.go` within the existing `config` package (not a new package).
5. THE Local_Config SHALL be the source of truth; symlinks on disk are derived state.
6. THE CLI SHALL expose the following functions: `LocalConfigPath()`, `ReadLocalConfig()`, `WriteLocalConfig()`, `GetActiveTools()`, `GetToolSkills()`, `AddSkillToTool()`, `RemoveSkillFromTool()`, `SetActiveTools()`.

---

### Requirement 3: `skillops init` Command

**User Story:** As a developer, I want to declare which IDEs my project uses in a single command, so that subsequent `add` and `remove` commands know where to create symlinks.

#### Acceptance Criteria

1. THE `init` Command SHALL present a TUI checklist of all tools from the global config for the user to select or deselect.
2. WHEN `.skillops/config.json` already exists, THE `init` Command SHALL pre-check the tools already declared in that file.
3. WHEN the user confirms the selection, THE `init` Command SHALL write the selected tool list to `.skillops/config.json`.
4. WHEN a tool is newly selected, THE `init` Command SHALL create the tool's skills directory (e.g., `.kiro/skills/`) if it does not exist.
5. WHEN a tool is deselected, THE `init` Command SHALL remove all symlinks from that tool's skills directory and remove the tool entry from local config.
6. WHEN a tool is deselected, THE `init` Command SHALL NOT delete the tool's root directory (e.g., `.kiro/`) — only symlinks inside the skills subdirectory are removed.
7. THE `init` Command SHALL be idempotent: running it multiple times with the same selection produces the same result.
8. THE `init` Command SHALL replace `skillops agentic` and its subcommands.

---

### Requirement 4: `skillops add` Command

**User Story:** As a developer, I want to link a skill into my project's active IDEs, so that each IDE can use that skill without manual symlink management.

#### Acceptance Criteria

1. WHEN `.skillops/config.json` does not exist, THE `add` Command SHALL exit with the error message: `"Run 'skillops init' first"`.
2. WHEN invoked with no arguments, THE `add` Command SHALL present a two-screen TUI: screen 1 selects skills from the global store, screen 2 selects target tools from the active tools in local config.
3. WHEN invoked as `skillops add <skill-name>`, THE `add` Command SHALL skip skill selection and present only the tool selection screen.
4. WHEN invoked with `--all`, THE `add` Command SHALL link the skill into all active tools without a tool selection screen.
5. WHEN invoked with `--tool <tool,...>`, THE `add` Command SHALL link the skill into only the specified tools.
6. WHEN creating a symlink, THE `add` Command SHALL use the short name (portion after `/`) as the symlink filename.
7. WHEN creating a symlink, THE `add` Command SHALL create the tool's skills directory if it does not exist.
8. WHEN a symlink is created, THE `add` Command SHALL update local config by adding the full `repo/skill` identity to the tool's entry.
9. WHEN a short name conflict is detected (same short name already linked from a different repo), THE `add` Command SHALL warn the user and SHALL NOT overwrite the existing symlink.
10. THE `add` Command SHALL present a confirmation screen summarizing changes before applying them.

---

### Requirement 5: `skillops remove` Command (Rewrite)

**User Story:** As a developer, I want to unlink a skill from my project's IDEs without deleting it from the global store, so that I can safely remove a skill from one project while keeping it available globally.

#### Acceptance Criteria

1. THE `remove` Command SHALL only remove symlinks and update local config; it SHALL NOT delete anything from the global store (`~/.skillops/skills/`).
2. WHEN invoked with no arguments, THE `remove` Command SHALL present a two-screen TUI: screen 1 selects skills currently linked in the project, screen 2 selects which tools to unlink from.
3. WHEN invoked as `skillops remove <skill-name>`, THE `remove` Command SHALL skip skill selection and present only the tool selection screen.
4. WHEN invoked with `--all`, THE `remove` Command SHALL unlink the skill from all active tools.
5. WHEN invoked with `--tool <tool,...>`, THE `remove` Command SHALL unlink the skill from only the specified tools.
6. WHEN a symlink does not exist for a given tool, THE `remove` Command SHALL skip that tool without error (idempotent).
7. WHEN a symlink is removed, THE `remove` Command SHALL update local config by removing the full `repo/skill` identity from the tool's entry.
8. THE `remove` Command SHALL present a confirmation screen summarizing changes before applying them.
9. THE `remove` Command SHALL replace the existing `skillops remove` (which deleted from global store) and `skillops remove-all` commands.

---

### Requirement 6: `skillops status` Command

**User Story:** As a developer, I want to see the current skill and IDE state of my project at a glance, so that I can quickly understand what is linked and what needs attention.

#### Acceptance Criteria

1. THE `status` Command SHALL render output using lipgloss styles from `internal/tui/styles.go`; it SHALL NOT produce plain text output.
2. THE `status` Command SHALL display each active tool as a section header.
3. FOR EACH skill in a tool's local config entry, THE `status` Command SHALL display `◉` if the symlink exists on disk, or `○` if the symlink is missing (broken/not yet synced).
4. WHEN a tool has no skills in local config, THE `status` Command SHALL display `— no skills linked` for that tool.
5. THE `status` Command SHALL display only the repo name and skill name; it SHALL NOT display the full `~/.skillops/skills/...` path.
6. THE `status` Command SHALL display a footer summary showing the count of active tools and linked skills.
7. WHEN `.skillops/config.json` does not exist, THE `status` Command SHALL exit with the message: `"No local config found. Run 'skillops init' first"`.

---

### Requirement 7: `skillops sync` Command

**User Story:** As a developer, I want to restore all symlinks from the local config after cloning a repo, so that I don't have to manually re-run `add` for every skill.

#### Acceptance Criteria

1. THE `sync` Command SHALL read `.skillops/config.json` and recreate any missing symlinks for all tools and skills declared in it.
2. WHEN a skill declared in local config exists in the global store, THE `sync` Command SHALL create the symlink if it does not already exist.
3. WHEN a skill declared in local config does NOT exist in the global store, THE `sync` Command SHALL emit a warning: `"skill '<repo>/<skill>' not found locally, run 'skillops pull'"`.
4. THE `sync` Command SHALL NOT remove any existing symlinks (that is the responsibility of `remove`).
5. THE `sync` Command SHALL NOT trigger `skillops update` or pull any remote changes.
6. THE `sync` Command SHALL create the tool's skills directory if it does not exist.
7. THE `sync` Command SHALL display a TUI summary of symlinks created and warnings emitted after completion.
8. WHEN `.skillops/config.json` does not exist, THE `sync` Command SHALL exit with the message: `"No local config found. Run 'skillops init' first"`.

---

### Requirement 8: Delete Deprecated Commands

**User Story:** As a developer, I want the CLI to remove outdated commands that are replaced by v2 equivalents, so that the command surface is clean and consistent.

#### Acceptance Criteria

1. THE CLI SHALL delete `cmd/agentic.go` and all subcommands it registers (`agentic`, `agentic manage`, `agentic remove-skill`, `agentic remove-skills`).
2. THE CLI SHALL delete the `remove-all` command from `cmd/remove.go`.
3. WHEN a user runs `skillops agentic`, THE CLI SHALL return a "command not found" error (cobra default behavior after deletion).

---

### Requirement 9: TUI Consistency

**User Story:** As a developer, I want all TUI screens to follow the same visual and interaction patterns, so that the tool feels cohesive and predictable.

#### Acceptance Criteria

1. ALL TUI models SHALL include a `quitting bool` field.
2. WHEN a TUI model is quitting, THE TUI SHALL return `""` from `View()` to prevent ghost borders in the terminal.
3. ALL TUI models SHALL print final output via `fmt.Println` after `p.Run()` returns in the command entry point, not inside `View()`.
4. ALL TUI screens SHALL use styles exclusively from `internal/tui/styles.go`; no one-off lipgloss styles SHALL be defined in command files.
5. ALL destructive actions (removing symlinks in bulk, deselecting tools in `init`) SHALL require a confirmation TUI step before execution.
6. ALL TUI checklist screens SHALL support keyboard navigation: `↑`/`↓` to move, `space` to toggle, `enter` to confirm, `esc`/`ctrl+c` to cancel.

---

### Requirement 10: Unchanged Commands

**User Story:** As a developer, I want existing commands that work well to remain unchanged, so that my current workflows are not disrupted.

#### Acceptance Criteria

1. THE `pull` Command SHALL remain unchanged in behavior and flags.
2. THE `list` Command SHALL remain unchanged in behavior.
3. THE `update` Command SHALL remain unchanged in behavior and flags.
4. THE `version` Command SHALL remain unchanged.
5. THE `config add-agentic`, `config remove-agentic`, and `config update-agentic` subcommands SHALL remain unchanged.
