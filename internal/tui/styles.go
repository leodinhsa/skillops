package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Premium Color Palette
	Purple = lipgloss.Color("#7D56F4")
	Pink   = lipgloss.Color("#F25DA1")
	Teal   = lipgloss.Color("#00F2FE")
	Gray   = lipgloss.Color("#2B2D31")
	White  = lipgloss.Color("#FFFFFF")
	Dim    = lipgloss.Color("#4E4F56")
	Green  = lipgloss.Color("#A6E22E")

	SelectedStyle = lipgloss.NewStyle().
			Foreground(Purple).
			Bold(true).
			PaddingLeft(1)

	NormalStyle = lipgloss.NewStyle().
			Foreground(White).
			PaddingLeft(1)

	CheckboxStyle = lipgloss.NewStyle().
			Foreground(Teal)

	DimStyle = lipgloss.NewStyle().
			Foreground(Dim)

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(White).
			Background(Purple).
			Padding(0, 2).
			MarginBottom(1)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(Pink).
			Bold(true).
			MarginTop(1).
			MarginBottom(1)

	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Purple).
			Padding(1, 2).
			Width(90).
			Height(30)

	InfoStyle = lipgloss.NewStyle().
			Foreground(Dim).
			Italic(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(Dim).
			MarginTop(1)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(Green).
			Bold(true)
)
