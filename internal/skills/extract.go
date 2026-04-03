package skills

import (
	"fmt"
	"os"
	"path/filepath"

	"skillops/internal/git"
	"skillops/internal/utils"
)

// PullSkillFromURL clones repoURL into a temp dir, finds the skill folder
// matching skillName using 3-rule discovery, copies it to destSkillDir,
// saves metadata.json, and cleans up the temp dir.
//
// Discovery rules (in order):
//  1. Root skill: SKILL.md at repo root → skill folder = repo root
//  2. Container skill: skills/<skillName>/SKILL.md → skill folder = skills/<skillName>
//  3. Direct subfolder: <skillName>/SKILL.md → skill folder = <skillName>
//
// destSkillDir is the final path where the skill folder will be placed,
// e.g. ~/.skillops/skills/<repoName>/<skillName>.
func PullSkillFromURL(repoURL, skillName, destSkillDir string) error {
	tempDir, err := os.MkdirTemp("", "skillops-pull-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := git.Clone(repoURL, tempDir); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	skillPath := findSkillPath(tempDir, skillName)
	if skillPath == "" {
		return fmt.Errorf("skill '%s' not found in repository", skillName)
	}

	if err := os.MkdirAll(filepath.Dir(destSkillDir), 0755); err != nil {
		return fmt.Errorf("failed to create destination parent dir: %w", err)
	}

	if err := utils.CopyDir(skillPath, destSkillDir); err != nil {
		return fmt.Errorf("failed to copy skill: %w", err)
	}

	meta := RepoMetadata{URL: repoURL, SkillName: skillName}
	if err := SaveMetadata(filepath.Dir(destSkillDir), meta); err != nil {
		// Non-fatal: warn but don't fail the pull
		fmt.Fprintf(os.Stderr, "Warning: failed to save metadata: %v\n", err)
	}

	return nil
}

// findSkillPath applies the 3-rule discovery to locate a skill directory
// within a cloned repo at repoRoot.
func findSkillPath(repoRoot, skillName string) string {
	// Rule 1: Root skill — SKILL.md at repo root
	if _, err := os.Stat(filepath.Join(repoRoot, "SKILL.md")); err == nil {
		return repoRoot
	}

	// Rule 2: Container skill — skills/<skillName>/SKILL.md
	containerPath := filepath.Join(repoRoot, "skills", skillName)
	if _, err := os.Stat(filepath.Join(containerPath, "SKILL.md")); err == nil {
		return containerPath
	}

	// Rule 3: Direct subfolder — <skillName>/SKILL.md
	directPath := filepath.Join(repoRoot, skillName)
	if _, err := os.Stat(filepath.Join(directPath, "SKILL.md")); err == nil {
		return directPath
	}

	return ""
}
