package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"skillops/internal/config"
	"skillops/internal/git"
	"skillops/internal/skills"
	"skillops/internal/tui"
	"skillops/internal/utils"

	"github.com/spf13/cobra"
)

var (
	updateSkillName string
)

var updateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update all or a specific skill from the source repository",
	GroupID: "skill",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(tui.TitleStyle.Render(" UPDATING SKILLS "))

		if updateSkillName != "" {
			updateSpecificSkill(updateSkillName)
		} else {
			updateAllSkills()
		}
	},
}

func updateSpecificSkill(name string) {
	allSkills, err := skills.Discover()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error discovering skills: %v\n", err)
		return
	}

	var targetSkill *skills.Skill
	for _, s := range allSkills {
		if skills.GetSkillName(s) == name {
			targetSkill = &s
			break
		}
	}

	if targetSkill == nil {
		fmt.Fprintf(os.Stderr, "❌ Error: skill '%s' not found locally.\n", name)
		return
	}

	repoPath := filepath.Join(config.SkillsDir, targetSkill.RepoName)
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err == nil {
		updateRepo(targetSkill.RepoName, repoPath)
	} else if _, err := os.Stat(filepath.Join(repoPath, "metadata.json")); err == nil {
		updateSpecificRepo(targetSkill.RepoName, repoPath)
	} else {
		fmt.Fprintf(os.Stderr, "❌ Error: no git repository or metadata found for %s\n", targetSkill.RepoName)
	}
}

func updateAllSkills() {
	entries, err := os.ReadDir(config.SkillsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading skills directory: %v\n", err)
		return
	}

	updatedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			repoPath := filepath.Join(config.SkillsDir, entry.Name())
			// Check if it's a repo or has metadata
			if _, err := os.Stat(filepath.Join(repoPath, ".git")); err == nil {
				updateRepo(entry.Name(), repoPath)
				updatedCount++
			} else if _, err := os.Stat(filepath.Join(repoPath, "metadata.json")); err == nil {
				updateSpecificRepo(entry.Name(), repoPath)
				updatedCount++
			}
		}
	}

	if updatedCount == 0 {
		fmt.Println("No skills found to update.")
	} else {
		fmt.Printf("\n✨ Finished updating %d skill repositories.\n", updatedCount)
	}
}

func updateRepo(name, path string) {
	fmt.Printf("Updating %s (git)...\n", tui.SuccessStyle.Render(name))

	gitCmd := exec.Command("git", "-C", path, "pull", "--rebase")
	output, err := gitCmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ❌ Failed to update %s: %v\n", name, err)
		fmt.Fprintf(os.Stderr, "  Output: %s\n", string(output))
		return
	}

	fmt.Printf("  %s\n", tui.DimStyle.Render("Successfully pulled latest changes."))
}

func updateSpecificRepo(name, path string) {
	meta, err := skills.LoadMetadata(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ❌ Error loading metadata for %s: %v\n", name, err)
		return
	}

	fmt.Printf("Updating %s (specific: %s)...\n", tui.SuccessStyle.Render(name), meta.SkillName)

	// Clone to temp
	tempDest, err := os.MkdirTemp("", "skillops-update-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ❌ Error creating temp dir: %v\n", err)
		return
	}
	defer os.RemoveAll(tempDest)

	if err := git.Clone(meta.URL, tempDest); err != nil {
		fmt.Fprintf(os.Stderr, "  ❌ Error cloning for update: %v\n", err)
		return
	}

	// Find skill folder
	skillPath := ""
	if meta.SkillName == "" || meta.SkillName == name {
		// Root skill
		if _, err := os.Stat(filepath.Join(tempDest, "SKILL.md")); err == nil {
			skillPath = tempDest
		}
	}
	if skillPath == "" {
		subfolders := []string{meta.SkillName, filepath.Join("skills", meta.SkillName)}
		for _, sub := range subfolders {
			candidate := filepath.Join(tempDest, sub)
			if _, err := os.Stat(filepath.Join(candidate, "SKILL.md")); err == nil {
				skillPath = candidate
				break
			}
		}
	}

	if skillPath == "" {
		fmt.Fprintf(os.Stderr, "  ❌ Error: skill '%s' not found in remote repository\n", meta.SkillName)
		return
	}

	// Target: ~/.skillops/skills/<repo_name>/<skill_name>
	finalDest := path
	if meta.SkillName != "" {
		finalDest = filepath.Join(path, meta.SkillName)
	}

	// Remove old and copy new
	os.RemoveAll(finalDest)
	os.MkdirAll(filepath.Dir(finalDest), 0755)
	if err := utils.CopyDir(skillPath, finalDest); err != nil {
		fmt.Fprintf(os.Stderr, "  ❌ Error copying updated skill: %v\n", err)
		return
	}

	fmt.Printf("  %s\n", tui.DimStyle.Render("Successfully updated skill from remote."))
}

func init() {
	updateCmd.Flags().StringVarP(&updateSkillName, "skill", "s", "", "Update a specific skill")
	rootCmd.AddCommand(updateCmd)
}
