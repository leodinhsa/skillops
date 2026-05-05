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

// initState represents the current screen in the init TUI flow.
type initState int

const (
	initStateChecklist initState = iota
	initStateConfirm
	initStateApply
)

// initModel drives the `skillops init` TUI.
// States: CHECKLIST → CONFIRM → APPLY
type initModel struct {
	// All tool names from global config, sorted
	allTools []string
	// Relative skills paths per tool
	toolPaths map[string]string
	// Which tools are checked (by index into allTools)
	checked map[int]bool
	// Which tools were already active before this run (from local config)
	previouslyActive map[string]bool

	cursor   int
	height   int // max visible rows in checklist
	state    initState
	quitting bool
	err      error
	applied  []string
}

// NewInitModel creates an initModel pre-checked from the existing local config.
func NewInitModel() (*initModel, error) {
	agentics, err := config.GetAgentics()
	if err != nil {
		return nil, fmt.Errorf("failed to read global config: %w", err)
	}

	names := make([]string, 0, len(agentics))
	for k := range agentics {
		names = append(names, k)
	}
	sort.Strings(names)

	// Read existing local config to pre-check active tools
	previouslyActive := make(map[string]bool)
	if activeTools, err := config.GetActiveTools(); err == nil {
		for _, t := range activeTools {
			previouslyActive[t] = true
		}
	}
	// If local config doesn't exist yet, previouslyActive stays empty

	checked := make(map[int]bool)
	for i, name := range names {
		if previouslyActive[name] {
			checked[i] = true
		}
	}

	return &initModel{
		allTools:         names,
		toolPaths:        agentics,
		checked:          checked,
		previouslyActive: previouslyActive,
		state:            initStateChecklist,
		height:           12,
	}, nil
}

func (m *initModel) Init() tea.Cmd { return nil }

func (m *initModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Reserve ~16 lines for title, info, help, scroll indicators, and border
		m.height = max(3, msg.Height-16)
		return m, nil
	case tea.KeyMsg:
		switch m.state {
		case initStateChecklist:
			switch msg.String() {
			case "ctrl+c", "esc":
				m.quitting = true
				return m, tea.Quit
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.allTools)-1 {
					m.cursor++
				}
			case " ":
				m.checked[m.cursor] = !m.checked[m.cursor]
			case "enter":
				m.state = initStateConfirm
				m.cursor = 0
			}

		case initStateConfirm:
			switch msg.String() {
			case "ctrl+c", "esc":
				// Go back to checklist
				m.state = initStateChecklist
				m.cursor = 0
			case "left", "h":
				m.cursor = 0
			case "right", "l":
				m.cursor = 1
			case "enter":
				if m.cursor == 0 {
					// Yes — apply
					if err := m.applyChanges(); err != nil {
						m.err = err
					}
					m.quitting = true
					return m, tea.Quit
				}
				// No — back to checklist
				m.state = initStateChecklist
				m.cursor = 0
			}
		}
	}
	return m, nil
}

// selectedToolNames returns the names of all checked tools.
func (m *initModel) selectedToolNames() []string {
	var selected []string
	for i, name := range m.allTools {
		if m.checked[i] {
			selected = append(selected, name)
		}
	}
	return selected
}

// applyChanges writes local config and manages skill directories / symlinks.
func (m *initModel) applyChanges() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	selected := m.selectedToolNames()

	// Determine added and removed tools
	selectedSet := make(map[string]bool, len(selected))
	for _, t := range selected {
		selectedSet[t] = true
	}

	// 1. Update local config
	if err := config.SetActiveTools(selected); err != nil {
		return fmt.Errorf("failed to update local config: %w", err)
	}

	// 2. For newly added tools: create skills directory
	for _, name := range selected {
		if !m.previouslyActive[name] {
			relPath := m.toolPaths[name]
			skillsDir := filepath.Join(cwd, relPath)
			if err := os.MkdirAll(skillsDir, 0755); err != nil {
				return fmt.Errorf("failed to create skills dir for %s: %w", name, err)
			}
			m.applied = append(m.applied, fmt.Sprintf("+ %s", name))
		}
	}

	// 3. For deselected tools: remove symlinks from skills dir (not the root dir)
	for _, name := range m.allTools {
		if m.previouslyActive[name] && !selectedSet[name] {
			relPath := m.toolPaths[name]
			skillsDir := filepath.Join(cwd, relPath)

			if err := removeSymlinksInDir(skillsDir, cwd, relPath); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to clean symlinks for %s: %v\n", name, err)
			}
			m.applied = append(m.applied, fmt.Sprintf("- %s", name))
		}
	}

	return nil
}

