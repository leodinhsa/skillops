package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"skillops/internal/config"
	"skillops/internal/skills"
	"skillops/internal/tui"

	"github.com/spf13/cobra"
)

var (
	addAllTools  bool
	addToolFlag  string
)

var addCmd = &cobra.Command{
	Use:     "add [skill]",
	Short:   "Link a skill into the project's active IDE tools",
	GroupID: "project",
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Guard: local config must exist
		if _, err := config.ReadLocalConfig(); err != nil {
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

		// Non-TUI paths: --all or --tool with a positional skill arg
		if addAllTools || addToolFlag != "" {
			if len(args) == 0 {
				fmt.Fprintln(os.Stderr, "Error: a skill name is required with --all or --tool")
				os.Exit(1)
			}
			skillArg := args[0]

			// Resolve skill from global store
			allSkills, err := skills.Discover()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error discovering skills: %v\n", err)
				os.Exit(1)
			}

			// Find matching skill(s) by short name or full identity
			var matched []struct {
				identity string
				path     string
			}
			for _, s := range allSkills {
				shortName := strings.SplitN(s.Name, "/", 2)
				if s.Name == skillArg || (len(shortName) == 2 && shortName[1] == skillArg) {
					matched = append(matched, struct {
						identity string
						path     string
					}{s.Name, s.Path})
				}
			}

			if len(matched) == 0 {
				fmt.Fprintf(os.Stderr, "Error: skill '%s' not found in global store\n", skillArg)
				os.Exit(1)
			}

			// Determine target tools
			var targetTools []string
			if addAllTools {
				var err error
				targetTools, err = config.GetActiveTools()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading active tools: %v\n", err)
					os.Exit(1)
				}
			} else {
				targetTools = strings.Split(addToolFlag, ",")
				for i, t := range targetTools {
					targetTools[i] = strings.TrimSpace(t)
				}
			}

			cwd, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			var results []string
			for _, skill := range matched {
				for _, tool := range targetTools {
					result, err := tui.LinkSkillToTool(cwd, skill.identity, skill.path, tool)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
						continue
					}
					if result != "" {
						results = append(results, result)
					}
				}
			}

			if len(results) > 0 {
				fmt.Println("\n✨ Skills linked:")
				for _, r := range results {
					fmt.Println("  " + r)
				}
			} else {
				fmt.Println("No changes made.")
			}
			return
		}

		// TUI flow
		preselected := ""
		if len(args) == 1 {
			preselected = args[0]
		}

		if err := tui.RunAdd(preselected); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	addCmd.Flags().BoolVar(&addAllTools, "all", false, "Link into all active tools")
	addCmd.Flags().StringVar(&addToolFlag, "tool", "", "Comma-separated list of tools to target")
	rootCmd.AddCommand(addCmd)
}
