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
	Agentics map[string]string `yaml:"agentics"`
}

var defaultAgentics = map[string]string{
	"claude":      ".claude/skills",
	"antigravity": ".agents/skills",
	"opencode":    ".opencode/skills",
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

	// Ensure config
	if _, err := os.Stat(ConfigPath); os.IsNotExist(err) {
		cfg := Config{
			Agentics: make(map[string]string),
		}
		for k, v := range defaultAgentics {
			cfg.Agentics[k] = v
		}
		if err := WriteConfig(cfg); err != nil {
			return fmt.Errorf("failed to write default config: %w", err)
		}
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
