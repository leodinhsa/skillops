package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"skillops/internal/config"
	"skillops/internal/skills"

	tea "github.com/charmbracelet/bubbletea"
)

// addState represents the current screen in the add TUI flow.
type addState int

const (
	addStateSkillSelect addState = iota
	addStateToolSelect
	addStateConfirm
)

// addItem is a skill entry shown in the skill selection screen.
type addItem struct {
	identity string // "repo/skill"
	repoName string
	path     string // absolute path in global store
}

// addModel drives the `skillops add` TUI (no-args flow).
// States: SKILL_SELECT → TOOL_SELECT → CONFIRM → done
type addModel struct {
	// Screen 1: skill selection
	skillItems   []addItem
	skillCursor  int
	skillChecked map[int]bool
	skillHeight  int // max visible rows

	// Screen 2: tool selection
	activeTools []string
	toolCursor  int
	toolChecked map[int]bool

	// Screen 3: confirm
	confirmCursor int

	state    addState
	quitting bool
	err      error
	results  []string // summary lines printed after p.Run()
}

// NewAddModel builds the add TUI model.
// If preselectedSkill is non-empty, skill selection is skipped and that skill is pre-selected.
func NewAddModel(preselectedSkill string) (*addModel, error) {
	allSkills, err := skills.Discover()
	if err != nil {
		return nil, fmt.Errorf("failed to discover skills: %w", err)
	}
	if len(allSkills) == 0 {
		return nil, fmt.Errorf("no skills found. Use 'skillops pull' to download skill repositories")
	}

	// Sort by repo then skill name
	sort.Slice(allSkills, func(i, j int) bool {
		if allSkills[i].RepoName != allSkills[j].RepoName {
			return allSkills[i].RepoName < allSkills[j].RepoName
		}
		return allSkills[i].Name < allSkills[j].Name
	})

	items := make([]addItem, len(allSkills))
	for i, s := range allSkills {
		items[i] = addItem{
			identity: s.Name, // already "repo/skill"
			repoName: s.RepoName,
			path:     s.Path,
		}
	}

	activeTools, err := config.GetActiveTools()
	if err != nil {
		return nil, fmt.Errorf("failed to read active tools: %w", err)
	}
	sort.Strings(activeTools)

	skillChecked := make(map[int]bool)
	startState := addStateSkillSelect

	if preselectedSkill != "" {
		// Pre-select matching skill and skip to tool selection
		for i, item := range items {
			shortName := strings.SplitN(item.identity, "/", 2)
			if len(shortName) == 2 && shortName[1] == preselectedSkill {
				skillChecked[i] = true
			} else if item.identity == preselectedSkill {
				skillChecked[i] = true
			}
		}
		startState = addStateToolSelect
	}

	return &addModel{
		skillItems:   items,
		skillChecked: skillChecked,
		skillHeight:  12,
		activeTools:  activeTools,
		toolChecked:  make(map[int]bool),
		state:        startState,
	}, nil
}

func (m *addModel) Init() tea.Cmd { return nil }

func (m *addModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case addStateSkillSelect:
			switch msg.String() {
			case "ctrl+c", "esc":
				m.quitting = true
				return m, tea.Quit
			case "up", "k":
				if m.skillCursor > 0 {
					m.skillCursor--
				}
			case "down", "j":
				if m.skillCursor < len(m.skillItems)-1 {
					m.skillCursor++
				}
			case " ":
				m.skillChecked[m.skillCursor] = !m.skillChecked[m.skillCursor]
			case "enter":
				// Must have at least one skill selected
				hasSkill := false
				for _, v := range m.skillChecked {
					if v {
						hasSkill = true
						break
					}
				}
				if hasSkill {
					m.state = addStateToolSelect
					m.toolCursor = 0
				}
			}

		case addStateToolSelect:
			switch msg.String() {
			case "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "esc":
				m.state = addStateSkillSelect
				m.toolCursor = 0
			case "up", "k":
				if m.toolCursor > 0 {
					m.toolCursor--
				}
			case "down", "j":
				if m.toolCursor < len(m.activeTools)-1 {
					m.toolCursor++
				}
			case " ":
				m.toolChecked[m.toolCursor] = !m.toolChecked[m.toolCursor]
			case "enter":
				hasTool := false
				for _, v := range m.toolChecked {
					if v {
						hasTool = true
						break
					}
				}
				if hasTool {
					m.state = addStateConfirm
					m.confirmCursor = 0
				}
			}

		case addStateConfirm:
			switch msg.String() {
			case "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "esc":
				m.state = addStateToolSelect
				m.confirmCursor = 0
			case "left", "h":
				m.confirmCursor = 0
			case "right", "l":
				m.confirmCursor = 1
			case "enter":
				if m.confirmCursor == 0 {
					if err := m.applyAdd(); err != nil {
						m.err = err
					}
					m.quitting = true
					return m, tea.Quit
				}
				// No — back to tool select
				m.state = addStateToolSelect
				m.confirmCursor = 0
			}
		}
	}
	return m, nil
}

