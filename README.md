# SkillOps

A lightweight CLI to manage AI agent skills across multiple Agentic IDEs using a symlink-first approach. Pull skill repositories once, link them into any IDE — no duplication.

## Installation

### Homebrew (macOS & Linux)

```bash
brew tap leodinhsa/skillops
brew install skillops
```

To upgrade to the latest version:

```bash
brew upgrade skillops
```

### From source

```bash
go build -o skillops .
mv skillops /usr/local/bin/
```

## How it works

```
~/.skillops/skills/          ← global store (pulled repos live here)
  my-repo/
    auth-agent/SKILL.md
    logging-agent/SKILL.md

my-project/
  .skillops/config.json      ← local project config (commit this)
  .kiro/skills/
    auth-agent → ~/.skillops/skills/my-repo/auth-agent   (symlink)
  .claude/skills/
    auth-agent → ~/.skillops/skills/my-repo/auth-agent   (symlink)
```

## Quick start

```bash
# 1. Pull a skill repo into the global store
skillops pull https://github.com/org/my-skills-repo

# 2. In your project, declare which IDEs to use
skillops init

# 3. Link skills into the active IDEs
skillops add

# 4. Check what's linked
skillops status
```

## Command reference

### Project commands

| Command | Description |
|---|---|
| `skillops init` | Declare which IDE tools are active in this project |
| `skillops add [skill]` | Link a skill into the project's active IDE tools |
| `skillops remove [skill]` | Unlink a skill from the project's IDE tools |
| `skillops status` | Show current skill and IDE state of this project |
| `skillops sync` | Restore all symlinks declared in the local config |

**`skillops add` flags:**
- `--all` — link into all active tools
- `--tool <name>` — comma-separated list of tools to target (e.g. `--tool kiro,cursor`)

**`skillops remove` flags:**
- `--all` — unlink from all active tools
- `--tool <name>` — comma-separated list of tools to unlink from

### Skill commands

| Command | Description |
|---|---|
| `skillops pull <url>` | Pull a skill repository from GitHub |
| `skillops pull <url> --skill <name>` | Pull a specific skill from a repository |
| `skillops list` | List all downloaded skills |
| `skillops update` | Update all pulled skill repositories |
| `skillops update --skill <name>` | Update a specific skill |
| `skillops config add-agentic -n <name> -p <path>` | Register a new IDE type globally |
| `skillops config update-agentic -n <name> -p <path>` | Update an existing IDE mapping |
| `skillops config remove-agentic -n <name>` | Remove a registered IDE mapping |

## Supported IDEs (defaults)

| IDE | Skills path |
|---|---|
| `claude-code` | `.claude/skills` |
| `cursor` | `.cursor/skills` |
| `windsurf` | `.windsurf/skills` |
| `kiro` | `.kiro/skills` |
| `gemini-cli` | `.gemini/skills` |
| `goose` | `.goose/skills` |
| `github-copilot` | `.github/skills` |
| `opencode` | `.agents/skills` |
| `antigravity` | `.agent/skills` |

Add any custom IDE with `skillops config add-agentic`.

## Configuration

| File | Purpose |
|---|---|
| `~/.skillops/config/agentics.yaml` | Global IDE registry |
| `~/.skillops/config/settings.yaml` | Registry sources for auto-pull |
| `~/.skillops/skills/` | Global skill store |
| `.skillops/config.json` | Local project config (per-project, commit to git) |

### Registry support (optional)

Configure skill registries in `~/.skillops/config/settings.yaml` to enable `skillops sync` to auto-pull missing skills:

```yaml
registries:
  - url: https://github.com/your-org
    name: company-internal
```

## Upgrading from v1

If you used skillops v1, run these two commands in each project:

```bash
skillops init   # declare which IDEs this project uses
skillops sync   # restore your skill links
```

Your global skill store (`~/.skillops/skills/`) and existing symlinks are untouched.

---

A skill is any directory containing a `SKILL.md` file.
