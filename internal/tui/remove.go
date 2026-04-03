package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"skillops/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

// removeState represents the current screen in the remove TUI flow.
type removeState int

const (
	removeStateSkillSelect removeState = iota
	removeStateToolSelect
	removeStateConfirm
)

// removeModel drives the `skillops remove` TUI (no-args flow).
// States: SKILL_SELECT → TOOL_SELECT → CONFIRM → done
type removeModel struct {
	// Screen 1: skill selection — only skills present in local config
	skillIdentities []string // "repo/skill"
	skillCursor     int
	selectedSkill   string // single selection

	// Screen 2: tool selection — only tools that have the selected skill
	eligibleTools []string
	toolCursor    int
	toolChecked   map[int]bool

	// Screen 3: confirm
	confirmCursor int

	state    removeState
	quitting bool
	err      error
	results  []string
}

// NewRemoveModel builds the remove TUI model.
// If preselectedSkill is non-empty, skill selection is skipped.
func NewRemoveModel(preselectedSkill string) (*removeModel, error) {
	cfg, err := config.ReadLocalConfig()
	if err != nil {
		return nil, err
	}

	// Collect all unique skill identities across all tools
	identitySet := make(map[string]bool)
	for _, skills := range cfg.Tools {
		for _, s := range skills {
			identitySet[s] = true
		}
	}

	identities := make([]string, 0, len(identitySet))
	for id := range identitySet {
		identities = append(identities, id)
	}
	sort.Strings(identities)

	m := &removeModel{
		skillIdentities: identities,
		toolChecked:     make(map[int]bool),
		state:           removeStateSkillSelect,
	}

	if preselectedSkill != "" {
		// Find matching identity
		for _, id := range identities {
			parts := strings.SplitN(id, "/", 2)
			if id == preselectedSkill || (len(parts) == 2 && parts[1] == preselectedSkill) {
				m.selectedSkill = id
				break
			}
		}
		if m.selectedSkill == "" {
			return nil, fmt.Errorf("skill '%s' not found in local config", preselectedSkill)
		}
		if err := m.loadEligibleTools(cfg); err != nil {
			return nil, err
		}
		m.state = removeStateToolSelect
	}

	return m, nil
}

// loadEligibleTools populates eligibleTools based on the selectedSkill.
func (m *removeModel) loadEligibleTools(cfg config.LocalConfig) error {
	var tools []string
	for tool, skillList := range cfg.Tools {
		for _, s := range skillList {
			if s == m.selectedSkill {
				tools = append(tools, tool)
				break
			}
		}
	}
	sort.Strings(tools)
	m.eligibleTools = tools
	m.toolChecked = make(map[int]bool)
	return nil
}

func (m *removeModel) Init() tea.Cmd { return nil }

func (m *removeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case removeStateSkillSelect:
			switch msg.String() {
			case "ctrl+c", "esc":
				m.quitting = true
				return m, tea.Quit
			case "up", "k":
				if m.skillCursor > 0 {
					m.skillCursor--
				}
			case "down", "j":
				if m.skillCursor < len(m.skillIdentities)-1 {
					m.skillCursor++
				}
			case "enter":
				if len(m.skillIdentities) == 0 {
					break
				}
				m.selectedSkill = m.skillIdentities[m.skillCursor]
				cfg, err := config.ReadLocalConfig()
				if err != nil {
					m.err = err
					m.quitting = true
					return m, tea.Quit
				}
				if err := m.loadEligibleTools(cfg); err != nil {
					m.err = err
					m.quitting = true
					return m, tea.Quit
				}
				m.state = removeStateToolSelect
				m.toolCursor = 0
			}

		case removeStateToolSelect:
			switch msg.String() {
			case "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "esc":
				m.state = removeStateSkillSelect
				m.toolCursor = 0
			case "up", "k":
				if m.toolCursor > 0 {
					m.toolCursor--
				}
			case "down", "j":
				if m.toolCursor < len(m.eligibleTools)-1 {
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
					m.state = removeStateConfirm
					m.confirmCursor = 0
				}
			}

		case removeStateConfirm:
			switch msg.String() {
			case "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "esc":
				m.state = removeStateToolSelect
				m.confirmCursor = 0
			case "left", "h":
				m.confirmCursor = 0
			case "right", "l":
				m.confirmCursor = 1
			case "enter":
				if m.confirmCursor == 0 {
					if err := m.applyRemove(); err != nil {
						m.err = err
					}
					m.quitting = true
					return m, tea.Quit
				}
				m.state = removeStateToolSelect
				m.confirmCursor = 0
			}
		}
	}
	return m, nil
}

// selectedTools returns the checked tool names.
func (m *removeModel) selectedTools() []string {
	var out []string
	for i, t := range m.eligibleTools {
		if m.toolChecked[i] {
			out = append(out, t)
		}
	}
	return out
}

