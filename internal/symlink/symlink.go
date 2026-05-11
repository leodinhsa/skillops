package symlink

import (
	"fmt"
	"os"
	"path/filepath"

	"skillops/internal/config"
	"skillops/internal/skills"
	"skillops/internal/utils"
	"strings"
)

func EnsureSymlink(skill skills.Skill, agentPath string) error {
	skillName := skills.GetSkillName(skill)

	// Validate skillName to prevent path traversal
	if err := utils.ValidateName(skillName); err != nil {
		return err
	}

	targetPath := filepath.Join(agentPath, skillName)

	// Create parent directory if needed
	if err := os.MkdirAll(agentPath, 0755); err != nil {
		return fmt.Errorf("failed to create agent directory: %w", err)
	}

	// Remove existing symlink or file
	if _, err := os.Lstat(targetPath); err == nil {
		if err := os.Remove(targetPath); err != nil {
			return fmt.Errorf("failed to remove existing symlink: %w", err)
		}
	}

	// Create symlink
	if err := os.Symlink(skill.Path, targetPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

func RemoveSymlink(skillName, agentPath string) error {
	// Validate skillName to prevent path traversal
	if err := utils.ValidateName(skillName); err != nil {
		return err
	}

	targetPath := filepath.Join(agentPath, skillName)

	// Check if it's a symlink
	info, err := os.Lstat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Already removed
		}
		return fmt.Errorf("failed to stat symlink: %w", err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("%s is not a symlink", targetPath)
	}

	if err := os.Remove(targetPath); err != nil {
		return fmt.Errorf("failed to remove symlink: %w", err)
	}

	return nil
}

func IsSymlinkEnabled(skill skills.Skill, agentPath string) bool {
	skillName := skills.GetSkillName(skill)
	targetPath := filepath.Join(agentPath, skillName)

	info, err := os.Lstat(targetPath)
	if err != nil {
		return false
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return false
	}

	// Verify it points to the correct target
	link, err := os.Readlink(targetPath)
	if err != nil {
		return false
	}

	return link == skill.Path
}

func GetEnabledSkills(agentPath string) (map[string]bool, error) {
	enabled := make(map[string]bool)

	entries, err := os.ReadDir(agentPath)
	if err != nil {
		if os.IsNotExist(err) {
			return enabled, nil
		}
		return nil, fmt.Errorf("failed to read agent directory: %w", err)
	}

	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 {
			enabled[entry.Name()] = true
		}
	}

	return enabled, nil
}

