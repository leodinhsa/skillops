package tui

import (
	"strings"
	"testing"

	"skillops/internal/config"
)

func TestDetectConflicts_NoConflicts(t *testing.T) {
	identities := []string{
		"github.com/anthropics/skills/skills/logger",
		"github.com/anthropics/skills/skills/auth",
		"github.com/company/utils/tools/formatter",
	}
	localConfig := config.LocalConfig{
		SymlinkNames: map[string]string{},
	}

	conflicts := DetectConflicts(identities, localConfig)

	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d: %+v", len(conflicts), conflicts)
	}
}

func TestDetectConflicts_TwoSkillsSameShortName(t *testing.T) {
	identities := []string{
		"github.com/company-a/utils/tools/logger",
		"github.com/company-b/helpers/services/logger",
	}
	localConfig := config.LocalConfig{
		SymlinkNames: map[string]string{},
	}

	conflicts := DetectConflicts(identities, localConfig)

	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d: %+v", len(conflicts), conflicts)
	}

	c := conflicts[0]
	if c.SymlinkName != "logger" {
		t.Errorf("expected symlink name 'logger', got %q", c.SymlinkName)
	}
	if len(c.Identities) != 2 {
		t.Errorf("expected 2 identities in conflict, got %d", len(c.Identities))
	}
}

func TestDetectConflicts_ThreeSkillsSameShortName(t *testing.T) {
	identities := []string{
		"github.com/company-a/utils/tools/logger",
		"github.com/company-b/helpers/services/logger",
		"gitlab.com/devops/infra/monitoring/logger",
	}
	localConfig := config.LocalConfig{
		SymlinkNames: map[string]string{},
	}

	conflicts := DetectConflicts(identities, localConfig)

	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d: %+v", len(conflicts), conflicts)
	}

	c := conflicts[0]
	if c.SymlinkName != "logger" {
		t.Errorf("expected symlink name 'logger', got %q", c.SymlinkName)
	}
	if len(c.Identities) != 3 {
		t.Errorf("expected 3 identities in conflict, got %d", len(c.Identities))
	}
}

func TestDetectConflicts_CustomNamePreventsConflict(t *testing.T) {
	identities := []string{
		"github.com/company-a/utils/tools/logger",
		"github.com/company-b/helpers/services/logger",
	}
	localConfig := config.LocalConfig{
		SymlinkNames: map[string]string{
			"github.com/company-a/utils/tools/logger": "logger-utils",
		},
	}

	conflicts := DetectConflicts(identities, localConfig)

	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts (custom name should prevent it), got %d: %+v", len(conflicts), conflicts)
	}
}

func TestFormatConflictError_SingleConflict(t *testing.T) {
	conflicts := []Conflict{
		{
			SymlinkName: "logger",
			Identities: []string{
				"github.com/company-a/utils/tools/logger",
				"github.com/company-b/helpers/services/logger",
			},
		},
	}

	err := FormatConflictError(conflicts)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	msg := err.Error()

	// Check key elements are present
	if !strings.Contains(msg, "Symlink conflicts detected (non-interactive mode)") {
		t.Error("error should mention non-interactive mode")
	}
	if !strings.Contains(msg, "Symlink name: logger") {
		t.Error("error should list the conflicting symlink name")
	}
	if !strings.Contains(msg, "github.com/company-a/utils/tools/logger") {
		t.Error("error should list first conflicting identity")
	}
	if !strings.Contains(msg, "github.com/company-b/helpers/services/logger") {
		t.Error("error should list second conflicting identity")
	}
	if !strings.Contains(msg, "symlink_names") {
		t.Error("error should suggest symlink_names config")
	}
	if !strings.Contains(msg, ".skillops/config.json") {
		t.Error("error should reference config.json")
	}
	if !strings.Contains(msg, "skillops sync") {
		t.Error("error should suggest running skillops sync")
	}
}

func TestFormatConflictError_MultipleConflicts(t *testing.T) {
	conflicts := []Conflict{
		{
			SymlinkName: "logger",
			Identities: []string{
				"github.com/company-a/utils/tools/logger",
				"github.com/company-b/helpers/services/logger",
			},
		},
		{
			SymlinkName: "auth",
			Identities: []string{
				"github.com/org-x/platform/auth",
				"gitlab.com/team-y/services/auth",
			},
		},
	}

	err := FormatConflictError(conflicts)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	msg := err.Error()

	// Both conflicts should be listed
	if !strings.Contains(msg, "Symlink name: logger") {
		t.Error("error should list 'logger' conflict")
	}
	if !strings.Contains(msg, "Symlink name: auth") {
		t.Error("error should list 'auth' conflict")
	}
	// All identities should be present
	if !strings.Contains(msg, "github.com/company-a/utils/tools/logger") {
		t.Error("missing identity: github.com/company-a/utils/tools/logger")
	}
	if !strings.Contains(msg, "github.com/company-b/helpers/services/logger") {
		t.Error("missing identity: github.com/company-b/helpers/services/logger")
	}
	if !strings.Contains(msg, "github.com/org-x/platform/auth") {
		t.Error("missing identity: github.com/org-x/platform/auth")
	}
	if !strings.Contains(msg, "gitlab.com/team-y/services/auth") {
		t.Error("missing identity: gitlab.com/team-y/services/auth")
	}
}

func TestSuggestCustomName(t *testing.T) {
	tests := []struct {
		identity  string
		shortName string
		want      string
	}{
		{
			identity:  "github.com/company-a/utils/tools/logger",
			shortName: "logger",
			want:      "logger-utils",
		},
		{
			identity:  "github.com/company-b/helpers/services/logger",
			shortName: "logger",
			want:      "logger-helpers",
		},
		{
			identity:  "github.com/anthropics/skills/skills/auth",
			shortName: "auth",
			want:      "auth-skills",
		},
	}

	for _, tt := range tests {
		t.Run(tt.identity, func(t *testing.T) {
			got := suggestCustomName(tt.identity, tt.shortName)
			if got != tt.want {
				t.Errorf("suggestCustomName(%q, %q) = %q, want %q", tt.identity, tt.shortName, got, tt.want)
			}
		})
	}
}

func TestHandleConflicts_NonTTY(t *testing.T) {
	// In test environments, stdin is typically not a TTY, so HandleConflicts
	// should return an error (non-TTY path).
	conflicts := []Conflict{
		{
			SymlinkName: "logger",
			Identities: []string{
				"github.com/company-a/utils/tools/logger",
				"github.com/company-b/helpers/services/logger",
			},
		},
	}

	result, err := HandleConflicts(conflicts)
	if err == nil {
		t.Fatal("expected error in non-TTY environment, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result in non-TTY, got %v", result)
	}

	// Verify the error message is the formatted conflict error
	msg := err.Error()
	if !strings.Contains(msg, "Symlink conflicts detected") {
		t.Error("error should be the formatted conflict error")
	}
}
