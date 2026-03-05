package tui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"skillops/internal/config"
	"skillops/internal/git"
	"skillops/internal/skills"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type listModel struct {
	allSkills       []skills.Skill
	filtered        []skills.Skill
	cursor          int
	quitting        bool
	filterInput     textinput.Model
	height          int
	lastCopied      string
	repoMetadata    map[string]skills.RepoMetadata
	repoSkillCounts map[string]int
	orderedRepos    []string
}

func NewListModel(allSkills []skills.Skill) *listModel {
	// Sort by repo name for grouping
	sort.Slice(allSkills, func(i, j int) bool {
		if allSkills[i].RepoName != allSkills[j].RepoName {
			return allSkills[i].RepoName < allSkills[j].RepoName
		}
		return allSkills[i].Name < allSkills[j].Name
	})

	var orderedRepos []string
	repoSkillCounts := make(map[string]int)
	repoMetadata := make(map[string]skills.RepoMetadata)

	for _, s := range allSkills {
		if repoSkillCounts[s.RepoName] == 0 {
			orderedRepos = append(orderedRepos, s.RepoName)
		}
		repoSkillCounts[s.RepoName]++
		if _, ok := repoMetadata[s.RepoName]; !ok {
			repoPath := filepath.Join(config.SkillsDir, s.RepoName)
			if meta, err := skills.LoadMetadata(repoPath); err == nil {
				repoMetadata[s.RepoName] = meta
			}
		}
	}

	ti := textinput.New()
	ti.Placeholder = "Search skills..."
	ti.Focus()

	return &listModel{
		allSkills:       allSkills,
		filtered:        allSkills,
		filterInput:     ti,
		height:          15,
		repoSkillCounts: repoSkillCounts,
		repoMetadata:    repoMetadata,
		orderedRepos:    orderedRepos,
	}
}

func (m *listModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *listModel) filter(term string) {
	if term == "" {
		m.filtered = m.allSkills
		return
	}
	var filtered []skills.Skill
	term = strings.ToLower(term)
	for _, s := range m.allSkills {
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

func (m *listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
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
					// Loop to beginning
					m.cursor = 0
				}
			}
			return m, nil
		case " ":
			if len(m.filtered) > 0 {
				skill := m.filtered[m.cursor]
				name := skills.GetSkillName(skill)
				cpCmd := exec.Command("pbcopy")
				cpCmd.Stdin = strings.NewReader(name)
				_ = cpCmd.Run()
				m.lastCopied = name
			}
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

func (m *listModel) View() string {
	if m.quitting {
		return ""
	}

	s := TitleStyle.Render(" DOWNLOADED SKILLS ") + "\n\n"
	s += m.filterInput.View() + "\n\n"

	if len(m.filtered) == 0 {
		s += DimStyle.Render("  No skills matching filter") + "\n"
	} else {
		// Calculate viewport
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
			s += DimStyle.Render("   ... scroll up") + "\n"
		}

		lastRepo := ""
		for i := start; i < end; i++ {
			skill := m.filtered[i]

			// Grouping header
			if skill.RepoName != lastRepo {
				displayRepoName := skill.RepoName
				if meta, ok := m.repoMetadata[skill.RepoName]; ok {
					fullPath := git.ExtractFullRepoPath(meta.URL)
					if fullPath != "" {
						displayRepoName = fullPath
					}
				}

				// Find index of this repo
				repoIdx := 1
				for j, r := range m.orderedRepos {
					if r == skill.RepoName {
						repoIdx = j + 1
						break
					}
				}

				count := m.repoSkillCounts[skill.RepoName]
				header := fmt.Sprintf("%d. 📦 %s %s", repoIdx, displayRepoName, DimStyle.Render(fmt.Sprintf("%d skills", count)))

				s += "\n" + HeaderStyle.Render(header) + "\n"
				lastRepo = skill.RepoName
			}

			cursor := "  "
			style := NormalStyle
			if i == m.cursor {
				cursor = "> "
				style = SelectedStyle
			}
			s += fmt.Sprintf("%s%s\n", cursor, style.Render(skills.GetSkillName(skill)))
		}

		if end < len(m.filtered) {
			s += "\n" + DimStyle.Render("   ... scroll down") + "\n"
		} else {
			s += "\n\n"
		}
	}

	if m.lastCopied != "" {
		s += "\n" + SuccessStyle.Render(fmt.Sprintf("✓ Copied '%s' to clipboard", m.lastCopied)) + "\n"
	} else {
		s += "\n\n"
	}

	// Calculate summary stats
	totalGroups := len(m.orderedRepos)
	totalSkills := len(m.allSkills)

	s += "\n" + DimStyle.Render(fmt.Sprintf("Total Repos:  %d", totalGroups))
	s += "\n" + DimStyle.Render(fmt.Sprintf("Total Skills: %d", totalSkills))

	s += HelpStyle.Render("\n\n ↑ ↓     : navigate  \n <space> : copy name  \n <tab>   : jump repo  \n <esc>   : quit")
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
