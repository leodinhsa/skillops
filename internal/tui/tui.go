package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"skillops/internal/config"
	"skillops/internal/skills"
	"skillops/internal/symlink"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Skill Selection TUI ---

type model struct {
	skills        []skills.Skill
	selected      map[int]bool
	agentPath     string
	enabledSkills map[string]bool
	filtered      []skills.Skill
	cursor        int
	quitting      bool
	height        int // Number of visible items
	agentName     string
	editingPath   bool
	pathInput     textinput.Model
	filterInput   textinput.Model
	confirming    bool
	confirmCursor int
}

func New(agentName string) (*model, error) {
	agentPath, err := config.GetAgenticPath(agentName)
	if err != nil {
		return nil, err
	}

	// Get all skills
	allSkills, err := skills.Discover()
	if err != nil {
		return nil, err
	}

	// Check for empty skills slice
	if len(allSkills) == 0 {
		return nil, fmt.Errorf("no skills discovered. Please pull a skill first using 'skillops pull'")
	}

	// Sort by repo name for grouping
	sort.Slice(allSkills, func(i, j int) bool {
		if allSkills[i].RepoName != allSkills[j].RepoName {
			return allSkills[i].RepoName < allSkills[j].RepoName
		}
		return allSkills[i].Name < allSkills[j].Name
	})

	// Get enabled skills in agent path
	enabled, err := symlink.GetEnabledSkills(agentPath)
	if err != nil {
		return nil, err
	}

	// Mark enabled skills as selected by default
	selected := make(map[int]bool)
	for i, skill := range allSkills {
		skillName := skills.GetSkillName(skill)
		if enabled[skillName] {
			selected[i] = true
		}
	}

	ti := textinput.New()
	ti.Placeholder = "Enter new path (e.g. .claude/skills)"
	ti.SetValue(agentPath)

	fi := textinput.New()
	fi.Placeholder = "Search skills..."
	fi.Focus()

	return &model{
		skills:        allSkills,
		filtered:      allSkills,
		selected:      selected,
		agentPath:     agentPath,
		agentName:     agentName,
		enabledSkills: enabled,
		cursor:        0,
		height:        15,
		pathInput:     ti,
		filterInput:   fi,
	}, nil
}

func (m *model) filter(term string) {
	if term == "" {
		m.filtered = m.skills
		return
	}
	var filtered []skills.Skill
	term = strings.ToLower(term)
	for _, s := range m.skills {
		if strings.Contains(strings.ToLower(s.Name), term) || strings.Contains(strings.ToLower(s.RepoName), term) {
			filtered = append(filtered, s)
		}
	}
	m.filtered = filtered
	if m.cursor >= len(m.filtered) {
		m.cursor = 0
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
		}
	}
}

func (m *model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.quitting {
		return m, tea.Quit
	}

	if m.editingPath {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				m.agentPath = m.pathInput.Value()
				if err := config.AddAgentic(m.agentName, m.agentPath); err != nil {
					fmt.Fprintf(os.Stderr, "Error updating path: %v\n", err)
				}
				m.editingPath = false
				// Refresh enabled skills for the new path
				if enabled, err := symlink.GetEnabledSkills(m.agentPath); err == nil {
					m.enabledSkills = enabled
					m.selected = make(map[int]bool)
					for i, skill := range m.skills {
						skillName := skills.GetSkillName(skill)
						if enabled[skillName] {
							m.selected[i] = true
						}
					}
				}
				return m, nil
			case "esc":
				m.editingPath = false
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.pathInput, cmd = m.pathInput.Update(msg)
		return m, cmd
	}

	if m.confirming {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "left", "h":
				m.confirmCursor = 0
			case "right", "l":
				m.confirmCursor = 1
			case "enter":
				if m.confirmCursor == 0 {
					if err := m.applyChanges(); err != nil {
						fmt.Fprintf(os.Stderr, "Error applying changes: %v\n", err)
					}
					m.quitting = true
					return m, tea.Quit
				}
				m.confirming = false
			case "esc", "q":
				m.confirming = false
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "tab":
			if len(m.filtered) > 0 {
				currentRepo := m.filtered[m.cursor].RepoName
				found := false
				for i := m.cursor + 1; i < len(m.filtered); i++ {
					if m.filtered[i].RepoName != currentRepo {
						m.cursor = i
						found = true
						break
					}
				}
				if !found {
					m.cursor = 0
				}
			}
			return m, nil
		case " ":
			if m.cursor >= 0 && m.cursor < len(m.filtered) {
				skill := m.filtered[m.cursor]
				// Find original index
				for i, s := range m.skills {
					if s.Path == skill.Path {
						m.selected[i] = !m.selected[i]
						break
					}
				}
			}
			return m, nil
		case "ctrl+p":
			m.editingPath = true
			m.pathInput.Focus()
			return m, textinput.Blink
		case "enter":
			added, removed := m.getChanges()
			if len(added) == 0 && len(removed) == 0 {
				m.quitting = true
				return m, tea.Quit
			}
			m.confirming = true
			m.confirmCursor = 0
			return m, nil
		}
	}

	var cmd tea.Cmd
	oldValue := m.filterInput.Value()
	m.filterInput, cmd = m.filterInput.Update(msg)
	if m.filterInput.Value() != oldValue {
		m.filter(m.filterInput.Value())
	}

	return m, cmd
}

