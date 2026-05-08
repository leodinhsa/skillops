package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractRepoName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://github.com/user/my-repo.git", "my-repo"},
		{"https://github.com/user/my-repo", "my-repo"},
		{"git@github.com:user/my-repo.git", "my-repo"},
		{"git@github.com:user/my-repo", "my-repo"},
		{"https://github.com/org/nested/repo.git", "repo"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ExtractRepoName(tt.input)
			if got != tt.want {
				t.Errorf("ExtractRepoName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractFullRepoPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://github.com/user/my-repo.git", "user/my-repo"},
		{"https://github.com/user/my-repo", "user/my-repo"},
		{"http://github.com/user/my-repo.git", "user/my-repo"},
		{"git@github.com:user/my-repo.git", "user/my-repo"},
		{"git@github.com:user/my-repo", "user/my-repo"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ExtractFullRepoPath(tt.input)
			if got != tt.want {
				t.Errorf("ExtractFullRepoPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeRepoURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://github.com/user/repo", "https://github.com/user/repo"},
		{"http://github.com/user/repo", "http://github.com/user/repo"},
		{"git@github.com:user/repo.git", "git@github.com:user/repo.git"},
		{"user/repo", "git@github.com:user/repo.git"},
		{"  user/repo  ", "git@github.com:user/repo.git"}, // trims whitespace
		{"myrepo", "git@github.com:myrepo.git"},           // no slash → single name
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeRepoURL(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeRepoURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"https://github.com/user/repo", false},
		{"http://github.com/user/repo", false},
		{"git@github.com:user/repo.git", false},
		{"git://github.com/user/repo.git", false},
		{"", true},
		{"   ", true},
		{"ftp://github.com/user/repo", true},
		{"github.com/user/repo", true},
		{"https://github.com/../etc/passwd", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			err := validateURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestParseRepoURL_HTTPS(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantHost     string
		wantRepoPath string
	}{
		{
			name:         "GitHub HTTPS with .git",
			input:        "https://github.com/anthropics/skills.git",
			wantHost:     "github.com",
			wantRepoPath: "anthropics/skills",
		},
		{
			name:         "GitHub HTTPS without .git",
			input:        "https://github.com/anthropics/skills",
			wantHost:     "github.com",
			wantRepoPath: "anthropics/skills",
		},
		{
			name:         "GitLab multi-level groups",
			input:        "https://gitlab.com/group/subgroup/project",
			wantHost:     "gitlab.com",
			wantRepoPath: "group/subgroup/project",
		},
		{
			name:         "Self-hosted GitLab deep path",
			input:        "https://gitlab.common.datumhq.com/datumhq-consulting-vn/management/datum-skills/software-skills",
			wantHost:     "gitlab.common.datumhq.com",
			wantRepoPath: "datumhq-consulting-vn/management/datum-skills/software-skills",
		},
		{
			name:         "HTTP URL",
			input:        "http://github.com/user/repo.git",
			wantHost:     "github.com",
			wantRepoPath: "user/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, repoPath, err := ParseRepoURL(tt.input)
			if err != nil {
				t.Fatalf("ParseRepoURL(%q) unexpected error: %v", tt.input, err)
			}
			if host != tt.wantHost {
				t.Errorf("host = %q, want %q", host, tt.wantHost)
			}
			if repoPath != tt.wantRepoPath {
				t.Errorf("repoPath = %q, want %q", repoPath, tt.wantRepoPath)
			}
		})
	}
}

func TestParseRepoURL_SSH(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantHost     string
		wantRepoPath string
	}{
		{
			name:         "GitHub SSH with .git",
			input:        "git@github.com:owner/repo.git",
			wantHost:     "github.com",
			wantRepoPath: "owner/repo",
		},
		{
			name:         "GitHub SSH without .git",
			input:        "git@github.com:owner/repo",
			wantHost:     "github.com",
			wantRepoPath: "owner/repo",
		},
		{
			name:         "Bitbucket SSH",
			input:        "git@bitbucket.org:org/repo.git",
			wantHost:     "bitbucket.org",
			wantRepoPath: "org/repo",
		},
		{
			name:         "Self-hosted SSH",
			input:        "git@gitlab.company.internal:team/backend/api-skills.git",
			wantHost:     "gitlab.company.internal",
			wantRepoPath: "team/backend/api-skills",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, repoPath, err := ParseRepoURL(tt.input)
			if err != nil {
				t.Fatalf("ParseRepoURL(%q) unexpected error: %v", tt.input, err)
			}
			if host != tt.wantHost {
				t.Errorf("host = %q, want %q", host, tt.wantHost)
			}
			if repoPath != tt.wantRepoPath {
				t.Errorf("repoPath = %q, want %q", repoPath, tt.wantRepoPath)
			}
		})
	}
}

func TestParseRepoURL_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "empty string",
			input:   "",
			wantErr: "cannot be empty",
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: "cannot be empty",
		},
		{
			name:    "unsupported protocol",
			input:   "ftp://github.com/user/repo",
			wantErr: "unsupported URL format",
		},
		{
			name:    "no protocol",
			input:   "github.com/user/repo",
			wantErr: "unsupported URL format",
		},
		{
			name:    "HTTPS missing path",
			input:   "https://github.com",
			wantErr: "invalid HTTPS URL format",
		},
		{
			name:    "HTTPS only host with slash",
			input:   "https://github.com/",
			wantErr: "invalid HTTPS URL format",
		},
		{
			name:    "HTTPS single path component",
			input:   "https://github.com/onlyone",
			wantErr: "at least owner/repo",
		},
		{
			name:    "SSH missing colon path",
			input:   "git@github.com",
			wantErr: "invalid SSH URL format",
		},
		{
			name:    "SSH empty path after colon",
			input:   "git@github.com:",
			wantErr: "invalid SSH URL format",
		},
		{
			name:    "path traversal in components",
			input:   "https://github.com/../etc/passwd",
			wantErr: "cannot be '..'",
		},
		{
			name:    "dot component",
			input:   "https://github.com/./repo",
			wantErr: "cannot be '.'",
		},
		{
			name:    "empty component (double slash)",
			input:   "https://github.com/owner//repo",
			wantErr: "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, repoPath, err := ParseRepoURL(tt.input)
			if err == nil {
				t.Fatalf("ParseRepoURL(%q) expected error, got host=%q repoPath=%q", tt.input, host, repoPath)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ParseRepoURL(%q) error = %q, want to contain %q", tt.input, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestParseRepoURL_IdentityPrefix(t *testing.T) {
	// Verify that host + "/" + repoPath forms the correct identity prefix
	tests := []struct {
		input      string
		wantPrefix string
	}{
		{"https://github.com/anthropics/skills.git", "github.com/anthropics/skills"},
		{"git@github.com:company/utils.git", "github.com/company/utils"},
		{"https://gitlab.com/group/subgroup/project", "gitlab.com/group/subgroup/project"},
		{"git@bitbucket.org:org/repo", "bitbucket.org/org/repo"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			host, repoPath, err := ParseRepoURL(tt.input)
			if err != nil {
				t.Fatalf("ParseRepoURL(%q) error: %v", tt.input, err)
			}
			prefix := host + "/" + repoPath
			if prefix != tt.wantPrefix {
				t.Errorf("identity prefix = %q, want %q", prefix, tt.wantPrefix)
			}
		})
	}
}

