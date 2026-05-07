package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Registry represents a skill source repository for auto-pull and team collaboration.
// URL is the full repo clone URL (no trailing slash).
type Registry struct {
	URL      string `json:"url" yaml:"url"`           // Full repo clone URL (e.g., "https://github.com/anthropics/skills")
	Name     string `json:"name" yaml:"name"`         // Human-readable name (e.g., "Anthropic Public Skills")
	Priority int    `json:"priority" yaml:"priority"` // Lower number = higher priority
}

// LocalConfig is the project-level config stored at .skillops/config.json.
// It is the source of truth for which tools and skills are active in a project.
// Version must be "2". Config v1 is NOT supported.
type LocalConfig struct {
	Version      string              `json:"version"`                  // always "2"
	Registries   []Registry          `json:"registries"`              // skill source registries
	Tools        map[string][]string `json:"tools"`                   // tool → []skill_identity (full-path format)
	SymlinkNames map[string]string   `json:"symlink_names,omitempty"` // identity → custom symlink name
}

// LocalConfigPath returns the absolute path to .skillops/config.json in cwd.
func LocalConfigPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		return filepath.Join(".skillops", "config.json")
	}
	return filepath.Join(cwd, ".skillops", "config.json")
}

// ReadLocalConfig reads and parses .skillops/config.json.
// Returns an error wrapping os.ErrNotExist if the file does not exist.
// Returns an error if the config version is not "2".
func ReadLocalConfig() (LocalConfig, error) {
	data, err := os.ReadFile(LocalConfigPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return LocalConfig{}, fmt.Errorf("local config not found: %w", os.ErrNotExist)
		}
		return LocalConfig{}, fmt.Errorf("failed to read local config: %w", err)
	}
	var cfg LocalConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return LocalConfig{}, fmt.Errorf("failed to parse local config: %w", err)
	}

	// Validate version — only v2 is supported
	if cfg.Version != "2" {
		if cfg.Version == "" || cfg.Version == "1" {
			return LocalConfig{}, fmt.Errorf("config version %q is not supported. This version requires config v2. Please run: skillops init", cfg.Version)
		}
		return LocalConfig{}, fmt.Errorf("unsupported config version %q: expected \"2\"", cfg.Version)
	}

	if cfg.Tools == nil {
		cfg.Tools = make(map[string][]string)
	}
	if cfg.Registries == nil {
		cfg.Registries = []Registry{}
	}
	return cfg, nil
}

// WriteLocalConfig writes cfg to .skillops/config.json, creating the
// .skillops/ directory if needed. Uses JSON with 2-space indentation.
// Version is always set to "2".
func WriteLocalConfig(cfg LocalConfig) error {
	cfg.Version = "2"
	if cfg.Tools == nil {
		cfg.Tools = make(map[string][]string)
	}
	if cfg.Registries == nil {
		cfg.Registries = []Registry{}
	}
	path := LocalConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create .skillops directory: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal local config: %w", err)
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

// GetActiveTools returns the list of tool names declared in local config.
func GetActiveTools() ([]string, error) {
	cfg, err := ReadLocalConfig()
	if err != nil {
		return nil, err
	}
	tools := make([]string, 0, len(cfg.Tools))
	for t := range cfg.Tools {
		tools = append(tools, t)
	}
	return tools, nil
}

// GetToolSkills returns the full "repo/skill" identities for a given tool.
func GetToolSkills(tool string) ([]string, error) {
	cfg, err := ReadLocalConfig()
	if err != nil {
		return nil, err
	}
	return cfg.Tools[tool], nil
}

// AddSkillToTool appends repoSkill ("repo/skill") to the tool's entry.
// No-ops if already present. Creates the tool entry if missing.
func AddSkillToTool(tool, repoSkill string) error {
	cfg, err := ReadLocalConfig()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		cfg = LocalConfig{Tools: make(map[string][]string)}
	}
	for _, s := range cfg.Tools[tool] {
		if s == repoSkill {
			return nil // already present, no-op
		}
	}
	cfg.Tools[tool] = append(cfg.Tools[tool], repoSkill)
	return WriteLocalConfig(cfg)
}

// RemoveSkillFromTool removes repoSkill from the tool's entry.
// No-ops if not present.
func RemoveSkillFromTool(tool, repoSkill string) error {
	cfg, err := ReadLocalConfig()
	if err != nil {
		return err
	}
	skills := cfg.Tools[tool]
	updated := skills[:0:0]
	for _, s := range skills {
		if s != repoSkill {
			updated = append(updated, s)
		}
	}
	if len(updated) == len(skills) {
		return nil // not present, no-op
	}
	cfg.Tools[tool] = updated
	return WriteLocalConfig(cfg)
}

// SetActiveTools replaces the tools map keys with the given list,
// preserving existing skill entries for tools that remain active.
// Tools removed from the list have their entries deleted.
func SetActiveTools(tools []string) error {
	cfg, err := ReadLocalConfig()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		cfg = LocalConfig{Tools: make(map[string][]string)}
	}
	next := make(map[string][]string, len(tools))
	for _, t := range tools {
		if existing, ok := cfg.Tools[t]; ok {
			next[t] = existing
		} else {
			next[t] = []string{}
		}
	}
	cfg.Tools = next
	return WriteLocalConfig(cfg)
}