func (m *model) getChanges() (added []string, removed []string) {
	for i, skill := range m.skills {
		skillName := skills.GetSkillName(skill)
		isSelected := m.selected[i]
		isEnabled := m.enabledSkills[skillName]

		if isSelected && !isEnabled {
			added = append(added, skill.Name)
		} else if !isSelected && isEnabled {
			removed = append(removed, skill.Name)
		}
	}
	return
}

func (m *model) applyChanges() error {
	for i, skill := range m.skills {
		skillName := skills.GetSkillName(skill)
		isSelected := m.selected[i]
		isEnabled := m.enabledSkills[skillName]

		if isSelected && !isEnabled {
			// Enable skill
			if err := symlink.EnsureSymlink(skill, m.agentPath); err != nil {
				return fmt.Errorf("failed to enable %s: %w", skill.Name, err)
			}
		} else if !isSelected && isEnabled {
			// Disable skill
			if err := symlink.RemoveSymlink(skillName, m.agentPath); err != nil {
				return fmt.Errorf("failed to disable %s: %w", skill.Name, err)
			}
		}
	}
	return nil
}

func (m *model) View() string {
	if m.quitting {
		return ""
	}

	var content string

	if m.confirming {
		added, removed := m.getChanges()
		content = TitleStyle.Render(" CONFIRM CHANGES ") + "\n\n"
		content += HeaderStyle.Render(fmt.Sprintf("Agentic: %s", m.agentName)) + "\n\n"

		if len(added) > 0 {
			content += SuccessStyle.Render("🚀 To be Added:") + "\n"
			for _, s := range added {
				content += fmt.Sprintf("  + %s\n", s)
			}
			content += "\n"
		}

		if len(removed) > 0 {
			content += lipgloss.NewStyle().Foreground(Pink).Render("🗑️  To be Removed:") + "\n"
			for _, s := range removed {
				content += fmt.Sprintf("  - %s\n", s)
			}
			content += "\n"
		}

		content += "Apply these changes?\n\n"

		choices := []string{"Yes, sync", "No, back"}
		for i, choice := range choices {
			style := NormalStyle
			if i == m.confirmCursor {
				style = SelectedStyle.Copy().Background(Purple).Foreground(White).Padding(0, 1)
			}
			content += style.Render(choice) + "  "
		}
		content += HelpStyle.Render("\n\n arrows: navigate • enter: select")
	} else if m.editingPath {
		content = TitleStyle.Render("Edit Path") + "\n\n"
		content += HeaderStyle.Render("Target Agentic: "+m.agentName) + "\n"
		content += m.pathInput.View() + "\n\n"
		content += HelpStyle.Render(" (Enter to confirm, Esc to cancel)")
	} else {
		content = TitleStyle.Render("Skill Management") + "\n\n"
		content += HeaderStyle.Render(fmt.Sprintf("Agentic: %s", m.agentName)) + "\n"
		content += InfoStyle.Render(fmt.Sprintf("Path: %s", m.agentPath)) + "\n\n"
		content += m.filterInput.View() + "\n\n"

		// Calculate scrolling window
		start := 0
		end := len(m.filtered)

		if len(m.filtered) > m.height {
			start = m.cursor - (m.height / 2)
			if start < 0 {
				start = 0
			}
			end = start + m.height
			if end > len(m.filtered) {
				end = len(m.filtered)
				start = end - m.height
			}
		}

		if start > 0 {
			content += DimStyle.Render("   ... scroll up") + "\n"
		}

		lastRepo := ""
		for i := start; i < end; i++ {
			skill := m.filtered[i]

			// Grouping header
			if skill.RepoName != lastRepo {
				content += "\n" + HeaderStyle.Render("📦 "+skill.RepoName) + "\n"
				lastRepo = skill.RepoName
			}

			// Find selection status from original index
			isSelected := false
			for origIdx, s := range m.skills {
				if s.Path == skill.Path {
					isSelected = m.selected[origIdx]
					break
				}
			}

			checkbox := CheckboxStyle.Render("○")
			if isSelected {
				checkbox = CheckboxStyle.Render("◉")
			}

			cursor := "  "
			style := NormalStyle
			if i == m.cursor {
				cursor = "> "
				style = SelectedStyle
			}

			line := fmt.Sprintf("%s%s %s", cursor, checkbox, style.Render(skills.GetSkillName(skill)))
			content += line + "\n"
		}

		if end < len(m.filtered) {
			content += DimStyle.Render("   ... scroll down") + "\n"
		} else {
			content += "\n" // Placeholder for scroll down
		}

		content += HelpStyle.Render("\n arrows: navigate • space: toggle • tab: jump repo • ctrl+p: path • enter: apply • esc: quit")
	}

	return BorderStyle.Render(content) + "\n"
}

