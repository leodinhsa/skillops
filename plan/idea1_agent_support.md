# Plan: Idea 1 - Agent Support Upgrade

## Objective
Upgrade `skillops` to support a wider range of AI agents as defined in the Vercel Skills README.

## Proposed Changes
Update `internal/config/config.go` to include the following agents and their project paths in the `defaultAgentics` map:

| Agent | `--agent` (Key) | Project Path (Value) |
|-------|-----------|--------------|
| Amp, Kimi Code CLI, Replit, Universal | `universal` | `.agents/skills/` |
| Antigravity | `antigravity` | `.agent/skills/` |
| Augment | `augment` | `.augment/skills/` |
| Claude Code | `claude-code` | `.claude/skills/` |
| OpenClaw | `openclaw` | `skills/` |
| Cline | `cline` | `.agents/skills/` |
| CodeBuddy | `codebuddy` | `.codebuddy/skills/` |
| Codex | `codex` | `.agents/skills/` |
| Command Code | `command-code` | `.commandcode/skills/` |
| Continue | `continue` | `.continue/skills/` |
| Cortex Code | `cortex` | `.cortex/skills/` |
| Crush | `crush` | `.crush/skills/` |
| Cursor | `cursor` | `.agents/skills/` |
| Droid | `droid` | `.factory/skills/` |
| Gemini CLI | `gemini-cli` | `.agents/skills/` |
| GitHub Copilot | `github-copilot` | `.agents/skills/` |
| Goose | `goose` | `.goose/skills/` |
| Junie | `junie` | `.junie/skills/` |
| iFlow CLI | `iflow-cli` | `.iflow/skills/` |
| Kilo Code | `kilo` | `.kilocode/skills/` |
| Kiro CLI | `kiro-cli` | `.kiro/skills/` |
| Kode | `kode` | `.kode/skills/` |
| MCPJam | `mcpjam` | `.mcpjam/skills/` |
| Mistral Vibe | `mistral-vibe` | `.vibe/skills/` |
| Mux | `mux` | `.mux/skills/` |
| OpenCode | `opencode` | `.agents/skills/` |
| OpenHands | `openhands` | `.openhands/skills/` |
| Pi | `pi` | `.pi/skills/` |
| Qoder | `qoder` | `.qoder/skills/` |
| Qwen Code | `qwen-code` | `.qwen/skills/` |
| Roo Code | `roo` | `.roo/skills/` |
| Trae | `trae` | `.trae/skills/` |
| Windsurf | `windsurf` | `.windsurf/skills/` |
| Zencoder | `zencoder` | `.zencoder/skills/` |
| Neovate | `neovate` | `.neovate/skills/` |
| Pochi | `pochi` | `.pochi/skills/` |
| AdaL | `adal` | `.adal/skills/` |

## TUI Improvements for Agent Selection
With ~40 supported agents, the `skillops agentic` selection TUI will exceed most terminal heights. 

### Proposed TUI Solution:
1. **Scrolling & Viewport**: Implement a viewport for `checklistModel` (similar to the skill management TUI). Show a fixed number of agents (e.g., 12 items) and allow scrolling with arrows.
2. **Visual Feedback**: Add "↑ more" and "↓ more" indicators or a scrollbar-like indicator to show that the list continues.
3. **Filtering**: Add a `textinput` field to allow users to type to filter the list of agents (e.g., typing "claude" shows only Claude related agents).

## Implementation Steps
1. Modify `internal/config/config.go`: Update `defaultAgentics`.
2. Modify `internal/tui/tui.go`: 
    - Add `height`, `start`, `end` logic to `checklistModel`.
    - Integrate `textinput` for filtering in `checklistModel`.
    - Update `checklistModel.View()` to render the search bar and scrolling window.
