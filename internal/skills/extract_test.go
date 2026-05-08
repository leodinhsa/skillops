package skills

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// createTestGitRepo creates a temporary git repository with a skill at the given pathInRepo.
// Returns the path to the bare repo (usable as a clone URL).
func createTestGitRepo(t *testing.T, pathInRepo string) string {
	t.Helper()

	// Create a temp dir for the source repo
	srcDir := t.TempDir()

	// Initialize git repo
	runGit(t, srcDir, "init")
	runGit(t, srcDir, "config", "user.email", "test@test.com")
	runGit(t, srcDir, "config", "user.name", "Test")

	// Create skill at pathInRepo
	skillDir := filepath.Join(srcDir, filepath.FromSlash(pathInRepo))
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Test Skill\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Commit
	runGit(t, srcDir, "add", ".")
	runGit(t, srcDir, "commit", "-m", "initial commit")

	// Create a bare clone to use as the "remote"
	bareDir := t.TempDir()
	runGit(t, "", "clone", "--bare", srcDir, bareDir)

	return "file://" + bareDir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\noutput: %s", args, err, output)
	}
}

func TestPullSkillFromURL_ValidSkill(t *testing.T) {
	// Create a test repo with a skill at "skills/logger"
	repoURL := createTestGitRepo(t, "skills/logger")

	// Pull the skill
	destDir := filepath.Join(t.TempDir(), "github.com", "test", "repo", "skills", "logger")
	err := PullSkillFromURL(repoURL, "skills/logger", destDir)
	if err != nil {
		t.Fatalf("PullSkillFromURL error: %v", err)
	}

	// Verify skill was copied
	if _, err := os.Stat(filepath.Join(destDir, "SKILL.md")); err != nil {
		t.Error("SKILL.md not found in destination")
	}
	if _, err := os.Stat(filepath.Join(destDir, "main.go")); err != nil {
		t.Error("main.go not found in destination")
	}

	// Verify metadata was saved
	if !HasSkillMetadata(destDir) {
		t.Error("expected skill metadata to be saved")
	}

	meta, err := LoadSkillMetadata(destDir)
	if err != nil {
		t.Fatalf("LoadSkillMetadata error: %v", err)
	}
	if meta.RepoURL != repoURL {
		t.Errorf("metadata RepoURL = %q, want %q", meta.RepoURL, repoURL)
	}
	if meta.PathInRepo != "skills/logger" {
		t.Errorf("metadata PathInRepo = %q, want %q", meta.PathInRepo, "skills/logger")
	}
	if meta.CommitHash == "" {
		t.Error("metadata CommitHash should not be empty")
	}
	if meta.PulledAt.IsZero() {
		t.Error("metadata PulledAt should not be zero")
	}
}

func TestPullSkillFromURL_NestedPath(t *testing.T) {
	// Create a test repo with a deeply nested skill
	repoURL := createTestGitRepo(t, "backend/services/api/auth")

	destDir := filepath.Join(t.TempDir(), "github.com", "company", "monorepo", "backend", "services", "api", "auth")
	err := PullSkillFromURL(repoURL, "backend/services/api/auth", destDir)
	if err != nil {
		t.Fatalf("PullSkillFromURL error: %v", err)
	}

	// Verify skill was copied
	if _, err := os.Stat(filepath.Join(destDir, "SKILL.md")); err != nil {
		t.Error("SKILL.md not found in destination")
	}

	// Verify metadata
	meta, err := LoadSkillMetadata(destDir)
	if err != nil {
		t.Fatalf("LoadSkillMetadata error: %v", err)
	}
	if meta.PathInRepo != "backend/services/api/auth" {
		t.Errorf("metadata PathInRepo = %q, want %q", meta.PathInRepo, "backend/services/api/auth")
	}
}

func TestPullSkillFromURL_SkillNotFound(t *testing.T) {
	// Create a test repo with a skill at a different path
	repoURL := createTestGitRepo(t, "skills/logger")

	destDir := filepath.Join(t.TempDir(), "dest")
	err := PullSkillFromURL(repoURL, "skills/nonexistent", destDir)
	if err == nil {
		t.Fatal("expected error for skill not found, got nil")
	}
	if !contains(err.Error(), "skill not found at path") {
		t.Errorf("error should mention 'skill not found at path', got: %v", err)
	}

	// Verify destination was NOT created (no partial state)
	if _, err := os.Stat(destDir); err == nil {
		t.Error("destination should not exist after failed pull")
	}
}

func TestPullSkillFromURL_Idempotent(t *testing.T) {
	// Create a test repo
	repoURL := createTestGitRepo(t, "skills/logger")

	destDir := filepath.Join(t.TempDir(), "github.com", "test", "repo", "skills", "logger")

	// Pull twice — second should succeed (overwrite)
	if err := PullSkillFromURL(repoURL, "skills/logger", destDir); err != nil {
		t.Fatalf("first PullSkillFromURL error: %v", err)
	}
	if err := PullSkillFromURL(repoURL, "skills/logger", destDir); err != nil {
		t.Fatalf("second PullSkillFromURL error: %v", err)
	}

	// Verify skill still exists
	if _, err := os.Stat(filepath.Join(destDir, "SKILL.md")); err != nil {
		t.Error("SKILL.md not found after second pull")
	}
}

func TestPullSkillFromURL_CleansUpTempOnError(t *testing.T) {
	// Use an invalid URL that will fail to clone
	destDir := filepath.Join(t.TempDir(), "dest")
	err := PullSkillFromURL("https://invalid.example.com/nonexistent/repo", "skills/logger", destDir)
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}

	// Verify no .tmp file left behind
	if _, err := os.Stat(destDir + ".tmp"); err == nil {
		t.Error("temp destination should be cleaned up on error")
	}
}