func Run(agentName string) error {
	m, err := New(agentName)
	if err != nil {
		return err
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		return err
	}

	fmt.Println("✨ Skill changes synced successfully.")
	return nil
}

// --- Agentic Checklist TUI ---

type checklistModel struct {
	agentNames    []string
	filteredNames []string
	agentPaths    map[string]string
	checked       map[int]bool
	cursor        int
	quitting      bool
	err           error
	applied       []string // List of changes made for summary
	filterInput   textinput.Model
	height        int // Number of visible items
}

func NewChecklistModel() (*checklistModel, error) {
	agentics, err := config.GetAgentics()
	if err != nil {
		return nil, err
	}

	var names []string
	for k := range agentics {
		names = append(names, k)
	}
	sort.Strings(names)

	checked := make(map[int]bool)
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	for i, name := range names {
		relPath := agentics[name]
		parts := strings.Split(filepath.ToSlash(relPath), "/")
		if len(parts) > 0 {
			rootSubDir := parts[0]
			fullPath := filepath.Join(cwd, rootSubDir)
			if _, err := os.Stat(fullPath); err == nil {
				checked[i] = true
			}
		}
	}

	ti := textinput.New()
	ti.Placeholder = "Type to filter agents..."
	ti.Focus()

	return &checklistModel{
		agentNames:    names,
		filteredNames: names,
		agentPaths:    agentics,
		checked:       checked,
		filterInput:   ti,
		height:        12,
	}, nil
}

func (m *checklistModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *checklistModel) filter(term string) {
	if term == "" {
		m.filteredNames = m.agentNames
		return
	}
	var filtered []string
	term = strings.ToLower(term)
	for _, name := range m.agentNames {
		if strings.Contains(strings.ToLower(name), term) {
			filtered = append(filtered, name)
		}
	}
	m.filteredNames = filtered
	if m.cursor >= len(m.filteredNames) {
		m.cursor = 0
		if len(m.filteredNames) > 0 {
			m.cursor = len(m.filteredNames) - 1
		}
	}
}

func (m *checklistModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.filteredNames)-1 {
				m.cursor++
			}
		case " ":
			if len(m.filteredNames) > 0 {
				name := m.filteredNames[m.cursor]
				// Find original index
				for i, n := range m.agentNames {
					if n == name {
						m.checked[i] = !m.checked[i]
						break
					}
				}
			}
			return m, nil
		case "enter":
			if err := m.applyChanges(); err != nil {
				m.err = err
			} else {
				m.quitting = true
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	oldValue := m.filterInput.Value()
	m.filterInput, cmd = m.filterInput.Update(msg)
	if m.filterInput.Value() != oldValue {
		m.filter(m.filterInput.Value())
	}

	return m, cmd
}