func TestGetLatestCommit(t *testing.T) {
	// Create a temp git repo
	tmp := t.TempDir()

	cmd := exec.Command("git", "init", tmp)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, output)
	}

	// Configure git
	cmd = exec.Command("git", "-C", tmp, "config", "user.email", "test@test.com")
	cmd.CombinedOutput()
	cmd = exec.Command("git", "-C", tmp, "config", "user.name", "Test")
	cmd.CombinedOutput()

	// Create a file and commit
	if err := os.WriteFile(filepath.Join(tmp, "test.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "-C", tmp, "add", ".")
	cmd.CombinedOutput()
	cmd = exec.Command("git", "-C", tmp, "commit", "-m", "test commit")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v\n%s", err, output)
	}

	// Get commit hash
	hash := GetLatestCommit(tmp)
	if hash == "" {
		t.Error("GetLatestCommit returned empty string for valid repo")
	}
	if len(hash) != 40 {
		t.Errorf("expected 40-char SHA hash, got %d chars: %q", len(hash), hash)
	}
}

func TestGetLatestCommit_InvalidPath(t *testing.T) {
	// Non-git directory should return empty string
	tmp := t.TempDir()
	hash := GetLatestCommit(tmp)
	if hash != "" {
		t.Errorf("expected empty string for non-git dir, got %q", hash)
	}
}
