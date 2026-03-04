package cmd

import (
	"fmt"
	"os"
	"skillops/internal/config"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	// Premium Color Palette for Help
	purple = lipgloss.Color("#7D56F4")
	pink   = lipgloss.Color("#F25DA1")
	teal   = lipgloss.Color("#00F2FE")
	white  = lipgloss.Color("#FFFFFF")
	dim    = lipgloss.Color("#4E4F56")

	helpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(white).
			Background(purple).
			Padding(0, 1).
			MarginBottom(1)

	helpHeaderStyle = lipgloss.NewStyle().
			Foreground(pink).
			Bold(true)

	helpCmdStyle = lipgloss.NewStyle().
			Foreground(teal).
			Bold(true)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(white)

	helpFlagStyle = lipgloss.NewStyle().
			Foreground(purple)

	helpDimStyle = lipgloss.NewStyle().
			Foreground(dim).
			Italic(true)
)

var rootCmd = &cobra.Command{
	Use:   "skillops",
	Short: "Skill Ops - Manage AI agent skills",
	Long:  `A CLI tool to pull skill repositories and manage symlinks for Agentic AI projects.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Ensure config exists
		if err := config.EnsureConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating config: %v\n", err)
			os.Exit(1)
		}
	},
	SilenceErrors:              true,
	SilenceUsage:               true,
	SuggestionsMinimumDistance: 2,
}

func init() {
	rootCmd.AddGroup(&cobra.Group{ID: "project", Title: "Project Configuration"})
	rootCmd.AddGroup(&cobra.Group{ID: "agentic", Title: "Agentic Configuration"})

	// Custom Help Template
	cobra.AddTemplateFunc("styleHeader", func(s string) string {
		return helpHeaderStyle.Render(strings.ToUpper(s))
	})
	cobra.AddTemplateFunc("styleCmd", func(s string) string {
		return helpCmdStyle.Render(s)
	})
	cobra.AddTemplateFunc("styleFlag", func(s string) string {
		return helpFlagStyle.Render(s)
	})
	cobra.AddTemplateFunc("styleDesc", func(s string) string {
		return helpDescStyle.Render(s)
	})
	cobra.AddTemplateFunc("styleTitle", func(s string) string {
		return helpTitleStyle.Render(s)
	})

	rootCmd.SetHelpTemplate(fmt.Sprintf(`
%s
{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}{{end}}

{{styleHeader "Usage:"}}
  {{styleCmd .UseLine}}{{if .HasAvailableSubCommands}} {{styleCmd "[command]"}}{{end}}

{{if .HasAvailableLocalFlags}}{{styleHeader "Flags:"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}

{{if .HasAvailableInheritedFlags}}{{styleHeader "Global Flags:"}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}

{{if .HasAvailableSubCommands}}{{styleHeader "Available Commands:"}}{{range .Groups}}
  {{$group := .}}{{styleTitle .Title}}{{range $.Commands}}{{if eq .GroupID $group.ID}}
    {{styleCmd (rpad .Name .NamePadding)}} {{styleDesc .Short}}{{end}}{{end}}{{end}}

  {{if .HasAvailableSubCommands}}{{range .Commands}}{{if not .GroupID}}
    {{styleCmd (rpad .Name .NamePadding)}} {{styleDesc .Short}}{{end}}{{end}}{{end}}{{end}}

{{if .HasAvailableSubCommands}}
Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`, helpTitleStyle.Render("SKILL OPS SERVICE")))
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		if cmd, _, _ := rootCmd.Find(os.Args[1:]); cmd == rootCmd && len(os.Args) > 1 {
			// If we didn't find a matching command, show suggestions
			fmt.Println("\nRun 'skillops --help' for usage details.")
		}
		os.Exit(1)
	}
}