func (m *checklistModel) applyChanges() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	type removal struct {
		name     string
		fullPath string
	}
	var toRemove []removal

	// First pass: identify removals
	for i, name := range m.agentNames {
		shouldExist := m.checked[i]
		if !shouldExist {
			relPath := m.agentPaths[name]
			parts := strings.Split(filepath.ToSlash(relPath), "/")
			if len(parts) > 0 {
				rootSubDir := parts[0]
				if rootSubDir != "" && rootSubDir != "." && rootSubDir != ".." {
					fullPath := filepath.Join(cwd, rootSubDir)
					if _, err := os.Stat(fullPath); err == nil {
						toRemove = append(toRemove, removal{name: name, fullPath: fullPath})
					}
				}
			}
		}
	}

	// Confirm removals if any
	for _, r := range toRemove {
		cm := NewConfirmModel(
			fmt.Sprintf("Remove '%s' from project?", r.name),
			fmt.Sprintf("This will delete: %s", r.fullPath),
		)
		p := tea.NewProgram(cm)
		cfModel, err := p.Run()
		if err != nil {
			return err
		}
		if cfModel.(*confirmModel).selected != 1 {
			// Skip this removal
			continue
		}

		// Final safety check
		if r.fullPath == cwd || r.fullPath == filepath.Dir(cwd) {
			return fmt.Errorf("safety: path protection triggered for %s", r.fullPath)
		}

		if err := os.RemoveAll(r.fullPath); err != nil {
			return fmt.Errorf("failed to remove %s: %w", r.fullPath, err)
		}
		m.applied = append(m.applied, fmt.Sprintf("🗑️  Disabled %s", r.name))
	}

	// Second pass: additions (no confirmation needed for mkdir)
	for i, name := range m.agentNames {
		if m.checked[i] {
			relPath := m.agentPaths[name]
			parts := strings.Split(filepath.ToSlash(relPath), "/")
			if len(parts) > 0 {
				rootSubDir := parts[0]
				if rootSubDir != "" && rootSubDir != "." && rootSubDir != ".." {
					fullPath := filepath.Join(cwd, rootSubDir)
					if _, err := os.Stat(fullPath); os.IsNotExist(err) {
						if err := os.MkdirAll(fullPath, 0755); err != nil {
							return fmt.Errorf("failed to create %s: %w", rootSubDir, err)
						}
						m.applied = append(m.applied, fmt.Sprintf("🚀 Enabled %s", name))
					}
				}
			}
		}
	}

	return nil
}

func (m *checklistModel) View() string {
	if m.err != nil {
		return BorderStyle.Render(TitleStyle.Render("Error")+"\n\n"+m.err.Error()+"\n\n"+HelpStyle.Render("press q to quit")) + "\n"
	}
	if m.quitting {
		return ""
	}

	var content string
	content = TitleStyle.Render("Project Agentics") + "\n\n"
	content += InfoStyle.Render("Select agentic environments to enable in this project:") + "\n"
	content += m.filterInput.View() + "\n\n"

	if len(m.filteredNames) == 0 {
		content += DimStyle.Render("  No agents matching filter") + "\n"
	} else {
		// Calculate scrolling
		start := 0
		end := len(m.filteredNames)

		if len(m.filteredNames) > m.height {
			start = m.cursor - (m.height / 2)
			if start < 0 {
				start = 0
			}
			end = start + m.height
			if end > len(m.filteredNames) {
				end = len(m.filteredNames)
				start = end - m.height
			}
		}

		if start > 0 {
			content += DimStyle.Render("   ... scroll up") + "\n"
		}

		for i := start; i < end; i++ {
			name := m.filteredNames[i]
			// Find original index for status
			origIdx := -1
			for j, n := range m.agentNames {
				if n == name {
					origIdx = j
					break
				}
			}

			checkbox := "○"
			if m.checked[origIdx] {
				checkbox = "◉"
			}

			cursor := "  "
			style := NormalStyle
			if i == m.cursor {
				cursor = "> "
				style = SelectedStyle
			}

			content += fmt.Sprintf("%s%s %s\n", cursor, CheckboxStyle.Render(checkbox), style.Render(name))
		}

		if end < len(m.filteredNames) {
			content += DimStyle.Render("   ... scroll down") + "\n"
		} else {
			content += "\n"
		}
	}

	content += HelpStyle.Render("\n arrows: navigate • space: toggle • enter: save • esc: quit")

	return BorderStyle.Render(content) + "\n"
}

