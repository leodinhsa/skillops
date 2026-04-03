# Plan: Idea 4 - List TUI Refactor

## Objective
Refactor the `skillops list` TUI to group skills by repository, support scrolling for large numbers of skills, and include filtering.

## Proposed Changes

### 1. Grouping Logic
- In `internal/tui/list.go`, sort the `allSkills` slice by `RepoName` before passing it to the model.
- Modify the `View()` function to detect when the `RepoName` changes and insert a decorative header for each repository.

### 2. Scrolling & Viewport Implementation
- Adopt the scrolling logic from `internal/tui/tui.go` (Skill Management TUI).
- Introduce a `viewportHeight` (e.g., 15 items).
- Maintain a `start` and `end` index for the visible window.
- **Dynamic Help**: Update the help text to show "scrolling active" when items are hidden.

### 3. Search/Filter Integration
- **Proposal**: Add a `textinput` field (using `bubbles/textinput`) at the top of the `list` TUI. 
- As the user types, the list narrows down to matches in either `SkillName` or `RepoName`.
- This handles the "too many skills" problem by allowing quick discovery.

### 4. Aesthetics
- Use `RepoHeaderStyle` (e.g., Purple background, White text) for repo titles.
- Use `Lipgloss` to ensure the borders and titles remain consistent with the rest of the app.
- Show "Total: X repositories, Y skills" in the top info bar.

## Implementation Steps
1. Update `internal/tui/list.go`:
    - Add `textinput.Model` to `listModel`.
    - Implement filtering logic in `Update()`.
    - Update `View()` to render search bar, repo headers, and the scrolled skill list.