// selectedSkills returns the checked skill items.
func (m *addModel) selectedSkills() []addItem {
	var out []addItem
	for i, item := range m.skillItems {
		if m.skillChecked[i] {
			out = append(out, item)
		}
	}
	return out
}

// selectedTools returns the checked tool names.
func (m *addModel) selectedTools() []string {
	var out []string
	for i, t := range m.activeTools {
		if m.toolChecked[i] {
			out = append(out, t)
		}
	}
	return out
}

// applyAdd creates symlinks and updates local config.
func (m *addModel) applyAdd() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	for _, skill := range m.selectedSkills() {
		for _, tool := range m.selectedTools() {
			result, err := LinkSkillToTool(cwd, skill.identity, skill.path, tool)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
				continue
			}
			if result != "" {
				m.results = append(m.results, result)
			}
		}
	}
	return nil
}

func (m *addModel) View() string {
	if m.quitting {
		return ""
	}

	switch m.state {
	case addStateSkillSelect:
		return m.viewSkillSelect()
	case addStateToolSelect:
		return m.viewToolSelect()
	case addStateConfirm:
		return m.viewConfirm()
	}
	return ""
}

func (m *addModel) viewSkillSelect() string {
	content := TitleStyle.Render(" ADD SKILL ") + "\n\n"
	content += InfoStyle.Render("Select skills to add (grouped by repo):") + "\n\n"

	// Build a flat list of (index, item) for scrolling
	type row struct {
		idx      int
		item     addItem
		isHeader bool
		repo     string
	}
	var rows []row
	lastRepo := ""
	for i, item := range m.skillItems {
		if item.repoName != lastRepo {
			rows = append(rows, row{isHeader: true, repo: item.repoName})
			lastRepo = item.repoName
		}
		rows = append(rows, row{idx: i, item: item})
	}

	// Find cursor row position (only non-header rows count for cursor)
	cursorRowIdx := -1
	for ri, r := range rows {
		if !r.isHeader && r.idx == m.skillCursor {
			cursorRowIdx = ri
			break
		}
	}

	// Scrolling window over rows
	start := 0
	end := len(rows)
	h := m.skillHeight + m.skillHeight/2 // account for headers taking space
	if len(rows) > h {
		start = cursorRowIdx - h/2
		if start < 0 {
			start = 0
		}
		end = start + h
		if end > len(rows) {
			end = len(rows)
			start = end - h
			if start < 0 {
				start = 0
			}
		}
	}

	if start > 0 {
		content += DimStyle.Render("   ↑ scroll up") + "\n"
	}

	for ri := start; ri < end; ri++ {
		r := rows[ri]
		if r.isHeader {
			content += "\n" + HeaderStyle.Render("📦 "+r.repo) + "\n"
			continue
		}

		checkbox := CheckboxStyle.Render("○")
		if m.skillChecked[r.idx] {
			checkbox = CheckboxStyle.Render("◉")
		}

		cursor := "  "
		style := NormalStyle
		if r.idx == m.skillCursor {
			cursor = "> "
			style = SelectedStyle
		}

		shortName := strings.SplitN(r.item.identity, "/", 2)
		displayName := r.item.identity
		if len(shortName) == 2 {
			displayName = shortName[1]
		}
		content += fmt.Sprintf("%s%s %s\n", cursor, checkbox, style.Render(displayName))
	}

	if end < len(rows) {
		content += DimStyle.Render("   ↓ scroll down") + "\n"
	} else {
		content += "\n"
	}

	content += HelpStyle.Render("\n ↑/↓: navigate • space: toggle • enter: next • esc: quit")
	return BorderStyle.Render(content) + "\n"
}

