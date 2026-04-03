package git

import (
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
