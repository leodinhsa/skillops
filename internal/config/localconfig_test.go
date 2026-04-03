package config

import (
	"os"
	"path/filepath"
	"testing"
)

// setupLocalConfig creates a temp project dir and redirects LocalConfigPath to it.
func setupLocalConfig(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()

	// Change working directory to tmp so LocalConfigPath() resolves correctly
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	return tmp
}

func TestWriteAndReadLocalConfig(t *testing.T) {
	setupLocalConfig(t)

	cfg := LocalConfig{
		Version: "1",
		Tools: map[string][]string{
			"cursor": {"repo-a/logger", "repo-a/auth"},
			"kiro":   {"repo-a/logger"},
		},
	}

	if err := WriteLocalConfig(cfg); err != nil {
		t.Fatalf("WriteLocalConfig error: %v", err)
	}

	got, err := ReadLocalConfig()
	if err != nil {
		t.Fatalf("ReadLocalConfig error: %v", err)
	}

	if got.Version != "1" {
		t.Errorf("Version = %q, want %q", got.Version, "1")
	}
	if len(got.Tools["cursor"]) != 2 {
		t.Errorf("cursor skills = %d, want 2", len(got.Tools["cursor"]))
	}
	if len(got.Tools["kiro"]) != 1 {
		t.Errorf("kiro skills = %d, want 1", len(got.Tools["kiro"]))
	}
}

func TestReadLocalConfig_NotFound(t *testing.T) {
	setupLocalConfig(t)

	_, err := ReadLocalConfig()
	if err == nil {
		t.Error("expected error for missing config, got nil")
	}
}

func TestAddSkillToTool(t *testing.T) {
	setupLocalConfig(t)

	// Bootstrap with empty config
	if err := WriteLocalConfig(LocalConfig{Tools: map[string][]string{"cursor": {}}}); err != nil {
		t.Fatal(err)
	}

	if err := AddSkillToTool("cursor", "repo/logger"); err != nil {
		t.Fatalf("AddSkillToTool error: %v", err)
	}

	cfg, _ := ReadLocalConfig()
	if len(cfg.Tools["cursor"]) != 1 || cfg.Tools["cursor"][0] != "repo/logger" {
		t.Errorf("unexpected tools: %v", cfg.Tools["cursor"])
	}
}

func TestAddSkillToTool_NoDuplicate(t *testing.T) {
	setupLocalConfig(t)

	if err := WriteLocalConfig(LocalConfig{Tools: map[string][]string{"cursor": {"repo/logger"}}}); err != nil {
		t.Fatal(err)
	}

	// Add same skill again
	if err := AddSkillToTool("cursor", "repo/logger"); err != nil {
		t.Fatalf("AddSkillToTool error: %v", err)
	}

	cfg, _ := ReadLocalConfig()
	if len(cfg.Tools["cursor"]) != 1 {
		t.Errorf("expected 1 skill (no duplicate), got %d", len(cfg.Tools["cursor"]))
	}
}

func TestRemoveSkillFromTool(t *testing.T) {
	setupLocalConfig(t)

	if err := WriteLocalConfig(LocalConfig{Tools: map[string][]string{
		"cursor": {"repo/logger", "repo/auth"},
	}}); err != nil {
		t.Fatal(err)
	}

	if err := RemoveSkillFromTool("cursor", "repo/logger"); err != nil {
		t.Fatalf("RemoveSkillFromTool error: %v", err)
	}

	cfg, _ := ReadLocalConfig()
	if len(cfg.Tools["cursor"]) != 1 || cfg.Tools["cursor"][0] != "repo/auth" {
		t.Errorf("unexpected tools after remove: %v", cfg.Tools["cursor"])
	}
}

func TestRemoveSkillFromTool_NotPresent(t *testing.T) {
	setupLocalConfig(t)

	if err := WriteLocalConfig(LocalConfig{Tools: map[string][]string{"cursor": {"repo/auth"}}}); err != nil {
		t.Fatal(err)
	}

	// Remove something not there — should be no-op, no error
	if err := RemoveSkillFromTool("cursor", "repo/nonexistent"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg, _ := ReadLocalConfig()
	if len(cfg.Tools["cursor"]) != 1 {
		t.Errorf("expected 1 skill unchanged, got %d", len(cfg.Tools["cursor"]))
	}
}

func TestSetActiveTools(t *testing.T) {
	setupLocalConfig(t)

	if err := WriteLocalConfig(LocalConfig{Tools: map[string][]string{
		"cursor":    {"repo/logger"},
		"windsurf":  {"repo/auth"},
		"to-remove": {"repo/old"},
	}}); err != nil {
		t.Fatal(err)
	}

	if err := SetActiveTools([]string{"cursor", "windsurf"}); err != nil {
		t.Fatalf("SetActiveTools error: %v", err)
	}

	cfg, _ := ReadLocalConfig()

	if _, ok := cfg.Tools["to-remove"]; ok {
		t.Error("removed tool should not be in config")
	}
	// Existing skills preserved
	if len(cfg.Tools["cursor"]) != 1 || cfg.Tools["cursor"][0] != "repo/logger" {
		t.Errorf("cursor skills not preserved: %v", cfg.Tools["cursor"])
	}
}

func TestGetActiveTools(t *testing.T) {
	setupLocalConfig(t)

	if err := WriteLocalConfig(LocalConfig{Tools: map[string][]string{
		"cursor":   {},
		"windsurf": {},
	}}); err != nil {
		t.Fatal(err)
	}

	tools, err := GetActiveTools()
	if err != nil {
		t.Fatalf("GetActiveTools error: %v", err)
	}
	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d: %v", len(tools), tools)
	}
}

func TestWriteLocalConfig_AlwaysSetsVersion(t *testing.T) {
	setupLocalConfig(t)

	// Write with wrong version
	cfg := LocalConfig{Version: "99", Tools: map[string][]string{}}
	if err := WriteLocalConfig(cfg); err != nil {
		t.Fatal(err)
	}

	got, _ := ReadLocalConfig()
	if got.Version != "1" {
		t.Errorf("Version = %q, want %q", got.Version, "1")
	}
}

func TestLocalConfigPath(t *testing.T) {
	setupLocalConfig(t)

	path := LocalConfigPath()
	if filepath.Base(filepath.Dir(path)) != ".skillops" {
		t.Errorf("unexpected config path: %s", path)
	}
	if filepath.Base(path) != "config.json" {
		t.Errorf("unexpected config filename: %s", filepath.Base(path))
	}
}
