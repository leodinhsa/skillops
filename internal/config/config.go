package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	AppName    = "skillops"
	ConfigName = "agentics.yaml"
)

var (
	HomeDir     string
	SkillOpsDir string
	ConfigDir   string
	SkillsDir   string
	ConfigPath  string // ~/.skillops/config/agentics.yaml
	Version     = "v1.0.0"
)

func init() {
	// Handle missing HOME environment variable
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		// Try to get current user's home directory
		userDir, err := os.UserHomeDir()
		if err != nil {
			// Fall back to using the current directory
			userDir, _ = os.Getwd()
			fmt.Fprintf(os.Stderr, "WARNING: HOME environment variable not set, using current directory: %s\n", userDir)
		}
		homeDir = userDir
	}
	HomeDir = homeDir
	SkillOpsDir = filepath.Join(HomeDir, "."+AppName)
	ConfigDir = filepath.Join(SkillOpsDir, "config")
	SkillsDir = filepath.Join(SkillOpsDir, "skills")
	ConfigPath = filepath.Join(ConfigDir, "agentics.yaml")
}

type Config struct {
	ConfigVersion int               `yaml:"config_version,omitempty"`
	Agentics      map[string]string `yaml:"agentics"`
}

// currentConfigVersion is bumped when the default agentics list changes
// in a breaking way (e.g. trimming from 35+ to 9 in v2).
const currentConfigVersion = 2

// legacyAgentics is the set of tool names that were in the pre-v2 default list
// and should be removed during migration (unless the user explicitly re-adds them).
var legacyAgentics = map[string]bool{
	"universal": true, "augment": true, "openclaw": true, "cline": true,
	"codebuddy": true, "codex": true, "command-code": true, "continue": true,
	"cortex": true, "crush": true, "droid": true, "junie": true,
	"iflow-cli": true, "kilo": true, "kiro-cli": true, "kode": true,
	"mcpjam": true, "mistral-vibe": true, "mux": true, "openhands": true,
	"pi": true, "qoder": true, "qwen-code": true, "roo": true,
	"trae": true, "zencoder": true, "neovate": true, "pochi": true,
	"adal": true, "claude": true, "universal s": true,
}

var defaultAgentics = map[string]string{
	"claude-code":    ".claude/skills",
	"cursor":         ".cursor/skills",
	"windsurf":       ".windsurf/skills",
	"kiro":           ".kiro/skills",
	"gemini-cli":     ".gemini/skills",
	"goose":          ".goose/skills",
	"github-copilot": ".github/skills",
	"opencode":       ".agents/skills",
	"antigravity":    ".agent/skills",
	// Commented out (not deleted) — re-enable by uncommenting:
	// "universal":    ".agents/skills",
	// "augment":      ".augment/skills",
	// "openclaw":     "skills",
	// "cline":        ".agents/skills",
	// "codebuddy":    ".codebuddy/skills",
	// "codex":        ".agents/skills",
	// "command-code": ".commandcode/skills",
	// "continue":     ".continue/skills",
	// "cortex":       ".cortex/skills",
	// "crush":        ".crush/skills",
	// "droid":        ".factory/skills",
	// "junie":        ".junie/skills",
	// "iflow-cli":    ".iflow/skills",
	// "kilo":         ".kilocode/skills",
	// "kiro-cli":     ".kiro/skills",
	// "kode":         ".kode/skills",
	// "mcpjam":       ".mcpjam/skills",
	// "mistral-vibe": ".vibe/skills",
	// "mux":          ".mux/skills",
	// "openhands":    ".openhands/skills",
	// "pi":           ".pi/skills",
	// "qoder":        ".qoder/skills",
	// "qwen-code":    ".qwen/skills",
	// "roo":          ".roo/skills",
	// "trae":         ".trae/skills",
	// "zencoder":     ".zencoder/skills",
	// "neovate":      ".neovate/skills",
	// "pochi":        ".pochi/skills",
	// "adal":         ".adal/skills",
}

func EnsureConfig() error {
	if err := os.MkdirAll(SkillOpsDir, 0755); err != nil {
		return fmt.Errorf("failed to create skillops directory: %w", err)
	}
	if err := os.MkdirAll(ConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	if err := os.MkdirAll(SkillsDir, 0755); err != nil {
		return fmt.Errorf("failed to create skills directory: %w", err)
	}

	cfg, err := ReadConfig()
	if err != nil {
		// File doesn't exist — start fresh at current version
		cfg = Config{
			ConfigVersion: currentConfigVersion,
			Agentics:      make(map[string]string),
		}
	}

	changed := false

	// Migration: if config is from before v2, prune legacy default entries
	if cfg.ConfigVersion < currentConfigVersion {
		for name := range legacyAgentics {
			if _, ok := cfg.Agentics[name]; ok {
				delete(cfg.Agentics, name)
				changed = true
			}
		}
		// Also fix any default entries that had wrong paths in v1
		for k, v := range defaultAgentics {
			cfg.Agentics[k] = v
		}
		cfg.ConfigVersion = currentConfigVersion
		changed = true
	}

	// Add any missing default entries
	for k, v := range defaultAgentics {
		if _, ok := cfg.Agentics[k]; !ok {
			cfg.Agentics[k] = v
			changed = true
		}
	}

	if changed {
		if err := WriteConfig(cfg); err != nil {
			return fmt.Errorf("failed to update config with new defaults: %w", err)
		}
	}

	if err := ensureSettings(); err != nil {
		return fmt.Errorf("failed to ensure settings: %w", err)
	}

	return nil
}

func ReadConfig() (Config, error) {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("failed to parse config: %w", err)
	}
	if cfg.Agentics == nil {
		cfg.Agentics = make(map[string]string)
	}
	return cfg, nil
}

func WriteConfig(cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(ConfigPath, data, 0600)
}

func GetAgentics() (map[string]string, error) {
	cfg, err := ReadConfig()
	if err != nil {
		return nil, err
	}
	return cfg.Agentics, nil
}

func GetAgenticPath(name string) (string, error) {
	cfg, err := ReadConfig()
	if err != nil {
		return "", err
	}
	path, ok := cfg.Agentics[name]
	if !ok {
		return "", fmt.Errorf("unknown agentic: %s", name)
	}
	return path, nil
}

func AddAgentic(name, path string) error {
	cfg, err := ReadConfig()
	if err != nil {
		cfg = Config{Agentics: make(map[string]string)}
	}
	cfg.Agentics[name] = path
	return WriteConfig(cfg)
}

func RemoveAgentic(name string) error {
	cfg, err := ReadConfig()
	if err != nil {
		return err
	}
	if _, ok := cfg.Agentics[name]; !ok {
		return fmt.Errorf("agentic '%s' not found", name)
	}
	delete(cfg.Agentics, name)
	return WriteConfig(cfg)
}

func UpdateAgentic(name, newPath string) error {
	cfg, err := ReadConfig()
	if err != nil {
		return err
	}
	if _, ok := cfg.Agentics[name]; !ok {
		return fmt.Errorf("agentic '%s' not found", name)
	}
	cfg.Agentics[name] = newPath
	return WriteConfig(cfg)
}
