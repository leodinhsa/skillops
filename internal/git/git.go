package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// validateURL checks if the URL is a valid git repository URL
func validateURL(repoURL string) error {
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return fmt.Errorf("repository URL cannot be empty")
	}

	// Check for valid prefixes
	hasValidPrefix := strings.HasPrefix(repoURL, "https://") ||
		strings.HasPrefix(repoURL, "http://") ||
		strings.HasPrefix(repoURL, "git@") ||
		strings.HasPrefix(repoURL, "git://") ||
		strings.HasPrefix(repoURL, "file://")

	if !hasValidPrefix {
		return fmt.Errorf("invalid repository URL format: %s", repoURL)
	}

	// Check for path traversal in URL
	if strings.Contains(repoURL, "..") {
		return fmt.Errorf("repository URL cannot contain '..'")
	}

	return nil
}

func Clone(repoURL, destDir string) error {
	// Validate URL
	if err := validateURL(repoURL); err != nil {
		return fmt.Errorf("invalid repository URL: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Remove existing directory if it exists
	if _, err := os.Stat(destDir); err == nil {
		if err := os.RemoveAll(destDir); err != nil {
			return fmt.Errorf("failed to remove existing directory: %w", err)
		}
	}

	// Create parent directory
	parentDir := filepath.Dir(destDir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Clone the repository with context
	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", repoURL, destDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("git clone timed out")
		}
		return fmt.Errorf("git clone failed: %w\noutput: %s", err, string(output))
	}

	return nil
}

func ExtractRepoName(repoURL string) string {
	parts := strings.Split(ExtractFullRepoPath(repoURL), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func ExtractFullRepoPath(repoURL string) string {
	// Handle SSH URLs: git@github.com:user/repo.git
	if strings.HasPrefix(repoURL, "git@") {
		// Remove git@ prefix
		parts := strings.Split(strings.TrimPrefix(repoURL, "git@"), ":")
		if len(parts) >= 2 {
			repoPath := parts[1]
			// Remove .git suffix
			return strings.TrimSuffix(repoPath, ".git")
		}
	}

	// Handle HTTPS URLs: https://github.com/user/repo.git
	// Remove protocol
	url := strings.TrimPrefix(repoURL, "https://")
	url = strings.TrimPrefix(url, "http://")

	// Remove domain part (first part before /)
	parts := strings.Split(url, "/")
	if len(parts) > 1 {
		url = strings.Join(parts[1:], "/")
	}

	// Remove .git suffix
	return strings.TrimSuffix(url, ".git")
}

func NormalizeRepoURL(repoURL string) string {
	repoURL = strings.TrimSpace(repoURL)

	// If it's already a full URL, return as-is
	if strings.HasPrefix(repoURL, "http://") || strings.HasPrefix(repoURL, "https://") {
		return repoURL
	}

	// If it starts with git@, it's SSH format
	if strings.HasPrefix(repoURL, "git@") {
		return repoURL
	}

	// If it's a shorthand like "user/repo", convert to SSH
	if !strings.Contains(repoURL, "/") {
		return "git@github.com:" + repoURL + ".git"
	}

	// Convert user/repo to SSH format
	return "git@github.com:" + repoURL + ".git"
}

// GetLatestCommit returns the HEAD commit hash from a git repository at the given path.
// Returns an empty string if the commit hash cannot be determined.
func GetLatestCommit(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// ParseRepoURL extracts host and repoPath from a git repository URL.
// Supports HTTPS, SSH, and self-hosted formats including multi-level groups.
// Strips .git suffix. Returns the identity prefix (host + "/" + repoPath).
//
// Examples:
//   - "https://github.com/anthropics/skills.git" → host="github.com", repoPath="anthropics/skills"
//   - "git@github.com:owner/repo.git" → host="github.com", repoPath="owner/repo"
//   - "https://gitlab.com/group/subgroup/project" → host="gitlab.com", repoPath="group/subgroup/project"
func ParseRepoURL(repoURL string) (host, repoPath string, err error) {
	url := strings.TrimSpace(repoURL)
	if url == "" {
		return "", "", fmt.Errorf("repository URL cannot be empty")
	}

	// Strip .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Detect URL format and extract host + repoPath
	switch {
	case strings.HasPrefix(url, "git@"):
		// SSH format: git@github.com:owner/repo
		url = strings.TrimPrefix(url, "git@")
		parts := strings.SplitN(url, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("invalid SSH URL format: %s", repoURL)
		}
		host = parts[0]
		repoPath = parts[1]

	case strings.HasPrefix(url, "https://"):
		url = strings.TrimPrefix(url, "https://")
		parts := strings.SplitN(url, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("invalid HTTPS URL format: %s", repoURL)
		}
		host = parts[0]
		repoPath = parts[1]

	case strings.HasPrefix(url, "http://"):
		url = strings.TrimPrefix(url, "http://")
		parts := strings.SplitN(url, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("invalid HTTP URL format: %s", repoURL)
		}
		host = parts[0]
		repoPath = parts[1]

	default:
		return "", "", fmt.Errorf("unsupported URL format (must be HTTPS or SSH): %s", repoURL)
	}

	// Strip trailing slash from repoPath
	repoPath = strings.TrimSuffix(repoPath, "/")

	// Validate repoPath has at least 2 components (owner/repo minimum)
	pathComponents := strings.Split(repoPath, "/")
	if len(pathComponents) < 2 {
		return "", "", fmt.Errorf("URL must contain at least owner/repo after host: %s", repoURL)
	}

	// Validate all components for path traversal and empty values
	for _, component := range pathComponents {
		if component == "" {
			return "", "", fmt.Errorf("invalid URL: path component cannot be empty: %s", repoURL)
		}
		if component == "." {
			return "", "", fmt.Errorf("invalid URL: path component cannot be '.': %s", repoURL)
		}
		if component == ".." {
			return "", "", fmt.Errorf("invalid URL: path component cannot be '..' (path traversal): %s", repoURL)
		}
	}

	return host, repoPath, nil
}
