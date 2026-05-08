package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"skillops/internal/config"
	"skillops/internal/git"
	"skillops/internal/skills"
	"skillops/internal/tui"

	"github.com/spf13/cobra"
)

var (
	skillFlag string
)

var pullCmd = &cobra.Command{
	Use:     "pull <repo_url>",
	Short:   "Pull a skill repository from GitHub",
	GroupID: "skill",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]

		// Parse the repository URL to extract host and repoPath
		host, repoPath, err := git.ParseRepoURL(url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Construct destination: ~/.skillops/skills/<host>/<repoPath>
		dest := filepath.Join(config.SkillsDir, host, filepath.FromSlash(repoPath))

		if skillFlag != "" {
			// Pull specific skill using pathInRepo
			fmt.Println(tui.TitleStyle.Render(" PULLING SPECIFIC SKILL "))
			fmt.Printf("Source: %s\n", tui.DimStyle.Render(url))
			fmt.Printf("Skill:  %s\n", tui.SuccessStyle.Render(skillFlag))

			// destSkillDir = ~/.skillops/skills/<host>/<repoPath>/<pathInRepo>
			destSkillDir := filepath.Join(dest, filepath.FromSlash(skillFlag))
			if err := skills.PullSkillFromURL(url, skillFlag, destSkillDir); err != nil {
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

			// Save repo metadata
			commitHash := git.GetLatestCommit(dest)
			meta := skills.NewRepoMetadata{
				RepoURL:    url,
				PulledAt:   time.Now(),
				CommitHash: commitHash,
			}
			if err := skills.SaveRepoMeta(dest, meta); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save repo metadata: %v\n", err)
			}
		}

		fmt.Printf("🚀 %s\n", tui.SuccessStyle.Render("Successfully pulled."))
	},
}

func init() {
	pullCmd.Flags().StringVarP(&skillFlag, "skill", "s", "", "Pull a specific skill from the repository (path in repo, e.g. 'skills/logger')")
	rootCmd.AddCommand(pullCmd)
}
