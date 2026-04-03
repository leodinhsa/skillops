package skills

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"skillops/internal/config"
)

// overrideSkillsDir temporarily sets config.SkillsDir for testing.
func overrideSkillsDir(t *testing.T, dir string) {
	t.Helper()
	orig := config.SkillsDir
	config.SkillsDir = dir
	t.Cleanup(func() { config.SkillsDir = orig })
}

func mkfile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDiscover_RootSkill(t *testing.T) {
	tmp := t.TempDir()
	overrideSkillsDir(t, tmp)

	// repo with SKILL.md at root
	mkfile(t, filepath.Join(tmp, "my-repo", "SKILL.md"), "# Root skill")

	skills, err := Discover()
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].Name != "my-repo/my-repo" {
		t.Errorf("Name = %q, want %q", skills[0].Name, "my-repo/my-repo")
	}
	if skills[0].RepoName != "my-repo" {
		t.Errorf("RepoName = %q, want %q", skills[0].RepoName, "my-repo")
	}
}

func TestDiscover_SubfolderSkill(t *testing.T) {
	tmp := t.TempDir()
	overrideSkillsDir(t, tmp)

	mkfile(t, filepath.Join(tmp, "my-repo", "logger", "SKILL.md"), "# Logger")
	mkfile(t, filepath.Join(tmp, "my-repo", "auth", "SKILL.md"), "# Auth")

	skills, err := Discover()
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d: %v", len(skills), skills)
	}

	names := []string{skills[0].Name, skills[1].Name}
	sort.Strings(names)
	if names[0] != "my-repo/auth" || names[1] != "my-repo/logger" {
		t.Errorf("unexpected names: %v", names)
	}
}

func TestDiscover_ContainerSkill(t *testing.T) {
	tmp := t.TempDir()
	overrideSkillsDir(t, tmp)

	// skills/<name>/SKILL.md pattern
	mkfile(t, filepath.Join(tmp, "my-repo", "skills", "formatter", "SKILL.md"), "# Formatter")

	skills, err := Discover()
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].Name != "my-repo/formatter" {
		t.Errorf("Name = %q, want %q", skills[0].Name, "my-repo/formatter")
	}
}

func TestDiscover_MultipleRepos(t *testing.T) {
	tmp := t.TempDir()
	overrideSkillsDir(t, tmp)

	mkfile(t, filepath.Join(tmp, "repo-a", "skill1", "SKILL.md"), "")
	mkfile(t, filepath.Join(tmp, "repo-b", "skill2", "SKILL.md"), "")
	mkfile(t, filepath.Join(tmp, "repo-b", "skill3", "SKILL.md"), "")

	skills, err := Discover()
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if len(skills) != 3 {
		t.Fatalf("expected 3 skills, got %d", len(skills))
	}
}

func TestDiscover_IgnoresNonSkillDirs(t *testing.T) {
	tmp := t.TempDir()
	overrideSkillsDir(t, tmp)

	// dir without SKILL.md — should be ignored
	if err := os.MkdirAll(filepath.Join(tmp, "my-repo", "not-a-skill"), 0755); err != nil {
		t.Fatal(err)
	}
	mkfile(t, filepath.Join(tmp, "my-repo", "not-a-skill", "README.md"), "no skill here")

	skills, err := Discover()
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(skills))
	}
}

func TestDiscover_EmptyDir(t *testing.T) {
	tmp := t.TempDir()
	overrideSkillsDir(t, tmp)

	skills, err := Discover()
	if err != nil {
		t.Fatalf("Discover() error: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(skills))
	}
}

func TestGetSkillName(t *testing.T) {
	tests := []struct {
		skill Skill
		want  string
	}{
		{Skill{Name: "repo/logger"}, "logger"},
		{Skill{Name: "repo/auth-agent"}, "auth-agent"},
		{Skill{Name: "single"}, "single"},
	}
	for _, tt := range tests {
		got := GetSkillName(tt.skill)
		if got != tt.want {
			t.Errorf("GetSkillName(%q) = %q, want %q", tt.skill.Name, got, tt.want)
		}
	}
}

func TestSaveAndLoadMetadata(t *testing.T) {
	tmp := t.TempDir()

	meta := RepoMetadata{URL: "https://github.com/user/repo", SkillName: "logger"}
	if err := SaveMetadata(tmp, meta); err != nil {
		t.Fatalf("SaveMetadata error: %v", err)
	}

	got, err := LoadMetadata(tmp)
	if err != nil {
		t.Fatalf("LoadMetadata error: %v", err)
	}
	if got.URL != meta.URL || got.SkillName != meta.SkillName {
		t.Errorf("got %+v, want %+v", got, meta)
	}
}

func TestLoadMetadata_Missing(t *testing.T) {
	tmp := t.TempDir()
	_, err := LoadMetadata(tmp)
	if err == nil {
		t.Error("expected error for missing metadata.json, got nil")
	}
}
