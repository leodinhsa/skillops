package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"skillops/internal/config"
)

// ParsedIdentity represents a parsed full-path skill identity
// Format: <host>/<repo-path>/<path-to-skill>
// Example: github.com/anthropics/skills/skills/skill-creator
// Example: gitlab.common.datumhq.com/datumhq-consulting-vn/management/datum-skills/software-skills/skills/logger
//
// Note: The boundary between repo-path and path-to-skill is NOT determined by parsing.
// It is determined by registry URL prefix matching at runtime.
type ParsedIdentity struct {
	Full      string // Full identity: "github.com/anthropics/skills/skills/logger"
	Host      string // Git host: "github.com"
	Path      string // Everything after host: "anthropics/skills/skills/logger"
	ShortName string // Final component for symlink: "logger"
}

// ParseIdentity parses a full-path skill identity into its components
// Returns error if identity is invalid (< 3 components, path traversal, empty components)
func ParseIdentity(identity string) (*ParsedIdentity, error) {
	// Split identity on "/"
	parts := strings.Split(identity, "/")

	// Validate minimum components (host/something/skill)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid identity '%s': need at least host/path/skill (minimum 3 components)", identity)
	}

	// Validate all components for path traversal and empty values
	for i, part := range parts {
		if part == "" {
			return nil, fmt.Errorf("invalid identity '%s': component %d is empty", identity, i+1)
		}
		if part == "." {
			return nil, fmt.Errorf("invalid identity '%s': component %d cannot be '.'", identity, i+1)
		}
		if part == ".." {
			return nil, fmt.Errorf("invalid identity '%s': component %d cannot be '..' (path traversal attempt)", identity, i+1)
		}
	}

	// Extract components
	host := parts[0]
	path := strings.Join(parts[1:], "/")
	shortName := parts[len(parts)-1]

	return &ParsedIdentity{
		Full:      identity,
		Host:      host,
		Path:      path,
		ShortName: shortName,
	}, nil
}

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

type RepoMetadata struct {
	URL       string `json:"url"`
	SkillName string `json:"skill_name,omitempty"`
}

func SaveMetadata(repoPath string, meta RepoMetadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(repoPath, "metadata.json"), data, 0644)
}

func LoadMetadata(repoPath string) (RepoMetadata, error) {
	data, err := os.ReadFile(filepath.Join(repoPath, "metadata.json"))
	if err != nil {
		return RepoMetadata{}, err
	}
	var meta RepoMetadata
	err = json.Unmarshal(data, &meta)
	return meta, err
}