func ManageAgentics() error {
	m, err := NewChecklistModel()
	if err != nil {
		return err
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	res := finalModel.(*checklistModel)
	if res.err != nil {
		return res.err
	}

	if len(res.applied) > 0 {
		fmt.Println("\n✨ Changes applied:")
		for _, msg := range res.applied {
			fmt.Println("  " + msg)
		}
		fmt.Println("\n💡 To manage skills for an agentic, use: skillops agentic manage [name]")
	} else {
		fmt.Println("No changes made.")
	}

	return nil
}

// --- Action Selection TUI ---

type actionModel struct {
	agentName string
	choices   []string
	cursor    int
	selected  int
	quitting  bool
}

func NewActionModel(agentName string) *actionModel {
	return &actionModel{
		agentName: agentName,
		choices:   []string{"Manage Skills", "Remove Agentic from project"},
	}
}

func (m *actionModel) Init() tea.Cmd { return nil }
func (m *actionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = m.cursor + 1
			m.quitting = true
			return m, tea.Quit
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *actionModel) View() string {
	if m.quitting {
		return ""
	}
	s := TitleStyle.Render(fmt.Sprintf("\nWhat would you like to do with '%s'?\n\n", m.agentName))
	for i, choice := range m.choices {
		cursor := " "
		style := NormalStyle
		if i == m.cursor {
			cursor = ">"
			style = SelectedStyle
		}
		s += fmt.Sprintf("\n%s %s", cursor, style.Render(choice))
	}
	s += HelpStyle.Render("\n arrows: navigate • enter: select • q: quit")
	return BorderStyle.Render(s) + "\n"
}

// --- Confirmation TUI ---

type confirmModel struct {
	message  string
	sub      string
	choices  []string
	cursor   int
	selected int // 1: Yes, 2: No
	quitting bool
}

func NewConfirmModel(message, sub string) *confirmModel {
	return &confirmModel{
		message: message,
		sub:     sub,
		choices: []string{"Yes, proceed", "No, cancel"},
		cursor:  1, // Default to "No" for safety
	}
}

func (m *confirmModel) Init() tea.Cmd { return nil }
func (m *confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			m.cursor = 0
		case "right", "l":
			m.cursor = 1
		case "enter":
			m.selected = m.cursor + 1
			m.quitting = true
			return m, tea.Quit
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *confirmModel) View() string {
	if m.quitting {
		return ""
	}

	s := TitleStyle.Render(" CONFIRMATION ") + "\n\n"
	s += m.message + "\n"
	if m.sub != "" {
		s += DimStyle.Render(m.sub) + "\n"
	}
	s += "\n"

	for i, choice := range m.choices {
		style := NormalStyle
		if i == m.cursor {
			style = SelectedStyle.Copy().Background(Purple).Foreground(White).Padding(0, 1)
		}
		s += style.Render(choice) + "  "
	}

	s += HelpStyle.Render("\n\n arrows/h/l: navigate • enter: confirm")
	return BorderStyle.BorderForeground(Pink).Render(s) + "\n"
}

// AskConfirm shows a yes/no confirmation TUI and returns true if Yes was selected
func AskConfirm(message, sub string) bool {
	cm := NewConfirmModel(message, sub)
	p := tea.NewProgram(cm)
	cfModel, err := p.Run()
	if err != nil {
		return false
	}
	cfRes := cfModel.(*confirmModel)
	return cfRes.selected == 1
}

func PerformAgenticAction(agentName string) error {
	am := NewActionModel(agentName)
	p := tea.NewProgram(am)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	res := finalModel.(*actionModel)
	if res.selected == 0 {
		return nil
	}

	if res.selected == 1 {
		// Manage Skills
		return Run(agentName)
	} else if res.selected == 2 {
		// Remove from project
		relPath, _ := config.GetAgenticPath(agentName)
		parts := strings.Split(filepath.ToSlash(relPath), "/")
		if len(parts) > 0 {
			rootSubDir := parts[0]
			// SAFETY check
			if rootSubDir == "" || rootSubDir == "." || rootSubDir == ".." {
				return fmt.Errorf("safety: cannot remove %s", relPath)
			}

			cwd, _ := os.Getwd()
			fullPath := filepath.Join(cwd, rootSubDir)

			// PROMPT FOR CONFIRMATION
			cm := NewConfirmModel(
				fmt.Sprintf("Are you sure you want to remove '%s' from this project?", agentName),
				fmt.Sprintf("Target path: %s", fullPath),
			)
			cp := tea.NewProgram(cm)
			cfModel, err := cp.Run()
			if err != nil {
				return err
			}
			cfRes := cfModel.(*confirmModel)
			if cfRes.selected != 1 {
				fmt.Println("❌ Removal cancelled.")
				return nil
			}

			if err := os.RemoveAll(fullPath); err != nil {
				return err
			}
			fmt.Printf("🗑️  Successfully removed %s from project root.\n", rootSubDir)
		}
	}

	return nil
}
