package tui

import (
	"fmt"
	"skillops/internal/skills"

	tea "github.com/charmbracelet/bubbletea"
)

type listModel struct {
	skills   []skills.Skill
	cursor   int
	quitting bool
}

func NewListModel(allSkills []skills.Skill) *listModel {
	return &listModel{
		skills: allSkills,
	}
}

func (m *listModel) Init() tea.Cmd { return nil }
func (m *listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.skills)-1 {
				m.cursor++
			}
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *listModel) View() string {
	if m.quitting {
		return ""
	}

	s := TitleStyle.Render(" DOWNLOADED SKILLS ") + "\n\n"
	s += InfoStyle.Render(fmt.Sprintf("Total: %d skills", len(m.skills))) + "\n\n"

	for i, skill := range m.skills {
		cursor := "  "
		style := NormalStyle
		if i == m.cursor {
			cursor = "> "
			style = SelectedStyle
		}
		s += fmt.Sprintf("%s%s %s\n", cursor, style.Render(skill.Name), DimStyle.Render("("+skill.RepoName+")"))
	}

	s += HelpStyle.Render("\n arrows: navigate • q: quit")
	return BorderStyle.Render(s) + "\n"
}

func ShowList() error {
	allSkills, err := skills.Discover()
	if err != nil {
		return err
	}
	if len(allSkills) == 0 {
		fmt.Println("No skills found. Use 'skillops pull' to download skill repositories.")
		return nil
	}

	p := tea.NewProgram(NewListModel(allSkills))
	_, err = p.Run()
	return err
}
