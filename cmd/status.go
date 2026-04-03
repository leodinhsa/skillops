package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"skillops/internal/config"
	"skillops/internal/tui"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:     "status",
	Short:   "Show the current skill and IDE state of this project",
	GroupID: "project",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.ReadLocalConfig()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				fmt.Fprintln(os.Stderr, "No local config found.")
				fmt.Fprintln(os.Stderr, "")
				fmt.Fprintln(os.Stderr, "If you're upgrading from v1, run:")
				fmt.Fprintln(os.Stderr, "  skillops init   — declare which IDEs this project uses")
				fmt.Fprintln(os.Stderr, "  skillops sync   — restore your skill links")
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Error reading local config: %v\n", err)
			os.Exit(1)
		}

		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Sort tools alphabetically
		tools := make([]string, 0, len(cfg.Tools))
		for t := range cfg.Tools {
			tools = append(tools, t)
		}
		sort.Strings(tools)

		totalLinked := 0
		var body strings.Builder

		for _, tool := range tools {
			body.WriteString("\n")
			body.WriteString(tui.HeaderStyle.Render("  "+tool) + "\n")

			skills := cfg.Tools[tool]
			if len(skills) == 0 {
				body.WriteString(tui.DimStyle.Render("    — no skills linked") + "\n")
				continue
			}

			toolRelPath, err := config.GetAgenticPath(tool)
			if err != nil {
				// Tool not in global config — show skills as missing
				for _, identity := range skills {
					parts := strings.SplitN(identity, "/", 2)
					shortName := identity
					repoName := ""
					if len(parts) == 2 {
						repoName = parts[0]
						shortName = parts[1]
					}
					line := fmt.Sprintf("    %s %-20s %s",
						tui.CheckboxStyle.Render("○"),
						shortName,
						tui.DimStyle.Render("("+repoName+") not linked"),
					)
					body.WriteString(line + "\n")
				}
				continue
			}

			for _, identity := range skills {
				parts := strings.SplitN(identity, "/", 2)
				shortName := identity
				repoName := ""
				if len(parts) == 2 {
					repoName = parts[0]
					shortName = parts[1]
				}

				symlinkPath := filepath.Join(cwd, toolRelPath, shortName)
				_, lstatErr := os.Lstat(symlinkPath)

				var indicator string
				var suffix string
				if lstatErr == nil {
					indicator = tui.CheckboxStyle.Render("◉")
					suffix = tui.DimStyle.Render("(" + repoName + ")")
					totalLinked++
				} else {
					indicator = tui.DimStyle.Render("○")
					suffix = tui.DimStyle.Render("(" + repoName + ") not linked")
				}

				line := fmt.Sprintf("    %s %-20s %s", indicator, shortName, suffix)
				body.WriteString(line + "\n")
			}
		}

		// Registry section
		settings, _ := config.ReadSettings()
		if len(settings.Registries) > 0 {
			body.WriteString("\n")
			var regNames []string
			for i, reg := range settings.Registries {
				name := reg.Name
				if name == "" {
					name = fmt.Sprintf("registry-%d", i+1)
				}
				regNames = append(regNames, name)
			}
			body.WriteString(tui.DimStyle.Render("  Registries: "+strings.Join(regNames, ", ")) + "\n")
		}

		// Footer
		body.WriteString("\n")
		footerText := fmt.Sprintf("  %d %s active • %d %s linked",
			len(tools), pluralize(len(tools), "tool", "tools"),
			totalLinked, pluralize(totalLinked, "skill", "skills"),
		)
		body.WriteString(tui.InfoStyle.Render(footerText) + "\n")

		// Build the full panel content
		titleLine := tui.TitleStyle.Render("  PROJECT STATUS  ")
		cwdLine := tui.DimStyle.Render("  " + cwd)

		content := titleLine + "\n" + cwdLine + "\n" + body.String()

		fmt.Println(tui.BorderStyle.Render(content))
	},
}

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
