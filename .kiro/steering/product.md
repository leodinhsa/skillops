---
inclusion: always
---

# SkillOps - Product Overview

`skillops` is a CLI tool for managing AI agent "skills" (modular capabilities/scripts) across multiple Agentic IDEs (Claude Code, Cursor, Windsurf, Kiro, Roo, etc.).

## Core Problem
Each IDE stores skills in different directories. Manually syncing shared skill repositories across IDEs is tedious and error-prone. Additionally, teams need to share skill sources without manual configuration.

## Solution Architecture

### Storage Structure
- **Global store**: `~/.skillops/skills/<host>/<owner>/<repo>/<path-to-skill>` holds all pulled skills
- **Symlinks**: Flat symlinks in IDE-specific project paths (e.g., `.kiro/skills/logger`)
- **Global config**: `~/.skillops/config/agentics.yaml` maps IDE names to their relative skill folder paths
- **Local config**: `.skillops/config.json` (version 2) tracks skills, registries, and custom symlink names per project (source of truth, commit to git)

### Config Files

#### Global Config (`~/.skillops/config/agentics.yaml`)
Maps IDE names to their skill directory paths:
```yaml
config_version: 2
agentics:
  kiro: .kiro/skills
  cursor: .cursor/skills
  claude-code: .claude/skills
```

#### Local Config (`.skillops/config.json` v2)
Project-specific configuration (commit to git):
```json
{
  "version": "2",
  "registries": [
    {
      "url": "https://github.com/anthropics/skills",
      "name": "Anthropic Public Skills",
      "priority": 1
    }
  ],
  "tools": {
    "kiro": [
      "github.com/anthropics/skills/skills/logger",
      "github.com/anthropics/skills/skills/auth"
    ]
  },
  "symlink_names": {
    "github.com/company-a/utils/tools/logger": "logger-utils"
  }
}
```

**Critical**: Config v1 is NOT supported. Version must be "2".

## Key Concepts

### Skill Identity
Full-path format: `<host>/<repo-path>/<path-to-skill>`

Examples:
- `github.com/anthropics/skills/skills/logger`
- `gitlab.com/devops-team/ci-helpers/docker-builder`
- `github.com/company/monorepo/backend/services/api/skills/auth` (nested)
- `gitlab.common.datumhq.com/datumhq-consulting-vn/management/datum-skills/software-skills/skills/logger` (multi-level groups)

**Critical**: A directory is only a valid skill if it contains `SKILL.md`.

**Key design decision**: The identity string does NOT encode where the repository ends and the skill path begins. This boundary is determined by **registry URL prefix matching** at runtime.

### Skill Components
- **Host**: Git hosting platform (e.g., `github.com`, `gitlab.com`, `gitlab.company.internal`)
- **Short name**: Final component used for symlink (e.g., `logger` from the identity path)

**Note**: There is no separate "owner" or "repo" field in the parsed identity. The full path between host and short name is stored as-is. The repo boundary is determined by registry matching when clone operations are needed.

### Registry
A full repository clone URL (no trailing slash):
- `https://github.com/anthropics/skills`
- `git@github.com:company-private/enterprise-skills`
- `https://gitlab.common.datumhq.com/datumhq-consulting-vn/management/datum-skills/software-skills`

Registries enable zero-config team onboarding. When a teammate clones the project and runs `skillops sync`, missing skills are auto-pulled from configured registries.

**Registry matching**: The normalized registry URL (protocol stripped) is compared as a prefix against the skill identity. The remainder after the prefix is the `path-to-skill` within the repository.

### Symlink Names
- **Default**: Short name (final path component)
- **Custom**: User-provided name to resolve conflicts when multiple skills have the same short name
- Only custom names are stored in `config.symlink_names`

### Metadata Files
- **Skill metadata** (`.so-skill-meta.json`): Contains `repo_url`, `path_in_repo`, `pulled_at`, `commit_hash`
- **Repo metadata** (`.so-repo-meta.json`): For full repository pulls

## Data Flow

```
Global store (~/.skillops/skills/<host>/<owner>/<repo>/<path>)
  └── populated by: skillops pull
  └── organized by: full-path structure

Local config (.skillops/config.json v2)        ← source of truth
  └── managed by: init / add / remove
  └── contains: skill identities, registries, custom symlink names

Project symlinks (derived state, flat structure)
  └── created by: add / sync
  └── removed by: remove / init (deselect)
  └── uses: short name or custom name
```

## Development Principles

### Source of Truth
`.skillops/config.json` (v2) is the source of truth. Symlinks are derived state that can be recreated via `skillops sync`.

### Skill Identity Format
- **Internal**: Always use full-path format `<host>/<repo-path>/<path-to-skill>`
- **Symlinks**: Use short name (final component) or custom name from config
- **Minimum**: 3 path components (host/something/skill)
- **Validation**: No empty components, no "." or "..", no path traversal
- **Repo boundary**: Determined by registry URL prefix matching, NOT by parsing the identity string

### Global Store Structure
- Organized by full identity path: `~/.skillops/skills/<host>/<repo-path>/<path-to-skill>`
- Supports multi-level groups (e.g., `gitlab.com/group/subgroup/project`)
- Prevents repository collision (different paths = different directories)
- Supports arbitrary nesting depth
- Repo boundary is NOT encoded in filesystem — determined by registry matching

### Symlink Structure
- **Flat**: IDE skill directories remain flat (no nested folders)
- **Global path**: Symlink target uses full nested path in global store
- **Name resolution**: Short name → custom name (if exists in config) → symlink filename

### Path Safety
- Never `os.RemoveAll` on root directories (`/`, `~`, cwd)
- Always validate paths are within `<cwd>/<toolRootDir>/skills/` before removal
- Always validate identity components (no empty, ".", "..", path traversal)
- Use `utils.ValidateName` before constructing file paths

### Conflict Detection & Resolution
When multiple skills have the same short name:
- **TTY environment**: Launch interactive TUI for custom name input
- **Non-TTY environment**: Fail with descriptive error listing conflicts and suggesting manual config.json edit
- Store custom names in `config.symlink_names` map
- Never silently overwrite

### Registry Matching
- Registry URL is a full repo clone URL (e.g., `https://github.com/anthropics/skills`)
- Normalize URL (strip protocol, replace `:` with `/` for SSH) to get prefix
- Match: identity starts with normalized registry prefix
- Path in repo = identity minus the matched prefix
- Sort by priority (lower number = higher priority)
- Auto-populate registries when adding skills (read from skill metadata)
- Sync uses registries to auto-pull missing skills

### User Experience
- Destructive or bulk actions require confirmation before execution
- Missing local config should guide users to run `skillops init` then `skillops sync`
- TUI interactions must follow the clean exit rule (see tech.md)
- Config v1 detection: Fail with clear error suggesting `skillops init` to migrate

### Zero-Config Team Onboarding
1. Developer clones project with `.skillops/config.json` (v2)
2. Runs `skillops sync`
3. System auto-pulls missing skills from configured registries
4. Symlinks created automatically
5. No manual registry configuration needed
