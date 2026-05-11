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

// setupGlobalConfig sets up a temporary agentics.yaml with the given tool mapping.
// It overrides config.ConfigPath and config.ConfigDir for test isolation.
func setupGlobalConfig(t *testing.T, agentics map[string]string) {
	t.Helper()
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	origConfigDir := config.ConfigDir
	origConfigPath := config.ConfigPath
	config.ConfigDir = configDir
	config.ConfigPath = filepath.Join(configDir, "agentics.yaml")
	t.Cleanup(func() {
		config.ConfigDir = origConfigDir
		config.ConfigPath = origConfigPath
	})

	// Write agentics.yaml
	content := "config_version: 2\nagentics:\n"
	for name, path := range agentics {
		content += "  " + name + ": " + path + "\n"
	}
	if err := os.WriteFile(config.ConfigPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// setupGlobalSkill creates a fake skill in the global store with a SKILL.md file.
// Returns the global skill path.
func setupGlobalSkill(t *testing.T, globalStoreDir, identity string) string {
	t.Helper()
	skillPath := filepath.Join(globalStoreDir, filepath.FromSlash(identity))
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte("# skill"), 0644); err != nil {
		t.Fatal(err)
	}
	return skillPath
}

func TestCreateSkillSymlink_DefaultShortName(t *testing.T) {
	// Set up global store
	globalStore := t.TempDir()
	overrideSkillsDir(t, globalStore)

	// Set up project directory as cwd
	projectDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	// Set up global config with tool path
	setupGlobalConfig(t, map[string]string{"kiro": ".kiro/skills"})

	// Create a skill in the global store
	identity := "github.com/anthropics/skills/skills/logger"
	setupGlobalSkill(t, globalStore, identity)

	// Create symlink with default short name
	localCfg := config.LocalConfig{
		Version: "2",
		Tools:   map[string][]string{"kiro": {identity}},
	}

	wasCreated, err := CreateSkillSymlink(identity, "kiro", localCfg)
	if err != nil {
		t.Fatalf("CreateSkillSymlink error: %v", err)
	}
	if !wasCreated {
		t.Error("expected wasCreated=true")
	}

	// Verify symlink exists with short name "logger"
	symlinkPath := filepath.Join(projectDir, ".kiro", "skills", "logger")
	info, err := os.Lstat(symlinkPath)
	if err != nil {
		t.Fatalf("symlink not created: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink, got regular file/dir")
	}

	// Verify symlink target
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatal(err)
	}
	expectedTarget := filepath.Join(globalStore, "github.com", "anthropics", "skills", "skills", "logger")
	if target != expectedTarget {
		t.Errorf("symlink target = %q, want %q", target, expectedTarget)
	}
}

func TestCreateSkillSymlink_CustomName(t *testing.T) {
	// Set up global store
	globalStore := t.TempDir()
	overrideSkillsDir(t, globalStore)

	// Set up project directory as cwd
	projectDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	// Set up global config
	setupGlobalConfig(t, map[string]string{"kiro": ".kiro/skills"})

	// Create a skill in the global store
	identity := "github.com/company-a/utils/tools/logger"
	setupGlobalSkill(t, globalStore, identity)

	// Create symlink with custom name from SymlinkNames
	localCfg := config.LocalConfig{
		Version:      "2",
		Tools:        map[string][]string{"kiro": {identity}},
		SymlinkNames: map[string]string{identity: "logger-utils"},
	}

	wasCreated, err := CreateSkillSymlink(identity, "kiro", localCfg)
	if err != nil {
		t.Fatalf("CreateSkillSymlink error: %v", err)
	}
	if !wasCreated {
		t.Error("expected wasCreated=true")
	}

	// Verify symlink exists with custom name "logger-utils"
	symlinkPath := filepath.Join(projectDir, ".kiro", "skills", "logger-utils")
	info, err := os.Lstat(symlinkPath)
	if err != nil {
		t.Fatalf("symlink not created at custom name path: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected symlink, got regular file/dir")
	}

	// Verify the default name was NOT used
	defaultPath := filepath.Join(projectDir, ".kiro", "skills", "logger")
	if _, err := os.Lstat(defaultPath); !os.IsNotExist(err) {
		t.Error("symlink should NOT exist at default short name path")
	}
}