// removeSymlinksInDir removes only symlinks inside skillsDir.
// It validates that each removal path is within cwd/relPath before removing.
func removeSymlinksInDir(skillsDir, cwd, relPath string) error {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // nothing to clean
		}
		return err
	}

	// Compute the canonical allowed prefix
	allowedPrefix := filepath.Clean(filepath.Join(cwd, relPath))

	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink == 0 {
			continue // skip non-symlinks
		}
		target := filepath.Join(skillsDir, entry.Name())

		// Safety: ensure the path is within the allowed skills dir
		cleanTarget := filepath.Clean(target)
		if !strings.HasPrefix(cleanTarget, allowedPrefix+string(filepath.Separator)) &&
			cleanTarget != allowedPrefix {
			fmt.Fprintf(os.Stderr, "Safety: skipping removal of %s (outside allowed path)\n", cleanTarget)
			continue
		}

		if err := os.Remove(target); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove symlink %s: %v\n", target, err)
		}
	}
	return nil
}

func (m *initModel) View() string {
	if m.quitting {
		return ""
	}

	switch m.state {
	case initStateChecklist:
		return m.viewChecklist()
	case initStateConfirm:
		return m.viewConfirm()
	}
	return ""
}

func (m *initModel) viewChecklist() string {
	content := TitleStyle.Render(" INIT PROJECT ") + "\n\n"
	content += InfoStyle.Render("Select the IDE tools active in this project:") + "\n\n"

	// Calculate scrolling window
	start := 0
	end := len(m.allTools)
	if len(m.allTools) > m.height {
		start = m.cursor - (m.height / 2)
		if start < 0 {
			start = 0
		}
		end = start + m.height
		if end > len(m.allTools) {
			end = len(m.allTools)
			start = end - m.height
		}
	}

	if start > 0 {
		content += DimStyle.Render("   ↑ scroll up") + "\n"
	}

	for i := start; i < end; i++ {
		name := m.allTools[i]
		checkbox := CheckboxStyle.Render("○")
		if m.checked[i] {
			checkbox = CheckboxStyle.Render("◉")
		}

		cursor := "  "
		style := NormalStyle
		if i == m.cursor {
			cursor = "> "
			style = SelectedStyle
		}

		content += fmt.Sprintf("%s%s %s\n", cursor, checkbox, style.Render(name))
	}

	if end < len(m.allTools) {
		content += DimStyle.Render("   ↓ scroll down") + "\n"
	} else {
		content += "\n"
	}

	content += HelpStyle.Render("\n ↑/↓: navigate • space: toggle • enter: confirm • esc: quit")
	return BorderStyle.Render(content) + "\n"
}

func (m *initModel) viewConfirm() string {
	content := TitleStyle.Render(" CONFIRM CHANGES ") + "\n\n"

	selectedSet := make(map[string]bool)
	for i, name := range m.allTools {
		if m.checked[i] {
			selectedSet[name] = true
		}
	}

	hasChanges := false
	for _, name := range m.allTools {
		wasActive := m.previouslyActive[name]
		isSelected := selectedSet[name]
		if wasActive != isSelected {
			hasChanges = true
			break
		}
	}

	if !hasChanges {
		content += InfoStyle.Render("No changes to apply.") + "\n\n"
	} else {
		for _, name := range m.allTools {
			wasActive := m.previouslyActive[name]
			isSelected := selectedSet[name]
			if isSelected && !wasActive {
				content += SuccessStyle.Render(fmt.Sprintf("  + %s", name)) + "\n"
			} else if !isSelected && wasActive {
				content += HeaderStyle.Render(fmt.Sprintf("  - %s", name)) + "\n"
			}
		}
		content += "\n"
	}

	content += "Apply these changes?\n\n"

	choices := []string{"Yes, apply", "No, back"}
	for i, choice := range choices {
		style := NormalStyle
		if i == m.cursor {
			style = SelectedStyle
		}
		content += style.Render(choice) + "  "
	}

	content += HelpStyle.Render("\n\n ←/→: navigate • enter: select • esc: back")
	return BorderStyle.Render(content) + "\n"
}

// RunInit launches the init TUI and prints a summary after completion.
func RunInit() error {
	m, err := NewInitModel()
	if err != nil {
		return err
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	res := finalModel.(*initModel)
	if res.err != nil {
		return res.err
	}

	if len(res.applied) > 0 {
		fmt.Println("\n✨ Project tools updated:")
		for _, msg := range res.applied {
			fmt.Println("  " + msg)
		}
	} else {
		fmt.Println("No changes made.")
	}

	return nil
}
