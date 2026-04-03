package symlink

import (
	"os"
	"path/filepath"
	"testing"

	"skillops/internal/config"
	"skillops/internal/skills"
)

func makeSkill(t *testing.T, dir, repoName, skillName string) skills.Skill {
	t.Helper()
	skillPath := filepath.Join(dir, repoName, skillName)
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte("# skill"), 0644); err != nil {
		t.Fatal(err)
	}
	return skills.Skill{
		Name:     repoName + "/" + skillName,
		RepoName: repoName,
		Path:     skillPath,
	}
}

func overrideSkillsDir(t *testing.T, dir string) {
	t.Helper()
	orig := config.SkillsDir
	config.SkillsDir = dir
	t.Cleanup(func() { config.SkillsDir = orig })
}

func TestEnsureSymlink_Creates(t *testing.T) {
	tmp := t.TempDir()
	skill := makeSkill(t, tmp, "repo", "logger")
	agentPath := filepath.Join(tmp, ".kiro", "skills")

	if err := EnsureSymlink(skill, agentPath); err != nil {
		t.Fatalf("EnsureSymlink error: %v", err)
	}

	linkPath := filepath.Join(agentPath, "logger")
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("symlink not created: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink, got regular file/dir")
	}

	target, _ := os.Readlink(linkPath)
	if target != skill.Path {
		t.Errorf("symlink target = %q, want %q", target, skill.Path)
	}
}

func TestEnsureSymlink_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	skill := makeSkill(t, tmp, "repo", "logger")
	agentPath := filepath.Join(tmp, ".kiro", "skills")

	if err := EnsureSymlink(skill, agentPath); err != nil {
		t.Fatal(err)
	}
	if err := EnsureSymlink(skill, agentPath); err != nil {
		t.Fatalf("second EnsureSymlink error: %v", err)
	}
}

func TestEnsureSymlink_InvalidName(t *testing.T) {
	tmp := t.TempDir()
	// Skill name with path traversal after the slash
	skill := skills.Skill{Name: "repo/../evil", RepoName: "repo", Path: tmp}
	agentPath := filepath.Join(tmp, "agent")

	err := EnsureSymlink(skill, agentPath)
	if err == nil {
		t.Error("expected error for path traversal in skill name")
	}
}

func TestRemoveSymlink(t *testing.T) {
	tmp := t.TempDir()
	skill := makeSkill(t, tmp, "repo", "logger")
	agentPath := filepath.Join(tmp, ".kiro", "skills")

	if err := EnsureSymlink(skill, agentPath); err != nil {
		t.Fatal(err)
	}

	if err := RemoveSymlink("logger", agentPath); err != nil {
		t.Fatalf("RemoveSymlink error: %v", err)
	}

	linkPath := filepath.Join(agentPath, "logger")
	if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
		t.Error("symlink should have been removed")
	}
}

func TestRemoveSymlink_AlreadyGone(t *testing.T) {
	tmp := t.TempDir()
	agentPath := filepath.Join(tmp, ".kiro", "skills")
	if err := os.MkdirAll(agentPath, 0755); err != nil {
		t.Fatal(err)
	}

	if err := RemoveSymlink("nonexistent", agentPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoveSymlink_NotASymlink(t *testing.T) {
	tmp := t.TempDir()
	agentPath := filepath.Join(tmp, ".kiro", "skills")
	if err := os.MkdirAll(agentPath, 0755); err != nil {
		t.Fatal(err)
	}

	realFile := filepath.Join(agentPath, "realfile")
	if err := os.WriteFile(realFile, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	err := RemoveSymlink("realfile", agentPath)
	if err == nil {
		t.Error("expected error when target is not a symlink")
	}
}

func TestIsSymlinkEnabled(t *testing.T) {
	tmp := t.TempDir()
	skill := makeSkill(t, tmp, "repo", "logger")
	agentPath := filepath.Join(tmp, ".kiro", "skills")

	if IsSymlinkEnabled(skill, agentPath) {
		t.Error("should not be enabled before linking")
	}

	if err := EnsureSymlink(skill, agentPath); err != nil {
		t.Fatal(err)
	}

	if !IsSymlinkEnabled(skill, agentPath) {
		t.Error("should be enabled after linking")
	}
}

func TestIsSymlinkEnabled_WrongTarget(t *testing.T) {
	tmp := t.TempDir()
	skill := makeSkill(t, tmp, "repo", "logger")
	agentPath := filepath.Join(tmp, ".kiro", "skills")
	if err := os.MkdirAll(agentPath, 0755); err != nil {
		t.Fatal(err)
	}

	wrongTarget := filepath.Join(tmp, "other")
	if err := os.MkdirAll(wrongTarget, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(wrongTarget, filepath.Join(agentPath, "logger")); err != nil {
		t.Fatal(err)
	}

	if IsSymlinkEnabled(skill, agentPath) {
		t.Error("should not be enabled when pointing to wrong target")
	}
}

func TestGetEnabledSkills(t *testing.T) {
	tmp := t.TempDir()
	skill1 := makeSkill(t, tmp, "repo", "logger")
	skill2 := makeSkill(t, tmp, "repo", "auth")
	agentPath := filepath.Join(tmp, ".kiro", "skills")

	if err := EnsureSymlink(skill1, agentPath); err != nil {
		t.Fatal(err)
	}
	if err := EnsureSymlink(skill2, agentPath); err != nil {
		t.Fatal(err)
	}

	enabled, err := GetEnabledSkills(agentPath)
	if err != nil {
		t.Fatalf("GetEnabledSkills error: %v", err)
	}
	if !enabled["logger"] {
		t.Error("logger should be enabled")
	}
	if !enabled["auth"] {
		t.Error("auth should be enabled")
	}
}

func TestGetEnabledSkills_EmptyDir(t *testing.T) {
	tmp := t.TempDir()
	agentPath := filepath.Join(tmp, ".kiro", "skills")

	enabled, err := GetEnabledSkills(agentPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(enabled) != 0 {
		t.Errorf("expected empty map, got %v", enabled)
	}
}

func TestFindSkillPath_DirectSubfolder(t *testing.T) {
	tmp := t.TempDir()
	overrideSkillsDir(t, tmp)

	skillPath := filepath.Join(tmp, "my-repo", "logger")
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := FindSkillPath("logger")
	if err != nil {
		t.Fatalf("FindSkillPath error: %v", err)
	}
	if got != skillPath {
		t.Errorf("got %q, want %q", got, skillPath)
	}
}

func TestFindSkillPath_ContainerSkill(t *testing.T) {
	tmp := t.TempDir()
	overrideSkillsDir(t, tmp)

	skillPath := filepath.Join(tmp, "my-repo", "skills", "formatter")
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := FindSkillPath("formatter")
	if err != nil {
		t.Fatalf("FindSkillPath error: %v", err)
	}
	if got != skillPath {
		t.Errorf("got %q, want %q", got, skillPath)
	}
}

func TestFindSkillPath_NotFound(t *testing.T) {
	tmp := t.TempDir()
	overrideSkillsDir(t, tmp)

	_, err := FindSkillPath("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent skill")
	}
}

func TestFindSkillPath_ByFullIdentity(t *testing.T) {
	tmp := t.TempDir()
	overrideSkillsDir(t, tmp)

	skillPath := filepath.Join(tmp, "my-repo", "logger")
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	// FindSkillPath with "my-repo/logger" full identity
	got, err := FindSkillPath("my-repo/logger")
	if err != nil {
		t.Fatalf("FindSkillPath error: %v", err)
	}
	if got != skillPath {
		t.Errorf("got %q, want %q", got, skillPath)
	}
}