func (m *addModel) viewToolSelect() string {
	content := TitleStyle.Render(" SELECT TOOLS ") + "\n\n"
	content += InfoStyle.Render("Select target tools:") + "\n\n"

	if len(m.activeTools) == 0 {
		content += DimStyle.Render("  No active tools. Run 'skillops init' first.") + "\n"
	} else {
		for i, tool := range m.activeTools {
			checkbox := CheckboxStyle.Render("○")
			if m.toolChecked[i] {
				checkbox = CheckboxStyle.Render("◉")
			}

			cursor := "  "
			style := NormalStyle
			if i == m.toolCursor {
				cursor = "> "
				style = SelectedStyle
			}

			content += fmt.Sprintf("%s%s %s\n", cursor, checkbox, style.Render(tool))
		}
	}

	content += HelpStyle.Render("\n ↑/↓: navigate • space: toggle • enter: confirm • esc: back")
	return BorderStyle.Render(content) + "\n"
}

func (m *addModel) viewConfirm() string {
	content := TitleStyle.Render(" CONFIRM ADD ") + "\n\n"

	for _, skill := range m.selectedSkills() {
		shortName := strings.SplitN(skill.identity, "/", 2)
		name := skill.identity
		if len(shortName) == 2 {
			name = shortName[1]
		}
		for _, tool := range m.selectedTools() {
			content += SuccessStyle.Render(fmt.Sprintf("  + %s → %s", name, tool)) + "\n"
		}
	}

	content += "\nApply these changes?\n\n"

	choices := []string{"Yes, add", "No, back"}
	for i, choice := range choices {
		style := NormalStyle
		if i == m.confirmCursor {
			style = SelectedStyle
		}
		content += style.Render(choice) + "  "
	}

	content += HelpStyle.Render("\n\n ←/→: navigate • enter: select • esc: back")
	return BorderStyle.Render(content) + "\n"
}

// RunAdd launches the add TUI (no-args flow).
func RunAdd(preselectedSkill string) error {
	m, err := NewAddModel(preselectedSkill)
	if err != nil {
		return err
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	res := finalModel.(*addModel)
	if res.err != nil {
		return res.err
	}

	if len(res.results) > 0 {
		fmt.Println("\n✨ Skills linked:")
		for _, r := range res.results {
			fmt.Println("  " + r)
		}
	} else {
		fmt.Println("No changes made.")
	}

	return nil
}

// LinkSkillToTool creates a symlink for a skill into a tool's skills directory
// and updates local config. Returns a summary string or an error.
// Conflict detection: if a symlink already exists pointing to a different repo, warn and skip.
func LinkSkillToTool(cwd, identity, skillPath, tool string) (string, error) {
	toolRelPath, err := config.GetAgenticPath(tool)
	if err != nil {
		return "", fmt.Errorf("unknown tool: %s", tool)
	}

	parts := strings.SplitN(identity, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid skill identity: %s", identity)
	}
	shortName := parts[1]

	skillsDir := filepath.Join(cwd, toolRelPath)
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create skills dir for %s: %w", tool, err)
	}

	symlinkPath := filepath.Join(skillsDir, shortName)

	// Conflict detection
	info, err := os.Lstat(symlinkPath)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			existing, readErr := os.Readlink(symlinkPath)
			if readErr == nil {
				if existing == skillPath {
					// Already linked to same target — silent no-op
					return "", nil
				}
				// Different target — conflict
				return "", fmt.Errorf("conflict: %s already linked to a different skill in %s (skipping)", shortName, tool)
			}
		}
		// Non-symlink file exists — skip with warning
		return "", fmt.Errorf("conflict: %s exists and is not a symlink in %s (skipping)", shortName, tool)
	}

	if err := os.Symlink(skillPath, symlinkPath); err != nil {
		return "", fmt.Errorf("failed to create symlink for %s in %s: %w", shortName, tool, err)
	}

	if err := config.AddSkillToTool(tool, identity); err != nil {
		return "", fmt.Errorf("symlink created but failed to update local config: %w", err)
	}

	return fmt.Sprintf("+ %s → %s", shortName, tool), nil
}
