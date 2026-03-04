package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"skillops/internal/config"
	"skillops/internal/symlink"
	"skillops/internal/tui"
	"skillops/internal/utils"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove <skill_name>",
	Short:   "Remove a skill from the global skills directory",
	GroupID: "agentic",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		skillName := args[0]
		if err := utils.ValidateName(skillName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// 1. Check if linked in any Agentic IDE
		links, err := symlink.FindAllSkillLinks(skillName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not check for active links: %v\n", err)
		}

		if len(links) > 0 {
			fmt.Printf("⚠️  Skill '%s' is currently linked in the following Agentic IDEs:\n", skillName)
			for _, l := range links {
				fmt.Printf("   - %s\n", l)
			}
			if !tui.AskConfirm(
				fmt.Sprintf("Skill '%s' is being used!", skillName),
				"Force remove and unlink from all?",
			) {
				fmt.Println("Aborted.")
				return
			}

			// 2. Unlink from all
			agentics, _ := config.GetAgentics()
			for _, agentName := range links {
				path := agentics[agentName]
				if err := symlink.RemoveSymlink(skillName, path); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to unlink from %s: %v\n", agentName, err)
				}
			}
		}

		// Find actual storage path
		skillPath, err := symlink.FindSkillPath(skillName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Removing skill '%s'...\n", skillName)
		if err := os.RemoveAll(skillPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to remove skill: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Skill removed successfully.")
	},
}

var removeAllCmd = &cobra.Command{
	Use:     "remove-all",
	Short:   "Remove all skills from the global skills directory",
	GroupID: "agentic",
	Run: func(cmd *cobra.Command, args []string) {
		if !tui.AskConfirm("Are you sure you want to remove ALL skills?", "This action cannot be undone.") {
			fmt.Println("Aborted.")
			return
		}

		entries, err := os.ReadDir(config.SkillsDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to read skills directory: %v\n", err)
			os.Exit(1)
		}

		for _, entry := range entries {
			path := filepath.Join(config.SkillsDir, entry.Name())
			if err := os.RemoveAll(path); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to remove %s: %v\n", path, err)
			}
		}
		fmt.Println("All skills removed successfully.")
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(removeAllCmd)
}
