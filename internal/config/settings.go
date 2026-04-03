package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Registry represents a remote skill registry.
type Registry struct {
	URL  string `yaml:"url"`
	Name string `yaml:"name,omitempty"`
}

// Settings holds global user settings (registries, etc.).
type Settings struct {
	Registries []Registry `yaml:"registries"`
}

// SettingsPath returns the path to ~/.skillops/config/settings.yaml.
func SettingsPath() string {
	return filepath.Join(ConfigDir, "settings.yaml")
}

// ReadSettings reads ~/.skillops/config/settings.yaml.
// If the file is absent, returns empty Settings{} with no error.
// If the file is malformed, logs a warning to stderr and returns empty Settings{}.
func ReadSettings() (Settings, error) {
	data, err := os.ReadFile(SettingsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return Settings{}, nil
		}
		return Settings{}, fmt.Errorf("failed to read settings: %w", err)
	}
	var s Settings
	if err := yaml.Unmarshal(data, &s); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: settings.yaml is malformed, ignoring registries: %v\n", err)
		return Settings{}, nil
	}
	return s, nil
}

// WriteSettings writes s to ~/.skillops/config/settings.yaml.
func WriteSettings(s Settings) error {
	if err := os.MkdirAll(ConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}
	return os.WriteFile(SettingsPath(), data, 0600)
}

// ensureSettings writes settings.yaml with an empty registries list if the file does not exist.
func ensureSettings() error {
	if _, err := os.Stat(SettingsPath()); os.IsNotExist(err) {
		return WriteSettings(Settings{Registries: []Registry{}})
	}
	return nil
}
