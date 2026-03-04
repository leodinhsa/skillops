# SkillOps: Project Memory & Development Guidelines

This document serves as the primary knowledge base for the `skillops` project. It captures the project's architecture, design philosophy, and critical development rules to ensure consistency and prevent regressions in future development cycles.

---

## 🎯 Project Overview
`skillops` is a CLI tool designed to manage "skills" (modular AI capabilities/scripts) for various Agentic IDEs (like Claude Desktop, Windsurf, Roo Code, etc.).
- **Core Problem**: Each IDE has its own skill/tool directory. Syncing a shared repository of skills across these IDEs manually is tedious and error-prone.
- **Solution**: A central `skills` directory (global) and a symbolic linking system that maps specific skills to the correct locations for each IDE.

---

## 🏗️ Technical Architecture

### 1. Directory Structure
- `cmd/`: CLI entry points using `cobra`.
- `internal/config/`: Manages global configuration (`~/.skillops/config.json`) and project-specific activation.
- `internal/skills/`: Handles skill discovery from the global skills directory.
- `internal/symlink/`: Core logic for creating/removing symlinks and detecting existing links.
- `internal/tui/`: High-fidelity terminal interfaces using `bubbletea` and `lipgloss`.

### 2. Global vs. Local State
- **Global Config**: Stores mappings of "Agentic IDE Name" to "Relative Path to Skills Folder".
- **Local Project**: A `.skillops/` directory or specific local markers (defined by paths in global config) determine if an agentic environment is "active" for the current workspace.

---

## 🎨 TUI Design System (The "Rich TUI" Rulebook)
All new commands MUST follow these visual and interaction patterns defined in `internal/tui/styles.go`.

### 1. Color Palette
- **Primary (Pink)**: `#F92672` (Titles, focus accents).
- **Secondary (Purple)**: `#AE81FF` (Borders, selection backgrounds).
- **Success (Green)**: `#A6E22E` (Emojis, success text).
- **Muted (Dimmed)**: `#6272A4` (Help text, secondary info).

### 2. Component Standards
- **Titles**: Centered title bar with `TitleStyle`.
- **Borders**: All main TUIs must be wrapped in `BorderStyle`.
- **Cursors**: Use `>` for navigation. Selected items in menus should have a Purple background with White text.

### 3. Clean Exit Rule (CRITICAL)
To avoid residual "ghost" borders in the terminal:
- **State**: TUI models must have a `quitting` boolean flag.
- **Update**: Set `m.quitting = true` before returning `tea.Quit`.
- **View**: If `m.quitting` is true, return an **empty string** `""`. 
- **Feedback**: Print final success messages or summaries using `fmt.Println` *after* the `p.Run()` call returns in the command's entry point.

---

## 🛠️ Interaction Patterns

### 1. The "Confirmation First" Rule
Any destructive action (deleting skills) or bulk sync (applying checklist changes) MUST be preceded by a confirmation TUI (`confirmModel`).
- **Checklists**: When managing multiple items, the `Enter` key should trigger a summary screen showing what will be added (`+`) and removed (`-`).

### 2. Path Safety
- Never perform `os.RemoveAll` on root directories (`/`, `~`, or the current working directory).
- Always validate that a path is "within" a managed agentic directory before unlinking/deleting.

---

## 🚀 Future Roadmap & Consistency
- **New Agentics**: To add a new IDE, update `internal/config/default_agentics.go` (if any) or use `skillops config add-agentic`.
- **Skill Naming**: Support both short names (e.g., `logger`) and repo-prefixed names (e.g., `brand/logger`) for discovery and symlinking.
- **Autocompletion**: Ensure `agentic manage <name>` only suggests agentics active in the current project root.

---

*Last Updated: 2026-03-04*
