# 🛠 Skill Ops

Skill Ops is a lightweight, high-performance Go CLI utility designed to manage AI agent skills using a **symlink-first** approach. It allows developers to pull skill repositories and selectively enable/disable them for specific Agentic IDEs (Claude, Antigravity, OpenCode, etc.) without file duplication.

## 🚀 Key Features

- **Symlink Management**: Skills are stored centrally and symlinked to project folders.
- **Agentic Checklist**: Manage multiple IDE environments simultaneously in your project root via a checklist TUI.
- **Interactive Skill TUI**: A premium, rich interactive interface for toggling skills and managing IDE paths.
- **Premium Help Output**: A structured, colored `--help` experience.
- **Safety First**: Protected against accidental recursive deletions and invalid path configurations.
- **Flexible Config**: Consolidated configuration in `~/.skillops/config/agentics.yaml`.

## 📦 Installation

### Homebrew (macOS & Linux)

```bash
brew tap leodinhsa/skillops
brew install skillops
```

### Manual (from source)

```bash
go build -o skillops main.go
mv skillops /usr/local/bin/
```

## 📁 Command Reference

### Project Configuration
- **`skillops agentic`**: Open a checklist TUI to enable/disable Agentic IDE environments in your project root.
- **`skillops agentic manage <name>`**: Interactively manage skills or remove the environment for a specific agentic.
- **`skillops agentic remove-skill <agent> <skill>`**: Remove a specific skill symlink.
- **`skillops agentic remove-skills <agent>`**: Remove all skill symlinks for an agentic.

### Skill Management
- **`skillops pull <url>`**: Download a new skill repository from GitHub.
- **`skillops list`**: Show all downloaded skill names and their status.
- **`skillops remove <name>`**: Safely delete a pulled skill repository (checks for active links).
- **`skillops remove-all`**: Clear all local skill repositories.

### Agentic Configuration
- **`skillops config add-agentic -n <name> -p <path>`**: Register a new IDE type globally.
- **`skillops config update-agentic -n <name>`**: Update an existing IDE mapping (interactive path).
- **`skillops config remove-agentic -n <name>`**: Remove a registered IDE mapping.

## ⚙️ Configuration

- **All Config**: `~/.skillops/config/agentics.yaml`

---
**Skill folders are identified by the presence of a `SKILL.md` file.**
