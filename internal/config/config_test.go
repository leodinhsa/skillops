package config

import (
	"os"
	"path/filepath"
	"testing"
)

// setupConfigDir redirects all config paths to a temp dir for isolation.
func setupConfigDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()

	origHome := HomeDir
	origSkillOps := SkillOpsDir
	origConfigDir := ConfigDir
	origSkillsDir := SkillsDir
	origConfigPath := ConfigPath

	HomeDir = tmp
	SkillOpsDir = filepath.Join(tmp, ".skillops")
	ConfigDir = filepath.Join(tmp, ".skillops", "config")
	SkillsDir = filepath.Join(tmp, ".skillops", "skills")
	ConfigPath = filepath.Join(tmp, ".skillops", "config", "agentics.yaml")

	t.Cleanup(func() {
		HomeDir = origHome
		SkillOpsDir = origSkillOps
		ConfigDir = origConfigDir
		SkillsDir = origSkillsDir
		ConfigPath = origConfigPath
	})
	return tmp
}

func TestEnsureConfig_CreatesDefaults(t *testing.T) {
	setupConfigDir(t)

	if err := EnsureConfig(); err != nil {
		t.Fatalf("EnsureConfig() error: %v", err)
	}

	cfg, err := ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig() error: %v", err)
	}

	// Should have default agentics
	for k, v := range defaultAgentics {
		got, ok := cfg.Agentics[k]
		if !ok {
			t.Errorf("missing default agentic %q", k)
			continue
		}
		if got != v {
			t.Errorf("agentic %q = %q, want %q", k, got, v)
		}
	}

	if cfg.ConfigVersion != currentConfigVersion {
		t.Errorf("ConfigVersion = %d, want %d", cfg.ConfigVersion, currentConfigVersion)
	}
}

func TestEnsureConfig_Idempotent(t *testing.T) {
	setupConfigDir(t)

	if err := EnsureConfig(); err != nil {
		t.Fatal(err)
	}
	if err := EnsureConfig(); err != nil {
		t.Fatalf("second EnsureConfig() error: %v", err)
	}

	cfg, _ := ReadConfig()
	// Should not duplicate entries
	count := 0
	for range cfg.Agentics {
		count++
	}
	if count != len(defaultAgentics) {
		t.Errorf("expected %d agentics, got %d", len(defaultAgentics), count)
	}
}

func TestEnsureConfig_PreservesUserEntries(t *testing.T) {
	setupConfigDir(t)

	if err := EnsureConfig(); err != nil {
		t.Fatal(err)
	}

	// User adds a custom agentic
	if err := AddAgentic("my-custom-ide", ".custom/skills"); err != nil {
		t.Fatal(err)
	}

	// Run EnsureConfig again
	if err := EnsureConfig(); err != nil {
		t.Fatal(err)
	}

	cfg, _ := ReadConfig()
	if _, ok := cfg.Agentics["my-custom-ide"]; !ok {
		t.Error("user-added agentic was removed by EnsureConfig")
	}
}

func TestEnsureConfig_MigratesLegacy(t *testing.T) {
	setupConfigDir(t)

	// Write a v1 config with legacy entries
	if err := os.MkdirAll(ConfigDir, 0755); err != nil {
		t.Fatal(err)
	}
	v1Config := `config_version: 1
agentics:
  roo: .roo/skills
  cursor: .cursor/skills
`
	if err := os.WriteFile(ConfigPath, []byte(v1Config), 0600); err != nil {
		t.Fatal(err)
	}

	if err := EnsureConfig(); err != nil {
		t.Fatalf("EnsureConfig() error: %v", err)
	}

	cfg, _ := ReadConfig()

	// Legacy entry "roo" should be removed
	if _, ok := cfg.Agentics["roo"]; ok {
		t.Error("legacy agentic 'roo' should have been removed during migration")
	}

	// Default entries should be present
	if _, ok := cfg.Agentics["cursor"]; !ok {
		t.Error("default agentic 'cursor' should be present after migration")
	}

	if cfg.ConfigVersion != currentConfigVersion {
		t.Errorf("ConfigVersion = %d, want %d", cfg.ConfigVersion, currentConfigVersion)
	}
}

func TestAddAndRemoveAgentic(t *testing.T) {
	setupConfigDir(t)
	if err := EnsureConfig(); err != nil {
		t.Fatal(err)
	}

	if err := AddAgentic("test-ide", ".test/skills"); err != nil {
		t.Fatalf("AddAgentic error: %v", err)
	}

	path, err := GetAgenticPath("test-ide")
	if err != nil {
		t.Fatalf("GetAgenticPath error: %v", err)
	}
	if path != ".test/skills" {
		t.Errorf("path = %q, want %q", path, ".test/skills")
	}

	if err := RemoveAgentic("test-ide"); err != nil {
		t.Fatalf("RemoveAgentic error: %v", err)
	}

	_, err = GetAgenticPath("test-ide")
	if err == nil {
		t.Error("expected error after removal, got nil")
	}
}

func TestRemoveAgentic_NotFound(t *testing.T) {
	setupConfigDir(t)
	if err := EnsureConfig(); err != nil {
		t.Fatal(err)
	}

	err := RemoveAgentic("nonexistent")
	if err == nil {
		t.Error("expected error removing nonexistent agentic")
	}
}

func TestUpdateAgentic(t *testing.T) {
	setupConfigDir(t)
	if err := EnsureConfig(); err != nil {
		t.Fatal(err)
	}

	if err := AddAgentic("my-ide", ".old/skills"); err != nil {
		t.Fatal(err)
	}
	if err := UpdateAgentic("my-ide", ".new/skills"); err != nil {
		t.Fatalf("UpdateAgentic error: %v", err)
	}

	path, _ := GetAgenticPath("my-ide")
	if path != ".new/skills" {
		t.Errorf("path = %q, want %q", path, ".new/skills")
	}
}

func TestGetAgenticPath_Unknown(t *testing.T) {
	setupConfigDir(t)
	if err := EnsureConfig(); err != nil {
		t.Fatal(err)
	}

	_, err := GetAgenticPath("unknown-ide")
	if err == nil {
		t.Error("expected error for unknown agentic")
	}
}

func TestWriteAndReadConfig_RoundTrip(t *testing.T) {
	setupConfigDir(t)
	if err := os.MkdirAll(ConfigDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		ConfigVersion: 2,
		Agentics: map[string]string{
			"cursor": ".cursor/skills",
			"kiro":   ".kiro/skills",
		},
	}

	if err := WriteConfig(cfg); err != nil {
		t.Fatalf("WriteConfig error: %v", err)
	}

	got, err := ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig error: %v", err)
	}

	if got.ConfigVersion != cfg.ConfigVersion {
		t.Errorf("ConfigVersion = %d, want %d", got.ConfigVersion, cfg.ConfigVersion)
	}
	for k, v := range cfg.Agentics {
		if got.Agentics[k] != v {
			t.Errorf("Agentics[%q] = %q, want %q", k, got.Agentics[k], v)
		}
	}
}
