package skills

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// TestParseIdentity_ValidIdentities tests parsing of valid full-path identities
func TestParseIdentity_ValidIdentities(t *testing.T) {
	tests := []struct {
		name          string
		identity      string
		wantHost      string
		wantPath      string
		wantShortName string
	}{
		{
			name:          "3 components (minimum)",
			identity:      "github.com/anthropics/logger",
			wantHost:      "github.com",
			wantPath:      "anthropics/logger",
			wantShortName: "logger",
		},
		{
			name:          "4 components",
			identity:      "github.com/anthropics/skills/logger",
			wantHost:      "github.com",
			wantPath:      "anthropics/skills/logger",
			wantShortName: "logger",
		},
		{
			name:          "5 components (nested)",
			identity:      "github.com/anthropics/skills/skills/logger",
			wantHost:      "github.com",
			wantPath:      "anthropics/skills/skills/logger",
			wantShortName: "logger",
		},
		{
			name:          "deeply nested (7 components)",
			identity:      "github.com/company/monorepo/backend/services/api/auth",
			wantHost:      "github.com",
			wantPath:      "company/monorepo/backend/services/api/auth",
			wantShortName: "auth",
		},
		{
			name:          "multi-level groups (GitLab)",
			identity:      "gitlab.common.datumhq.com/datumhq-consulting-vn/management/datum-skills/software-skills/skills/logger",
			wantHost:      "gitlab.common.datumhq.com",
			wantPath:      "datumhq-consulting-vn/management/datum-skills/software-skills/skills/logger",
			wantShortName: "logger",
		},
		{
			name:          "self-hosted GitLab",
			identity:      "gitlab.company.internal/team/backend/api-skills/database/migrations",
			wantHost:      "gitlab.company.internal",
			wantPath:      "team/backend/api-skills/database/migrations",
			wantShortName: "migrations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseIdentity(tt.identity)
			if err != nil {
				t.Fatalf("ParseIdentity(%q) unexpected error: %v", tt.identity, err)
			}
			if parsed.Full != tt.identity {
				t.Errorf("Full = %q, want %q", parsed.Full, tt.identity)
			}
			if parsed.Host != tt.wantHost {
				t.Errorf("Host = %q, want %q", parsed.Host, tt.wantHost)
			}
			if parsed.Path != tt.wantPath {
				t.Errorf("Path = %q, want %q", parsed.Path, tt.wantPath)
			}
			if parsed.ShortName != tt.wantShortName {
				t.Errorf("ShortName = %q, want %q", parsed.ShortName, tt.wantShortName)
			}
		})
	}
}

// TestParseIdentity_InvalidComponentCount tests that identities with < 3 components are rejected
func TestParseIdentity_InvalidComponentCount(t *testing.T) {
	tests := []struct {
		name     string
		identity string
	}{
		{
			name:     "0 components (empty string)",
			identity: "",
		},
		{
			name:     "1 component",
			identity: "github.com",
		},
		{
			name:     "2 components",
			identity: "github.com/anthropics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseIdentity(tt.identity)
			if err == nil {
				t.Errorf("ParseIdentity(%q) expected error for < 3 components, got nil (parsed: %+v)", tt.identity, parsed)
			}
			if parsed != nil {
				t.Errorf("ParseIdentity(%q) expected nil result on error, got %+v", tt.identity, parsed)
			}
			// Verify error message mentions component count
			if err != nil && !contains(err.Error(), "minimum 3 components") {
				t.Errorf("ParseIdentity(%q) error message should mention 'minimum 3 components', got: %v", tt.identity, err)
			}
		})
	}
}

// TestParseIdentity_InvalidComponents tests that path traversal and empty components are rejected
func TestParseIdentity_InvalidComponents(t *testing.T) {
	tests := []struct {
		name     string
		identity string
		wantErr  string
	}{
		{
			name:     "empty component in middle",
			identity: "github.com//skills/logger",
			wantErr:  "empty",
		},
		{
			name:     "dot component",
			identity: "github.com/./skills/logger",
			wantErr:  "cannot be '.'",
		},
		{
			name:     "double-dot component (path traversal)",
			identity: "github.com/../skills/logger",
			wantErr:  "cannot be '..'",
		},
		{
			name:     "double-dot in path",
			identity: "github.com/anthropics/skills/../logger",
			wantErr:  "cannot be '..'",
		},
		{
			name:     "empty component at end (trailing slash)",
			identity: "github.com/anthropics/skills/",
			wantErr:  "empty",
		},
		{
			name:     "empty host",
			identity: "/anthropics/skills/logger",
			wantErr:  "empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseIdentity(tt.identity)
			if err == nil {
				t.Errorf("ParseIdentity(%q) expected error, got nil (parsed: %+v)", tt.identity, parsed)
			}
			if parsed != nil {
				t.Errorf("ParseIdentity(%q) expected nil result on error, got %+v", tt.identity, parsed)
			}
			if err != nil && !contains(err.Error(), tt.wantErr) {
				t.Errorf("ParseIdentity(%q) error should contain %q, got: %v", tt.identity, tt.wantErr, err)
			}
		})
	}
}

// contains checks if a string contains a substring (case-sensitive)
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
