package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"skillops/internal/config"
	"strings"
)

// ValidateName checks for path traversal attempts and empty names
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("name cannot contain '..'")
	}
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, "\\") {
		return fmt.Errorf("name cannot start with path separator")
	}
	return nil
}

// IsAgenticEnabled checks if the agentic exists globally and in the current project root
func IsAgenticEnabled(name string) (bool, string, error) {
	relPath, err := config.GetAgenticPath(name)
	if err != nil {
		return false, "", err
	}

	parts := strings.Split(filepath.ToSlash(relPath), "/")
	if len(parts) == 0 {
		return false, "", nil
	}
	rootSubDir := parts[0]

	cwd, err := os.Getwd()
	if err != nil {
		return false, rootSubDir, err
	}

	fullPath := filepath.Join(cwd, rootSubDir)
	info, err := os.Stat(fullPath)
	if err != nil {
		return false, rootSubDir, nil
	}
	return info.IsDir(), rootSubDir, nil
}

// CopyDir copies a directory recursively from src to dst
func CopyDir(src string, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		// File
		buf, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, buf, info.Mode())
	})
}
