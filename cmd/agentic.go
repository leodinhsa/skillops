package cmd

import (
	"fmt"
	"os"

	"skillops/internal/config"
	"skillops/internal/symlink"
	"skillops/internal/tui"
	"skillops/internal/utils"

	"github.com/spf13/cobra"
)

var agenticCmd = &cobra.Command{
	Use:     "agentic",
	Short:   "Manage agentic IDEs and their skills",
	GroupID: "project",
	Run: func(cmd *cobra.Command, args []string) {
		// New root TUI: Checklist for active agentics
		if err := tui.ManageAgentics(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var agenticManageCmd = &cobra.Command{
	Use:   "manage <agentic_name>",
	Short: "Manage skills for a specific agentic",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		agentName := args[0]

		if err := utils.ValidateName(agentName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		enabled, rootSubDir, err := utils.IsAgenticEnabled(agentName)
		if err != nil {
			// Global config missing
			fmt.Fprintf(os.Stderr, "Error: Unknown agentic '%s'.\n", agentName)
			fmt.Fprintf(os.Stderr, "💡 To register it globally, use: skillops config add-agentic -n %s -p <path>\n", agentName)
			os.Exit(1)
		}

		if !enabled {
			fmt.Fprintf(os.Stderr, "Error: Agentic '%s' is not enabled in this project.\n", agentName)
			fmt.Fprintf(os.Stderr, "💡 Use 'skillops agentic' to enable the '%s' environment.\n", rootSubDir)
			os.Exit(1)
		}

		if err := tui.PerformAgenticAction(agentName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var agenticRemoveSkillCmd = &cobra.Command{
	Use:   "remove-skill <agentic_name> <skill_name>",
	Short: "Remove a specific skill's symlink from an agentic",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		agentName := args[0]
		skillName := args[1]

		enabled, rootSubDir, err := utils.IsAgenticEnabled(agentName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Unknown agentic '%s'.\n", agentName)
			fmt.Fprintf(os.Stderr, "💡 To register it globally, use: skillops config add-agentic -n %s -p <path>\n", agentName)
			os.Exit(1)
		}
		if !enabled {
			fmt.Fprintf(os.Stderr, "Error: Agentic '%s' is not enabled in this project.\n", agentName)
			fmt.Fprintf(os.Stderr, "💡 Use 'skillops agentic' to enable the '%s' environment.\n", rootSubDir)
			os.Exit(1)
		}

		path, _ := config.GetAgenticPath(agentName)
		if err := symlink.RemoveSymlink(skillName, path); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Removed skill '%s' from agentic '%s'.\n", skillName, agentName)
	},
}

var agenticRemoveSkillsCmd = &cobra.Command{
	Use:   "remove-skills <agentic_name>",
	Short: "Remove all skill symlinks from an agentic",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		agentName := args[0]

		enabled, rootSubDir, err := utils.IsAgenticEnabled(agentName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Unknown agentic '%s'.\n", agentName)
			fmt.Fprintf(os.Stderr, "💡 To register it globally, use: skillops config add-agentic -n %s -p <path>\n", agentName)
			os.Exit(1)
		}
		if !enabled {
			fmt.Fprintf(os.Stderr, "Error: Agentic '%s' is not enabled in this project.\n", agentName)
			fmt.Fprintf(os.Stderr, "💡 Use 'skillops agentic' to enable the '%s' environment.\n", rootSubDir)
			os.Exit(1)
		}

		path, _ := config.GetAgenticPath(agentName)
		enabledSkills, err := symlink.GetEnabledSkills(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		for skillName := range enabledSkills {
			if err := symlink.RemoveSymlink(skillName, path); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to remove %s: %v\n", skillName, err)
			}
		}
		fmt.Printf("Removed all skills from agentic '%s'.\n", agentName)
	},
}

func init() {
	completeAgenticNames := func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		agentics, err := config.GetAgentics()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		var names []string
		for name := range agentics {
			if enabled, _, _ := utils.IsAgenticEnabled(name); enabled {
				names = append(names, name)
			}
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}

	agenticManageCmd.ValidArgsFunction = completeAgenticNames
	agenticRemoveSkillCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return completeAgenticNames(cmd, args, toComplete)
		}
		// Second argument is skill name - could potentially autocomplete enabled skills here
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	agenticRemoveSkillsCmd.ValidArgsFunction = completeAgenticNames

	agenticCmd.AddCommand(agenticManageCmd)
	agenticCmd.AddCommand(agenticRemoveSkillCmd)
	agenticCmd.AddCommand(agenticRemoveSkillsCmd)
	rootCmd.AddCommand(agenticCmd)
}