// FindAllSkillLinks checks all registered agentics to see if a skill is linked
func FindAllSkillLinks(skillName string) ([]string, error) {
	// Extract short name if repo-prefixed (e.g., repo/skill -> skill)
	shortName := skillName
	parts := strings.Split(skillName, "/")
	if len(parts) >= 2 {
		shortName = parts[len(parts)-1]
	}

	agentics, err := config.GetAgentics()
	if err != nil {
		return nil, err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	var linkedAgentics []string
	for name, relPath := range agentics {
		parts := strings.Split(filepath.ToSlash(relPath), "/")
		if len(parts) == 0 {
			continue
		}
		rootSubDir := parts[0]
		fullPath := filepath.Join(cwd, rootSubDir)

		// If the agentic exists in project root, check if skill is linked
		if _, err := os.Stat(fullPath); err == nil {
			targetPath := filepath.Join(cwd, relPath, shortName)
			if info, err := os.Lstat(targetPath); err == nil {
				if info.Mode()&os.ModeSymlink != 0 {
					linkedAgentics = append(linkedAgentics, name)
				}
			}
		}
	}

	return linkedAgentics, nil
}

// GetToolSkillsDir returns the absolute path to a tool's skills directory in the current project.
// It reads the tool's relative path from global config and joins it with cwd.
func GetToolSkillsDir(tool string) (string, error) {
	relPath, err := config.GetAgenticPath(tool)
	if err != nil {
		return "", fmt.Errorf("unknown tool: %s", tool)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	return filepath.Join(cwd, relPath), nil
}

// CreateSkillSymlink creates a symlink for a full-path skill identity in the given tool's skills directory.
// It uses the custom symlink name from localConfig.SymlinkNames if available, otherwise falls back to the
// short name (final path component). Returns wasCreated=true only when a new symlink is created.
// Returns wasCreated=false, nil for idempotent no-ops (symlink already points to correct target).
func CreateSkillSymlink(identity, tool string, localConfig config.LocalConfig) (bool, error) {
	// 1. Parse identity
	parsed, err := skills.ParseIdentity(identity)
	if err != nil {
		return false, fmt.Errorf("invalid identity: %w", err)
	}

	// 2. Determine symlink name
	symlinkName := localConfig.SymlinkNames[identity]
	if symlinkName == "" {
		symlinkName = parsed.ShortName
	}

	// 3. Construct paths
	globalPath := filepath.Join(config.SkillsDir, filepath.FromSlash(identity))

	toolDir, err := GetToolSkillsDir(tool)
	if err != nil {
		return false, err
	}
	symlinkPath := filepath.Join(toolDir, symlinkName)

	// 4. Verify global skill exists (must contain SKILL.md)
	skillMdPath := filepath.Join(globalPath, "SKILL.md")
	if _, err := os.Stat(skillMdPath); err != nil {
		return false, fmt.Errorf("skill not found in global store: %s", identity)
	}

	// 5. Ensure tool skills directory exists
	if err := os.MkdirAll(toolDir, 0755); err != nil {
		return false, fmt.Errorf("failed to create tool skills directory: %w", err)
	}

	// 6. Check for conflicts
	if info, err := os.Lstat(symlinkPath); err == nil {
		// Something exists at the symlink path
		if info.Mode()&os.ModeSymlink != 0 {
			existingTarget, err := os.Readlink(symlinkPath)
			if err != nil {
				return false, fmt.Errorf("failed to read existing symlink: %w", err)
			}
			if existingTarget == globalPath {
				return false, nil // Already linked correctly (idempotent)
			}
			return false, fmt.Errorf("symlink conflict: %s already points to %s", symlinkName, existingTarget)
		}
		return false, fmt.Errorf("symlink conflict: %s already exists and is not a symlink", symlinkName)
	}

	// 7. Create symlink
	if err := os.Symlink(globalPath, symlinkPath); err != nil {
		return false, fmt.Errorf("failed to create symlink: %w", err)
	}

	return true, nil
}

func FindSkillPath(skillName string) (string, error) {
	skillsDir := config.SkillsDir

	// 1. Try direct path first (in case it's repo-prefixed)
	directPath := filepath.Join(skillsDir, skillName)
	if _, err := os.Stat(filepath.Join(directPath, "SKILL.md")); err == nil {
		return directPath, nil
	}

	// 2. Extract short name for deep search
	shortName := skillName
	parts := strings.Split(skillName, "/")
	if len(parts) >= 2 {
		shortName = parts[len(parts)-1]
	}

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		repoName := entry.Name()
		repoPath := filepath.Join(skillsDir, repoName)

		// Check if SKILL.md exists at root level
		if _, err := os.Stat(filepath.Join(repoPath, "SKILL.md")); err == nil {
			if repoName == shortName {
				return repoPath, nil
			}
		}

		// Scan subdirectories
		subEntries, _ := os.ReadDir(repoPath)
		for _, subEntry := range subEntries {
			if !subEntry.IsDir() {
				continue
			}

			if subEntry.Name() == "skills" {
				containerPath := filepath.Join(repoPath, "skills")
				containerEntries, _ := os.ReadDir(containerPath)
				for _, ce := range containerEntries {
					if ce.IsDir() && ce.Name() == shortName {
						if _, err := os.Stat(filepath.Join(containerPath, ce.Name(), "SKILL.md")); err == nil {
							return filepath.Join(containerPath, ce.Name()), nil
						}
					}
				}
			} else if subEntry.Name() == shortName {
				subPath := filepath.Join(repoPath, subEntry.Name())
				if _, err := os.Stat(filepath.Join(subPath, "SKILL.md")); err == nil {
					return subPath, nil
				}
			}
		}
	}
	return "", fmt.Errorf("skill '%s' not found in %s", skillName, skillsDir)
}
