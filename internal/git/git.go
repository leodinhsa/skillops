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
		strings.HasPrefix(repoURL, "git://")

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
	// Handle SSH URLs: git@github.com:user/repo.git
	if strings.HasPrefix(repoURL, "git@") {
		// Remove git@ prefix
		parts := strings.Split(strings.TrimPrefix(repoURL, "git@"), ":")
		if len(parts) >= 2 {
			repoPath := parts[1]
			// Remove .git suffix
			repoPath = strings.TrimSuffix(repoPath, ".git")
			parts = strings.Split(repoPath, "/")
			if len(parts) > 0 {
				return parts[len(parts)-1]
			}
		}
	}

	// Handle HTTPS URLs: https://github.com/user/repo.git
	// Remove protocol
	url := strings.TrimPrefix(repoURL, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "github.com/")

	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Get the last part (repo name)
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
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
