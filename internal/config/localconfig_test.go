package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
		Version: "2",
		Registries: []Registry{
			{URL: "https://github.com/anthropics/skills", Name: "Anthropic Skills", Priority: 1},
		},
		Tools: map[string][]string{
			"cursor": {"github.com/anthropics/skills/skills/logger", "github.com/anthropics/skills/skills/auth"},
			"kiro":   {"github.com/anthropics/skills/skills/logger"},
		},
	}

	if err := WriteLocalConfig(cfg); err != nil {
		t.Fatalf("WriteLocalConfig error: %v", err)
	}

	got, err := ReadLocalConfig()
	if err != nil {
		t.Fatalf("ReadLocalConfig error: %v", err)
	}

	if got.Version != "2" {
		t.Errorf("Version = %q, want %q", got.Version, "2")
	}
	if len(got.Tools["cursor"]) != 2 {
		t.Errorf("cursor skills = %d, want 2", len(got.Tools["cursor"]))
	}
	if len(got.Tools["kiro"]) != 1 {
		t.Errorf("kiro skills = %d, want 1", len(got.Tools["kiro"]))
	}
	if len(got.Registries) != 1 {
		t.Errorf("registries = %d, want 1", len(got.Registries))
	}
	if got.Registries[0].URL != "https://github.com/anthropics/skills" {
		t.Errorf("registry URL = %q, want %q", got.Registries[0].URL, "https://github.com/anthropics/skills")
	}
}

func TestWriteAndReadLocalConfig_WithSymlinkNames(t *testing.T) {
	setupLocalConfig(t)

	cfg := LocalConfig{
		Version: "2",
		Registries: []Registry{
			{URL: "https://github.com/anthropics/skills", Name: "Anthropic Skills", Priority: 1},
		},
		Tools: map[string][]string{
			"kiro": {
				"github.com/company-a/utils/tools/logger",
				"github.com/company-b/helpers/services/logger",
			},
		},
		SymlinkNames: map[string]string{
			"github.com/company-a/utils/tools/logger":      "logger-utils",
			"github.com/company-b/helpers/services/logger": "logger-services",
		},
	}

	if err := WriteLocalConfig(cfg); err != nil {
		t.Fatalf("WriteLocalConfig error: %v", err)
	}

	got, err := ReadLocalConfig()
	if err != nil {
		t.Fatalf("ReadLocalConfig error: %v", err)
	}

	if len(got.SymlinkNames) != 2 {
		t.Errorf("symlink_names = %d, want 2", len(got.SymlinkNames))
	}
	if got.SymlinkNames["github.com/company-a/utils/tools/logger"] != "logger-utils" {
		t.Errorf("symlink name = %q, want %q", got.SymlinkNames["github.com/company-a/utils/tools/logger"], "logger-utils")
	}
}

func TestWriteLocalConfig_SymlinkNamesOmittedWhenEmpty(t *testing.T) {
	setupLocalConfig(t)

	cfg := LocalConfig{
		Version:    "2",
		Registries: []Registry{},
		Tools:      map[string][]string{"kiro": {}},
		// SymlinkNames intentionally nil/empty
	}

	if err := WriteLocalConfig(cfg); err != nil {
		t.Fatalf("WriteLocalConfig error: %v", err)
	}

	// Read raw JSON to verify symlink_names is omitted
	data, err := os.ReadFile(LocalConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "symlink_names") {
		t.Error("expected symlink_names to be omitted from JSON when empty")
	}
}

func TestReadLocalConfig_NotFound(t *testing.T) {
	setupLocalConfig(t)

	_, err := ReadLocalConfig()
	if err == nil {
		t.Error("expected error for missing config, got nil")
	}
}

func TestReadLocalConfig_V1ConfigFails(t *testing.T) {
	setupLocalConfig(t)

	// Write a v1 config manually
	v1Config := map[string]interface{}{
		"version": "1",
		"tools": map[string][]string{
			"cursor": {"repo/logger"},
		},
	}
	data, _ := json.MarshalIndent(v1Config, "", "  ")
	os.MkdirAll(".skillops", 0755)
	os.WriteFile(filepath.Join(".skillops", "config.json"), data, 0644)

	_, err := ReadLocalConfig()
	if err == nil {
		t.Fatal("expected error for v1 config, got nil")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("error should mention 'not supported', got: %v", err)
	}
	if !strings.Contains(err.Error(), "skillops init") {
		t.Errorf("error should suggest 'skillops init', got: %v", err)
	}
}

func TestReadLocalConfig_NoVersionFails(t *testing.T) {
	setupLocalConfig(t)

	// Write a config with no version field
	noVersionConfig := map[string]interface{}{
		"tools": map[string][]string{
			"cursor": {"repo/logger"},
		},
	}
	data, _ := json.MarshalIndent(noVersionConfig, "", "  ")
	os.MkdirAll(".skillops", 0755)
	os.WriteFile(filepath.Join(".skillops", "config.json"), data, 0644)

	_, err := ReadLocalConfig()
	if err == nil {
		t.Fatal("expected error for config with no version, got nil")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("error should mention 'not supported', got: %v", err)
	}
}