// applyRemove removes symlinks and updates local config.
func (m *removeModel) applyRemove() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	parts := strings.SplitN(m.selectedSkill, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid skill identity: %s", m.selectedSkill)
	}
	shortName := parts[1]

	for _, tool := range m.selectedTools() {
		result, err := UnlinkSkillFromTool(cwd, m.selectedSkill, shortName, tool)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			continue
		}
		if result != "" {
			m.results = append(m.results, result)
		}
	}
	return nil
}

func (m *removeModel) View() string {
	if m.quitting {
		return ""
	}

	switch m.state {
	case removeStateSkillSelect:
		return m.viewSkillSelect()
	case removeStateToolSelect:
		return m.viewToolSelect()
	case removeStateConfirm:
		return m.viewConfirm()
	}
	return ""
}

func (m *removeModel) viewSkillSelect() string {
	content := TitleStyle.Render(" REMOVE SKILL ") + "\n\n"
	content += InfoStyle.Render("Select a skill to unlink:") + "\n\n"

	if len(m.skillIdentities) == 0 {
		content += DimStyle.Render("  No skills linked in this project.") + "\n"
	} else {
		for i, id := range m.skillIdentities {
			cursor := "  "
			style := NormalStyle
			if i == m.skillCursor {
				cursor = "> "
				style = SelectedStyle
			}
			parts := strings.SplitN(id, "/", 2)
			display := id
			if len(parts) == 2 {
				display = fmt.Sprintf("%s  %s", parts[1], DimStyle.Render("("+parts[0]+")"))
			}
			content += fmt.Sprintf("%s%s\n", cursor, style.Render(display))
		}
	}

	content += HelpStyle.Render("\n ↑/↓: navigate • enter: select • esc: quit")
	return BorderStyle.Render(content) + "\n"
}

func (m *removeModel) viewToolSelect() string {
	parts := strings.SplitN(m.selectedSkill, "/", 2)
	shortName := m.selectedSkill
	if len(parts) == 2 {
		shortName = parts[1]
	}

	content := TitleStyle.Render(" SELECT TOOLS ") + "\n\n"
	content += InfoStyle.Render(fmt.Sprintf("Unlink '%s' from which tools?", shortName)) + "\n\n"

	if len(m.eligibleTools) == 0 {
		content += DimStyle.Render("  No tools have this skill linked.") + "\n"
	} else {
		for i, tool := range m.eligibleTools {
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

func (m *removeModel) viewConfirm() string {
	content := TitleStyle.Render(" CONFIRM REMOVE ") + "\n\n"

	parts := strings.SplitN(m.selectedSkill, "/", 2)
	shortName := m.selectedSkill
	if len(parts) == 2 {
		shortName = parts[1]
	}

	for _, tool := range m.selectedTools() {
		content += HeaderStyle.Render(fmt.Sprintf("  - %s from %s", shortName, tool)) + "\n"
	}

	content += "\nApply these changes?\n\n"

	choices := []string{"Yes, remove", "No, back"}
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

// RunRemove launches the remove TUI (no-args flow).
func RunRemove(preselectedSkill string) error {
	m, err := NewRemoveModel(preselectedSkill)
	if err != nil {
		return err
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	res := finalModel.(*removeModel)
	if res.err != nil {
		return res.err
	}

	if len(res.results) > 0 {
		fmt.Println("\n✨ Skills unlinked:")
		for _, r := range res.results {
			fmt.Println("  " + r)
		}
	} else {
		fmt.Println("No changes made.")
	}

	return nil
}

// UnlinkSkillFromTool removes the symlink for shortName in the tool's skills dir
// and updates local config. Never touches the global store.
// Returns a summary string or an error.
func UnlinkSkillFromTool(cwd, identity, shortName, tool string) (string, error) {
	toolRelPath, err := config.GetAgenticPath(tool)
	if err != nil {
		return "", fmt.Errorf("unknown tool: %s", tool)
	}

	symlinkPath := filepath.Join(cwd, toolRelPath, shortName)

	info, err := os.Lstat(symlinkPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Idempotent — symlink already gone, still update config
			if cfgErr := config.RemoveSkillFromTool(tool, identity); cfgErr != nil {
				return "", cfgErr
			}
			return "", nil
		}
		return "", fmt.Errorf("failed to stat %s: %w", symlinkPath, err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		// Not a symlink — skip with warning, do not os.Remove
		fmt.Fprintf(os.Stderr, "Warning: %s is not a symlink, skipping removal\n", symlinkPath)
		return "", nil
	}

	if err := os.Remove(symlinkPath); err != nil {
		return "", fmt.Errorf("failed to remove symlink %s: %w", symlinkPath, err)
	}

	if err := config.RemoveSkillFromTool(tool, identity); err != nil {
		return "", fmt.Errorf("symlink removed but failed to update local config: %w", err)
	}

	return fmt.Sprintf("- %s from %s", shortName, tool), nil
}
