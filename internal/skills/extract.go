package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"skillops/internal/git"
	"skillops/internal/utils"
)

// PullSkillFromURL clones repoURL into a temp dir, extracts the skill at pathInRepo,
// copies it atomically (temp-then-rename) to destSkillDir, and saves skill metadata.
//
// Parameters:
//   - repoURL: Full git repository URL (e.g., "https://github.com/anthropics/skills")
//   - pathInRepo: Path from repo root to skill (e.g., "skills/logger")
//   - destSkillDir: Destination path in global store
//     (e.g., "~/.skillops/skills/github.com/anthropics/skills/skills/logger")
//
// The function uses shallow clone (--depth 1) for efficiency and atomic copy
// (temp-then-rename) to prevent partial state.
func PullSkillFromURL(repoURL, pathInRepo, destSkillDir string) error {
	// 1. Create temporary directory
	tempDir, err := os.MkdirTemp("", "skillops-pull-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 2. Clone repository (shallow clone for efficiency)
	if err := git.Clone(repoURL, tempDir); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// 3. Construct skill source path
	skillSource := filepath.Join(tempDir, filepath.FromSlash(pathInRepo))

	// 4. Verify skill exists (must contain SKILL.md)
	if _, err := os.Stat(filepath.Join(skillSource, "SKILL.md")); err != nil {
		return fmt.Errorf("skill not found at path: %s", pathInRepo)
	}

	// 5. Create destination parent directories
	if err := os.MkdirAll(filepath.Dir(destSkillDir), 0755); err != nil {
		return fmt.Errorf("failed to create destination parent: %w", err)
	}

	// 6. Copy skill directory atomically (temp-then-rename)
	tempDest := destSkillDir + ".tmp"
	// Clean up any leftover temp from a previous failed attempt
	os.RemoveAll(tempDest)

	if err := utils.CopyDir(skillSource, tempDest); err != nil {
		os.RemoveAll(tempDest)
		return fmt.Errorf("failed to copy skill: %w", err)
	}

	// Remove existing destination if it exists (for idempotent re-pulls)
	if _, err := os.Stat(destSkillDir); err == nil {
		os.RemoveAll(destSkillDir)
	}

	// Atomic rename
	if err := os.Rename(tempDest, destSkillDir); err != nil {
		os.RemoveAll(tempDest)
		return fmt.Errorf("failed to finalize skill: %w", err)
	}

	// 7. Get commit hash from cloned repo
	commitHash := git.GetLatestCommit(tempDir)

	// 8. Save skill metadata (non-fatal: warn but don't fail)
	meta := SkillMetadata{
		RepoURL:    repoURL,
		PathInRepo: pathInRepo,
		PulledAt:   time.Now(),
		CommitHash: commitHash,
	}
	if err := SaveSkillMetadata(destSkillDir, meta); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save metadata: %v\n", err)
	}

	return nil
}
