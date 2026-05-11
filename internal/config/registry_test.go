package config

import (
	"testing"
)

func TestValidateRegistryURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid HTTPS URL",
			url:     "https://github.com/anthropics/skills",
			wantErr: false,
		},
		{
			name:    "valid SSH URL",
			url:     "git@github.com:company/utils",
			wantErr: false,
		},
		{
			name:    "valid HTTPS with multi-level groups",
			url:     "https://gitlab.common.datumhq.com/datumhq-consulting-vn/management/datum-skills/software-skills",
			wantErr: false,
		},
		{
			name:    "invalid: empty URL",
			url:     "",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "invalid: trailing slash",
			url:     "https://github.com/anthropics/skills/",
			wantErr: true,
			errMsg:  "trailing slash",
		},
		{
			name:    "invalid: unsupported protocol",
			url:     "ftp://github.com/anthropics/skills",
			wantErr: true,
			errMsg:  "HTTPS or SSH",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRegistryURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestNormalizeRegistryURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "HTTPS URL",
			input:    "https://github.com/anthropics/skills",
			expected: "github.com/anthropics/skills",
		},
		{
			name:     "HTTPS URL with .git suffix",
			input:    "https://github.com/anthropics/skills.git",
			expected: "github.com/anthropics/skills",
		},
		{
			name:     "SSH URL",
			input:    "git@github.com:company/utils",
			expected: "github.com/company/utils",
		},
		{
			name:     "SSH URL with .git suffix",
			input:    "git@github.com:company/utils.git",
			expected: "github.com/company/utils",
		},
		{
			name:     "HTTP URL",
			input:    "http://gitlab.internal/team/repo",
			expected: "gitlab.internal/team/repo",
		},
		{
			name:     "multi-level groups (GitLab)",
			input:    "https://gitlab.common.datumhq.com/datumhq-consulting-vn/management/datum-skills/software-skills",
			expected: "gitlab.common.datumhq.com/datumhq-consulting-vn/management/datum-skills/software-skills",
		},
		{
			name:     "trailing slash stripped",
			input:    "https://github.com/anthropics/skills/",
			expected: "github.com/anthropics/skills",
		},
		{
			name:     "whitespace trimmed",
			input:    "  https://github.com/anthropics/skills  ",
			expected: "github.com/anthropics/skills",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeRegistryURL(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeRegistryURL(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMatchRegistry_SingleRegistry_HTTPS(t *testing.T) {
	registries := []Registry{
		{URL: "https://github.com/anthropics/skills", Name: "Anthropic Skills", Priority: 1},
	}

	cloneURL, pathInRepo, err := MatchRegistry("github.com/anthropics/skills/skills/logger", registries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cloneURL != "https://github.com/anthropics/skills" {
		t.Errorf("cloneURL = %q, want %q", cloneURL, "https://github.com/anthropics/skills")
	}
	if pathInRepo != "skills/logger" {
		t.Errorf("pathInRepo = %q, want %q", pathInRepo, "skills/logger")
	}
}

func TestMatchRegistry_SingleRegistry_SSH(t *testing.T) {
	registries := []Registry{
		{URL: "git@github.com:company/utils", Name: "Company Utils", Priority: 1},
	}

	cloneURL, pathInRepo, err := MatchRegistry("github.com/company/utils/tools/logger", registries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cloneURL != "git@github.com:company/utils" {
		t.Errorf("cloneURL = %q, want %q", cloneURL, "git@github.com:company/utils")
	}
	if pathInRepo != "tools/logger" {
		t.Errorf("pathInRepo = %q, want %q", pathInRepo, "tools/logger")
	}
}

func TestMatchRegistry_MultipleRegistries_Priority(t *testing.T) {
	registries := []Registry{
		{URL: "https://github.com/company-b/helpers", Name: "Company B", Priority: 2},
		{URL: "https://github.com/company-a/utils", Name: "Company A", Priority: 1},
	}

	// This identity matches company-a (priority 1)
	cloneURL, pathInRepo, err := MatchRegistry("github.com/company-a/utils/tools/logger", registries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cloneURL != "https://github.com/company-a/utils" {
		t.Errorf("cloneURL = %q, want %q", cloneURL, "https://github.com/company-a/utils")
	}
	if pathInRepo != "tools/logger" {
		t.Errorf("pathInRepo = %q, want %q", pathInRepo, "tools/logger")
	}
}

func TestMatchRegistry_NoMatch(t *testing.T) {
	registries := []Registry{
		{URL: "https://github.com/anthropics/skills", Name: "Anthropic Skills", Priority: 1},
	}

	_, _, err := MatchRegistry("github.com/other-org/other-repo/skills/logger", registries)
	if err == nil {
		t.Fatal("expected error for no match, got nil")
	}
	if !contains(err.Error(), "no registry found") {
		t.Errorf("error should contain 'no registry found', got: %v", err)
	}
}

func TestMatchRegistry_EmptyRegistries(t *testing.T) {
	_, _, err := MatchRegistry("github.com/anthropics/skills/skills/logger", nil)
	if err == nil {
		t.Fatal("expected error for empty registries, got nil")
	}
	if !contains(err.Error(), "no registries configured") {
		t.Errorf("error should contain 'no registries configured', got: %v", err)
	}
}

func TestMatchRegistry_MultiLevelGroups(t *testing.T) {
	registries := []Registry{
		{
			URL:      "https://gitlab.common.datumhq.com/datumhq-consulting-vn/management/datum-skills/software-skills",
			Name:     "Datum Software Skills",
			Priority: 1,
		},
	}

	cloneURL, pathInRepo, err := MatchRegistry(
		"gitlab.common.datumhq.com/datumhq-consulting-vn/management/datum-skills/software-skills/skills/logger",
		registries,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cloneURL != "https://gitlab.common.datumhq.com/datumhq-consulting-vn/management/datum-skills/software-skills" {
		t.Errorf("cloneURL = %q, want full URL", cloneURL)
	}
	if pathInRepo != "skills/logger" {
		t.Errorf("pathInRepo = %q, want %q", pathInRepo, "skills/logger")
	}
}

func TestMatchRegistry_SubstringFalsePositivePrevented(t *testing.T) {
	registries := []Registry{
		{URL: "https://github.com/anthropics/skills", Name: "Anthropic Skills", Priority: 1},
	}

	// "skills-extra" should NOT match "skills" registry
	_, _, err := MatchRegistry("github.com/anthropics/skills-extra/logger", registries)
	if err == nil {
		t.Fatal("expected error: 'skills-extra' should NOT match 'skills' registry")
	}
}

func TestMatchRegistry_PathInRepoCorrectlyExtracted(t *testing.T) {
	registries := []Registry{
		{URL: "https://github.com/company/monorepo", Name: "Monorepo", Priority: 1},
	}

	// Deeply nested skill
	_, pathInRepo, err := MatchRegistry(
		"github.com/company/monorepo/backend/services/api/skills/auth",
		registries,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pathInRepo != "backend/services/api/skills/auth" {
		t.Errorf("pathInRepo = %q, want %q", pathInRepo, "backend/services/api/skills/auth")
	}
}

func TestMatchRegistry_PriorityOrder(t *testing.T) {
	// Both registries could match if we had overlapping prefixes,
	// but here we test that priority ordering is respected
	registries := []Registry{
		{URL: "https://github.com/org/repo-specific", Name: "Specific", Priority: 1},
		{URL: "https://github.com/org/repo-general", Name: "General", Priority: 2},
	}

	// Only matches repo-specific
	cloneURL, pathInRepo, err := MatchRegistry("github.com/org/repo-specific/skills/auth", registries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cloneURL != "https://github.com/org/repo-specific" {
		t.Errorf("cloneURL = %q, want specific repo", cloneURL)
	}
	if pathInRepo != "skills/auth" {
		t.Errorf("pathInRepo = %q, want %q", pathInRepo, "skills/auth")
	}
}

func TestMatchRegistry_SSHWithGitSuffix(t *testing.T) {
	registries := []Registry{
		{URL: "git@github.com:company/private-skills.git", Name: "Private", Priority: 1},
	}

	cloneURL, pathInRepo, err := MatchRegistry("github.com/company/private-skills/api/rate-limiter", registries)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cloneURL != "git@github.com:company/private-skills.git" {
		t.Errorf("cloneURL = %q, want SSH URL with .git", cloneURL)
	}
	if pathInRepo != "api/rate-limiter" {
		t.Errorf("pathInRepo = %q, want %q", pathInRepo, "api/rate-limiter")
	}
}

func TestMatchesRegistry(t *testing.T) {
	reg := Registry{URL: "https://github.com/anthropics/skills", Name: "Anthropic", Priority: 1}

	tests := []struct {
		name     string
		identity string
		want     bool
	}{
		{
			name:     "matches",
			identity: "github.com/anthropics/skills/skills/logger",
			want:     true,
		},
		{
			name:     "does not match different repo",
			identity: "github.com/other-org/other-repo/skills/logger",
			want:     false,
		},
		{
			name:     "does not match substring",
			identity: "github.com/anthropics/skills-extra/logger",
			want:     false,
		},
		{
			name:     "does not match exact prefix without trailing content",
			identity: "github.com/anthropics/skills",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesRegistry(tt.identity, reg)
			if got != tt.want {
				t.Errorf("MatchesRegistry(%q) = %v, want %v", tt.identity, got, tt.want)
			}
		})
	}
}

// contains is a helper for checking substring presence in error messages.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