func TestCreateSkillSymlink_Idempotent(t *testing.T) {
	// Set up global store
	globalStore := t.TempDir()
	overrideSkillsDir(t, globalStore)

	// Set up project directory as cwd
	projectDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	// Set up global config
	setupGlobalConfig(t, map[string]string{"kiro": ".kiro/skills"})

	// Create a skill in the global store
	identity := "github.com/anthropics/skills/skills/logger"
	setupGlobalSkill(t, globalStore, identity)

	localCfg := config.LocalConfig{
		Version: "2",
		Tools:   map[string][]string{"kiro": {identity}},
	}

	// First call: should create
	wasCreated, err := CreateSkillSymlink(identity, "kiro", localCfg)
	if err != nil {
		t.Fatalf("first CreateSkillSymlink error: %v", err)
	}
	if !wasCreated {
		t.Error("first call: expected wasCreated=true")
	}

	// Second call: should be idempotent (already linked correctly)
	wasCreated, err = CreateSkillSymlink(identity, "kiro", localCfg)
	if err != nil {
		t.Fatalf("second CreateSkillSymlink error: %v", err)
	}
	if wasCreated {
		t.Error("second call: expected wasCreated=false (idempotent)")
	}
}

func TestCreateSkillSymlink_Conflict(t *testing.T) {
	// Set up global store
	globalStore := t.TempDir()
	overrideSkillsDir(t, globalStore)

	// Set up project directory as cwd
	projectDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	// Set up global config
	setupGlobalConfig(t, map[string]string{"kiro": ".kiro/skills"})

	// Create two skills with the same short name in the global store
	identity1 := "github.com/company-a/utils/tools/logger"
	identity2 := "github.com/company-b/helpers/services/logger"
	setupGlobalSkill(t, globalStore, identity1)
	setupGlobalSkill(t, globalStore, identity2)

	localCfg := config.LocalConfig{
		Version: "2",
		Tools:   map[string][]string{"kiro": {identity1, identity2}},
	}

	// Create first symlink
	wasCreated, err := CreateSkillSymlink(identity1, "kiro", localCfg)
	if err != nil {
		t.Fatalf("first CreateSkillSymlink error: %v", err)
	}
	if !wasCreated {
		t.Error("first call: expected wasCreated=true")
	}

	// Try to create second symlink with same short name — should conflict
	wasCreated, err = CreateSkillSymlink(identity2, "kiro", localCfg)
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	if wasCreated {
		t.Error("expected wasCreated=false on conflict")
	}

	// Error should mention "symlink conflict"
	if !contains(err.Error(), "symlink conflict") {
		t.Errorf("error should mention 'symlink conflict', got: %v", err)
	}
}

func TestCreateSkillSymlink_GlobalPathNotFound(t *testing.T) {
	// Set up global store (empty — no skills)
	globalStore := t.TempDir()
	overrideSkillsDir(t, globalStore)

	// Set up project directory as cwd
	projectDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	// Set up global config
	setupGlobalConfig(t, map[string]string{"kiro": ".kiro/skills"})

	// Try to create symlink for a skill that doesn't exist in global store
	identity := "github.com/anthropics/skills/skills/nonexistent"
	localCfg := config.LocalConfig{
		Version: "2",
		Tools:   map[string][]string{"kiro": {identity}},
	}

	wasCreated, err := CreateSkillSymlink(identity, "kiro", localCfg)
	if err == nil {
		t.Fatal("expected error for missing global skill, got nil")
	}
	if wasCreated {
		t.Error("expected wasCreated=false when skill not found")
	}

	// Error should mention "skill not found in global store"
	if !contains(err.Error(), "skill not found in global store") {
		t.Errorf("error should mention 'skill not found in global store', got: %v", err)
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
