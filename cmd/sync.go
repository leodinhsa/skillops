package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"skillops/internal/config"
	"skillops/internal/skills"
	"skillops/internal/tui"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:     "sync",
	Short:   "Restore all symlinks declared in the local config",
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

		// Read settings for registry auto-pull support
		settings, _ := config.ReadSettings()
		registries := settings.Registries

		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Sort tools for deterministic output
		tools := make([]string, 0, len(cfg.Tools))
		for t := range cfg.Tools {
			tools = append(tools, t)
		}
		sort.Strings(tools)

		var (
			created    int
			autoPulled int
			warnings   []string
		)

		for _, tool := range tools {
			toolRelPath, err := config.GetAgenticPath(tool)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("unknown tool '%s', skipping", tool))
				continue
			}

			skillsDir := filepath.Join(cwd, toolRelPath)
			if err := os.MkdirAll(skillsDir, 0755); err != nil {
				warnings = append(warnings, fmt.Sprintf("failed to create skills dir for %s: %v", tool, err))
				continue
			}

			for _, identity := range cfg.Tools[tool] {
				parts := strings.SplitN(identity, "/", 2)
				if len(parts) != 2 {
					warnings = append(warnings, fmt.Sprintf("invalid skill identity '%s', skipping", identity))
					continue
				}
				repoName := parts[0]
				skillName := parts[1]

				globalSkillPath := filepath.Join(config.SkillsDir, repoName, skillName)
				symlinkPath := filepath.Join(skillsDir, skillName)

				// [A] Check if skill exists in global store
				if _, err := os.Stat(globalSkillPath); os.IsNotExist(err) {
					// [B] No registries configured?
					if len(registries) == 0 {
						warnings = append(warnings, fmt.Sprintf("skill '%s' not found locally, run 'skillops pull'", identity))
						continue
					}

					// [C] Try each registry in order
					pulled := false
					for i, reg := range registries {
						regName := reg.Name
						if regName == "" {
							regName = fmt.Sprintf("registry-%d", i+1)
						}

						cloneURL := strings.TrimRight(reg.URL, "/") + "/" + repoName
						if pullErr := skills.PullSkillFromURL(cloneURL, skillName, globalSkillPath); pullErr == nil {
							autoPulled++
							pulled = true
							break
						}
					}

					if !pulled {
						warnings = append(warnings, fmt.Sprintf("skill '%s' not found in any configured registry", identity))
						continue
					}
				}

				// [D] Create symlink if it doesn't already exist
				if _, err := os.Lstat(symlinkPath); err == nil {
					// Already exists — skip (idempotent)
					continue
				}

				if err := os.Symlink(globalSkillPath, symlinkPath); err != nil {
					warnings = append(warnings, fmt.Sprintf("failed to create symlink for '%s' in %s: %v", identity, tool, err))
					continue
				}
				created++
			}
		}

		// Render TUI summary panel
		fmt.Println(renderSyncSummary(created, autoPulled, warnings))
	},
}

// renderSyncSummary builds the lipgloss summary panel for sync output.
func renderSyncSummary(created, autoPulled int, warnings []string) string {
	var sb strings.Builder

	sb.WriteString(tui.SuccessStyle.Render(fmt.Sprintf("✓  %d %s created", created, pluralize(created, "symlink", "symlinks"))) + "\n")

	if autoPulled > 0 {
		sb.WriteString(tui.CheckboxStyle.Render(fmt.Sprintf("↓  %d auto-pulled from registry", autoPulled)) + "\n")
	}

	if len(warnings) > 0 {
		sb.WriteString(tui.HeaderStyle.Render(fmt.Sprintf("⚠  %d %s", len(warnings), pluralize(len(warnings), "warning", "warnings"))) + "\n")
		for _, w := range warnings {
			sb.WriteString(tui.DimStyle.Render("   • "+w) + "\n")
		}
	}

	return tui.BorderStyle.Render(sb.String())
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