func TestReadLocalConfig_UnsupportedVersionFails(t *testing.T) {
	setupLocalConfig(t)

	// Write a config with unsupported version
	badConfig := map[string]interface{}{
		"version": "99",
		"tools":   map[string][]string{},
	}
	data, _ := json.MarshalIndent(badConfig, "", "  ")
	os.MkdirAll(".skillops", 0755)
	os.WriteFile(filepath.Join(".skillops", "config.json"), data, 0644)

	_, err := ReadLocalConfig()
	if err == nil {
		t.Fatal("expected error for unsupported version, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported config version") {
		t.Errorf("error should mention 'unsupported config version', got: %v", err)
	}
}

func TestAddSkillToTool(t *testing.T) {
	setupLocalConfig(t)

	// Bootstrap with empty v2 config
	if err := WriteLocalConfig(LocalConfig{Tools: map[string][]string{"cursor": {}}}); err != nil {
		t.Fatal(err)
	}

	if err := AddSkillToTool("cursor", "github.com/anthropics/skills/skills/logger"); err != nil {
		t.Fatalf("AddSkillToTool error: %v", err)
	}

	cfg, _ := ReadLocalConfig()
	if len(cfg.Tools["cursor"]) != 1 || cfg.Tools["cursor"][0] != "github.com/anthropics/skills/skills/logger" {
		t.Errorf("unexpected tools: %v", cfg.Tools["cursor"])
	}
}

func TestAddSkillToTool_NoDuplicate(t *testing.T) {
	setupLocalConfig(t)

	if err := WriteLocalConfig(LocalConfig{Tools: map[string][]string{"cursor": {"github.com/anthropics/skills/skills/logger"}}}); err != nil {
		t.Fatal(err)
	}

	// Add same skill again
	if err := AddSkillToTool("cursor", "github.com/anthropics/skills/skills/logger"); err != nil {
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
		"cursor": {"github.com/anthropics/skills/skills/logger", "github.com/anthropics/skills/skills/auth"},
	}}); err != nil {
		t.Fatal(err)
	}

	if err := RemoveSkillFromTool("cursor", "github.com/anthropics/skills/skills/logger"); err != nil {
		t.Fatalf("RemoveSkillFromTool error: %v", err)
	}

	cfg, _ := ReadLocalConfig()
	if len(cfg.Tools["cursor"]) != 1 || cfg.Tools["cursor"][0] != "github.com/anthropics/skills/skills/auth" {
		t.Errorf("unexpected tools after remove: %v", cfg.Tools["cursor"])
	}
}

func TestRemoveSkillFromTool_NotPresent(t *testing.T) {
	setupLocalConfig(t)

	if err := WriteLocalConfig(LocalConfig{Tools: map[string][]string{"cursor": {"github.com/anthropics/skills/skills/auth"}}}); err != nil {
		t.Fatal(err)
	}

	// Remove something not there — should be no-op, no error
	if err := RemoveSkillFromTool("cursor", "github.com/anthropics/skills/skills/nonexistent"); err != nil {
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
		"cursor":    {"github.com/anthropics/skills/skills/logger"},
		"windsurf":  {"github.com/anthropics/skills/skills/auth"},
		"to-remove": {"github.com/anthropics/skills/skills/old"},
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
	if len(cfg.Tools["cursor"]) != 1 || cfg.Tools["cursor"][0] != "github.com/anthropics/skills/skills/logger" {
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

func TestWriteLocalConfig_AlwaysSetsVersion2(t *testing.T) {
	setupLocalConfig(t)

	// Write with wrong version — should be overridden to "2"
	cfg := LocalConfig{Version: "99", Tools: map[string][]string{}}
	if err := WriteLocalConfig(cfg); err != nil {
		t.Fatal(err)
	}

	got, _ := ReadLocalConfig()
	if got.Version != "2" {
		t.Errorf("Version = %q, want %q", got.Version, "2")
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

func TestWriteLocalConfig_RegistriesPreserved(t *testing.T) {
	setupLocalConfig(t)

	cfg := LocalConfig{
		Registries: []Registry{
			{URL: "https://github.com/anthropics/skills", Name: "Anthropic Skills", Priority: 1},
			{URL: "git@github.com:company/private-skills", Name: "Company Private", Priority: 2},
		},
		Tools: map[string][]string{
			"kiro": {"github.com/anthropics/skills/skills/logger"},
		},
	}

	if err := WriteLocalConfig(cfg); err != nil {
		t.Fatalf("WriteLocalConfig error: %v", err)
	}

	got, err := ReadLocalConfig()
	if err != nil {
		t.Fatalf("ReadLocalConfig error: %v", err)
	}

	if len(got.Registries) != 2 {
		t.Fatalf("registries = %d, want 2", len(got.Registries))
	}
	if got.Registries[0].Priority != 1 || got.Registries[1].Priority != 2 {
		t.Errorf("registry priorities not preserved: %v", got.Registries)
	}
}
