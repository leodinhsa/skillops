package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// SkillMetadataFile is the filename for per-skill metadata.
	SkillMetadataFile = ".so-skill-meta.json"
	// RepoMetadataFile is the filename for per-repo metadata (full pulls).
	RepoMetadataFile = ".so-repo-meta.json"
)

// SkillMetadata stores provenance information for a single skill.
// Saved as .so-skill-meta.json inside the skill directory.
type SkillMetadata struct {
	RepoURL    string    `json:"repo_url"`
	PathInRepo string    `json:"path_in_repo"`
	PulledAt   time.Time `json:"pulled_at"`
	CommitHash string    `json:"commit_hash"`
}

// SaveSkillMetadata writes skill metadata to <skillPath>/.so-skill-meta.json.
// Uses indented JSON for human readability.
func SaveSkillMetadata(skillPath string, meta SkillMetadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal skill metadata: %w", err)
	}
	metaPath := filepath.Join(skillPath, SkillMetadataFile)
	if err := os.WriteFile(metaPath, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("failed to write skill metadata: %w", err)
	}
	return nil
}

// LoadSkillMetadata reads and parses .so-skill-meta.json from the given skill path.
// Returns an error if the file does not exist or cannot be parsed.
func LoadSkillMetadata(skillPath string) (SkillMetadata, error) {
	metaPath := filepath.Join(skillPath, SkillMetadataFile)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return SkillMetadata{}, fmt.Errorf("failed to read skill metadata: %w", err)
	}
	var meta SkillMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return SkillMetadata{}, fmt.Errorf("failed to parse skill metadata: %w", err)
	}
	return meta, nil
}

// HasSkillMetadata checks whether .so-skill-meta.json exists in the given skill path.
func HasSkillMetadata(skillPath string) bool {
	metaPath := filepath.Join(skillPath, SkillMetadataFile)
	_, err := os.Stat(metaPath)
	return err == nil
}

// NewRepoMetadata stores provenance information for a full repository pull.
// Saved as .so-repo-meta.json at the repository root in the global store.
type NewRepoMetadata struct {
	RepoURL    string    `json:"repo_url"`
	PulledAt   time.Time `json:"pulled_at"`
	CommitHash string    `json:"commit_hash"`
}

// SaveRepoMeta writes repo metadata to <repoPath>/.so-repo-meta.json.
// Uses indented JSON for human readability.
func SaveRepoMeta(repoPath string, meta NewRepoMetadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal repo metadata: %w", err)
	}
	metaPath := filepath.Join(repoPath, RepoMetadataFile)
	if err := os.WriteFile(metaPath, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("failed to write repo metadata: %w", err)
	}
	return nil
}

// LoadRepoMeta reads and parses .so-repo-meta.json from the given repo path.
// Returns an error if the file does not exist or cannot be parsed.
func LoadRepoMeta(repoPath string) (NewRepoMetadata, error) {
	metaPath := filepath.Join(repoPath, RepoMetadataFile)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return NewRepoMetadata{}, fmt.Errorf("failed to read repo metadata: %w", err)
	}
	var meta NewRepoMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return NewRepoMetadata{}, fmt.Errorf("failed to parse repo metadata: %w", err)
	}
	return meta, nil
}
