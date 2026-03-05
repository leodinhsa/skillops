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

			// Clone to temp
			tempDest, err := os.MkdirTemp("", "skillops-pull-*")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating temp dir: %v\n", err)
				os.Exit(1)
			}
			defer os.RemoveAll(tempDest)

			if err := git.Clone(url, tempDest); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
				os.Exit(1)
			}

			// Find skill folder
			skillPath := ""
			// Case 1 & 3: Root or direct subfolder
			rootSkill := filepath.Join(tempDest, "SKILL.md")
			if _, err := os.Stat(rootSkill); err == nil {
				// Check if name matches (simplistic check: we take the whole repo as the skill if it has SKILL.md at root)
				skillPath = tempDest
			}

			// Case 1 & 2: Subfolder
			if skillPath == "" || skillPath == tempDest {
				subfolders := []string{skillName, filepath.Join("skills", skillName)}
				for _, sub := range subfolders {
					candidate := filepath.Join(tempDest, sub)
					if _, err := os.Stat(filepath.Join(candidate, "SKILL.md")); err == nil {
						skillPath = candidate
						break
					}
				}
			}

			if skillPath == "" {
				fmt.Fprintf(os.Stderr, "❌ Error: skill '%s' not found in repository\n", skillName)
				os.Exit(1)
			}

			// Target: ~/.skillops/skills/<repo_name>/<skill_name>
			finalDest := filepath.Join(dest, skillName)
			if skillPath == tempDest {
				// If root skill, we use repoName as folder?
				// The prompt says: git@github.com/example -> example/SKILL.md
				// Case 3 says Move root SKILL.md to ~/.skillops/skills/repo/SKILL.md
				// But skillops discovery expects repo/skill/SKILL.md
				// Let's stick to repo/skillName/SKILL.md
				finalDest = filepath.Join(dest, repoName)
			}

			os.MkdirAll(filepath.Dir(finalDest), 0755)
			if err := utils.CopyDir(skillPath, finalDest); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Error copying skill: %v\n", err)
				os.Exit(1)
			}

			// Save metadata in repo folder
			meta := skills.RepoMetadata{URL: url, SkillName: skillName}
			skills.SaveMetadata(dest, meta)
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
