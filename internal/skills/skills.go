package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"skillops/internal/config"
)

type Skill struct {
	Name     string // Display name: repo_name/skill_name
	RepoName string
	Path     string // Absolute path to skill directory
}

func Discover() ([]Skill, error) {
	skillsDir := config.SkillsDir

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read skills directory: %w", err)
	}

	var skills []Skill

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		repoName := entry.Name()
		repoPath := filepath.Join(skillsDir, repoName)

		// Check if SKILL.md exists at root level
		rootSkillPath := filepath.Join(repoPath, "SKILL.md")
		if _, err := os.Stat(rootSkillPath); err == nil {
			// Root skill: repo_name/repo_name
			skills = append(skills, Skill{
				Name:     fmt.Sprintf("%s/%s", repoName, repoName),
				RepoName: repoName,
				Path:     repoPath,
			})
		}

		// Scan subdirectories
		subEntries, err := os.ReadDir(repoPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to read directory %s: %v\n", repoPath, err)
			continue
		}

		for _, subEntry := range subEntries {
			if !subEntry.IsDir() {
				continue
			}

			// Rule 3: Container Skill (skills/<folder>/SKILL.md)
			if subEntry.Name() == "skills" {
				containerPath := filepath.Join(repoPath, "skills")
				containerEntries, err := os.ReadDir(containerPath)
				if err != nil {
					continue
				}
				for _, ce := range containerEntries {
					if !ce.IsDir() {
						continue
					}
					skillPath := filepath.Join(containerPath, ce.Name())
					if _, err := os.Stat(filepath.Join(skillPath, "SKILL.md")); err == nil {
						skills = append(skills, Skill{
							Name:     fmt.Sprintf("%s/%s", repoName, ce.Name()),
							RepoName: repoName,
							Path:     skillPath,
						})
					}
				}
				continue
			}

			// Rule 2: Subfolder Skill (<folder>/SKILL.md)
			subPath := filepath.Join(repoPath, subEntry.Name())
			skillMDPath := filepath.Join(subPath, "SKILL.md")

			if _, err := os.Stat(skillMDPath); err == nil {
				skills = append(skills, Skill{
					Name:     fmt.Sprintf("%s/%s", repoName, subEntry.Name()),
					RepoName: repoName,
					Path:     subPath,
				})
			}
		}
	}

	return skills, nil
}

func GetSkillName(skill Skill) string {
	parts := strings.Split(skill.Name, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return skill.Name
}
