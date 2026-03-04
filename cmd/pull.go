package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"skillops/internal/config"
	"skillops/internal/git"
	"skillops/internal/tui"
	"skillops/internal/utils"

	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull <repo_url>",
	Short: "Pull a skill repository from GitHub",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]
		url = git.NormalizeRepoURL(url)
		repoName := git.ExtractRepoName(url)

		if err := utils.ValidateName(repoName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid repo name from URL: %v\n", err)
			os.Exit(1)
		}

		dest := filepath.Join(config.SkillsDir, repoName)
		fmt.Println(tui.TitleStyle.Render(" PULLING SKILL "))
		fmt.Printf("Source: %s\n", tui.DimStyle.Render(url))
		fmt.Printf("Target: %s\n\n", tui.DimStyle.Render(dest))

		if err := git.Clone(url, dest); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("🚀 %s\n", tui.SuccessStyle.Render("Successfully pulled repository."))
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)
}
