package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoadSkillMetadata(t *testing.T) {
	tmp := t.TempDir()

	meta := SkillMetadata{
		RepoURL:    "https://github.com/anthropics/skills",
		PathInRepo: "skills/logger",
		PulledAt:   time.Date(2026, 5, 6, 10, 30, 0, 0, time.UTC),
		CommitHash: "abc123def456",
	}

	if err := SaveSkillMetadata(tmp, meta); err != nil {
		t.Fatalf("SaveSkillMetadata error: %v", err)
	}

	got, err := LoadSkillMetadata(tmp)
	if err != nil {
		t.Fatalf("LoadSkillMetadata error: %v", err)
	}

	if got.RepoURL != meta.RepoURL {
		t.Errorf("RepoURL = %q, want %q", got.RepoURL, meta.RepoURL)
	}
	if got.PathInRepo != meta.PathInRepo {
		t.Errorf("PathInRepo = %q, want %q", got.PathInRepo, meta.PathInRepo)
	}
	if !got.PulledAt.Equal(meta.PulledAt) {
		t.Errorf("PulledAt = %v, want %v", got.PulledAt, meta.PulledAt)
	}
	if got.CommitHash != meta.CommitHash {
		t.Errorf("CommitHash = %q, want %q", got.CommitHash, meta.CommitHash)
	}
}

func TestLoadSkillMetadata_Missing(t *testing.T) {
	tmp := t.TempDir()

	_, err := LoadSkillMetadata(tmp)
	if err == nil {
		t.Error("expected error for missing metadata, got nil")
	}
}

func TestHasSkillMetadata(t *testing.T) {
	tmp := t.TempDir()

	// No metadata yet
	if HasSkillMetadata(tmp) {
		t.Error("HasSkillMetadata should return false when no metadata exists")
	}

	// Save metadata
	meta := SkillMetadata{
		RepoURL:    "https://github.com/anthropics/skills",
		PathInRepo: "skills/logger",
		PulledAt:   time.Now(),
		CommitHash: "abc123",
	}
	if err := SaveSkillMetadata(tmp, meta); err != nil {
		t.Fatalf("SaveSkillMetadata error: %v", err)
	}

	// Now it should exist
	if !HasSkillMetadata(tmp) {
		t.Error("HasSkillMetadata should return true after saving metadata")
	}
}

func TestSaveSkillMetadata_HumanReadableJSON(t *testing.T) {
	tmp := t.TempDir()

	meta := SkillMetadata{
		RepoURL:    "https://github.com/anthropics/skills",
		PathInRepo: "skills/logger",
		PulledAt:   time.Date(2026, 5, 6, 10, 30, 0, 0, time.UTC),
		CommitHash: "abc123def456",
	}

	if err := SaveSkillMetadata(tmp, meta); err != nil {
		t.Fatalf("SaveSkillMetadata error: %v", err)
	}

	// Read raw file to verify it's indented (human-readable)
	data, err := os.ReadFile(filepath.Join(tmp, SkillMetadataFile))
	if err != nil {
		t.Fatalf("failed to read metadata file: %v", err)
	}

	// Should be indented JSON (contains "  " indentation)
	content := string(data)
	if !contains(content, "  \"repo_url\"") {
		t.Error("expected indented JSON output, got non-indented")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("metadata file is not valid JSON: %v", err)
	}
}

func TestSaveAndLoadRepoMeta(t *testing.T) {
	tmp := t.TempDir()

	meta := NewRepoMetadata{
		RepoURL:    "https://github.com/anthropics/skills",
		PulledAt:   time.Date(2026, 5, 6, 10, 30, 0, 0, time.UTC),
		CommitHash: "abc123def456",
	}

	if err := SaveRepoMeta(tmp, meta); err != nil {
		t.Fatalf("SaveRepoMeta error: %v", err)
	}

	got, err := LoadRepoMeta(tmp)
	if err != nil {
		t.Fatalf("LoadRepoMeta error: %v", err)
	}

	if got.RepoURL != meta.RepoURL {
		t.Errorf("RepoURL = %q, want %q", got.RepoURL, meta.RepoURL)
	}
	if !got.PulledAt.Equal(meta.PulledAt) {
		t.Errorf("PulledAt = %v, want %v", got.PulledAt, meta.PulledAt)
	}
	if got.CommitHash != meta.CommitHash {
		t.Errorf("CommitHash = %q, want %q", got.CommitHash, meta.CommitHash)
	}
}

func TestLoadRepoMeta_Missing(t *testing.T) {
	tmp := t.TempDir()

	_, err := LoadRepoMeta(tmp)
	if err == nil {
		t.Error("expected error for missing repo metadata, got nil")
	}
}

func TestSaveRepoMeta_HumanReadableJSON(t *testing.T) {
	tmp := t.TempDir()

	meta := NewRepoMetadata{
		RepoURL:    "https://github.com/anthropics/skills",
		PulledAt:   time.Now(),
		CommitHash: "abc123",
	}

	if err := SaveRepoMeta(tmp, meta); err != nil {
		t.Fatalf("SaveRepoMeta error: %v", err)
	}

	// Read raw file to verify it's indented
	data, err := os.ReadFile(filepath.Join(tmp, RepoMetadataFile))
	if err != nil {
		t.Fatalf("failed to read repo metadata file: %v", err)
	}

	content := string(data)
	if !contains(content, "  \"repo_url\"") {
		t.Error("expected indented JSON output, got non-indented")
	}
}
