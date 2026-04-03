package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"skillops/internal/config"
	"skillops/internal/git"
	"skillops/internal/skills"
	"skillops/internal/tui"
	"skillops/internal/utils"

	"github.com/spf13/cobra"
)

var (
	skillName string
)

var pullCmd = &cobra.Command{
	Use:     "pull <repo_url>",
	Short:   "Pull a skill repository from GitHub",
	GroupID: "skill",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]
		url = git.NormalizeRepoURL(url)
		repoName := git.ExtractRepoName(url)

		if err := utils.ValidateName(repoName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid repo name from URL: %v\n", err)
			os.Exit(1)
		}

		dest := filepath.Join(config.SkillsDir, repoName)

		if skillName != "" {
			// Extract specific skill
			fmt.Println(tui.TitleStyle.Render(" PULLING SPECIFIC SKILL "))
			fmt.Printf("Source: %s\n", tui.DimStyle.Render(url))
			fmt.Printf("Skill:  %s\n", tui.SuccessStyle.Render(skillName))

			finalDest := filepath.Join(dest, skillName)
			if err := skills.PullSkillFromURL(url, skillName, finalDest); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Pull entire repo
			fmt.Println(tui.TitleStyle.Render(" PULLING SKILL "))
			fmt.Printf("Source: %s\n", tui.DimStyle.Render(url))
			fmt.Printf("Target: %s\n\n", tui.DimStyle.Render(dest))

			if err := git.Clone(url, dest); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
				os.Exit(1)
			}

			// Save metadata
			meta := skills.RepoMetadata{URL: url}
			skills.SaveMetadata(dest, meta)
		}

		fmt.Printf("🚀 %s\n", tui.SuccessStyle.Render("Successfully pulled."))
	},
}

func init() {
	pullCmd.Flags().StringVarP(&skillName, "skill", "s", "", "Pull a specific skill from the repository")
	rootCmd.AddCommand(pullCmd)
}
