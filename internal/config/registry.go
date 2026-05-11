package config

import (
	"fmt"
	"sort"
	"strings"
)

// ValidateRegistryURL checks that a registry URL is valid.
// It must not have a trailing slash, must be HTTPS or SSH format,
// and must contain at least a path after the host (e.g., owner/repo).
//
// Note: This function is intended to be called when adding/modifying registries
// (e.g., in the add command or config commands), not during config load.
// This avoids rejecting existing configs if validation rules evolve.
func ValidateRegistryURL(url string) error {
	if url == "" {
		return fmt.Errorf("registry URL cannot be empty")
	}
	if strings.HasSuffix(url, "/") {
		return fmt.Errorf("registry URL must not have a trailing slash: %s", url)
	}
	// Only HTTPS and SSH are supported (no plain HTTP)
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "git@") {
		return fmt.Errorf("registry URL must be HTTPS or SSH format: %s", url)
	}

	// Validate that URL has a meaningful path after host (at least owner/repo)
	normalized := NormalizeRegistryURL(url)
	parts := strings.SplitN(normalized, "/", 2)
	if len(parts) < 2 || parts[1] == "" {
		return fmt.Errorf("registry URL must contain a path after host (e.g., owner/repo): %s", url)
	}

	return nil
}

// NormalizeRegistryURL converts a registry URL to the format used in skill identities.
// It strips the protocol (https://, http://, git@), replaces ":" with "/" for SSH,
// strips .git suffix, and removes trailing slashes.
//
// Examples:
//   - "https://github.com/anthropics/skills" → "github.com/anthropics/skills"
//   - "git@github.com:company/utils.git" → "github.com/company/utils"
//   - "https://gitlab.common.datumhq.com/group/subgroup/project" → "gitlab.common.datumhq.com/group/subgroup/project"
func NormalizeRegistryURL(registryURL string) string {
	url := strings.TrimSpace(registryURL)

	// Strip .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Strip protocol and normalize SSH format
	switch {
	case strings.HasPrefix(url, "https://"):
		url = strings.TrimPrefix(url, "https://")
	case strings.HasPrefix(url, "http://"):
		url = strings.TrimPrefix(url, "http://")
	case strings.HasPrefix(url, "git@"):
		// SSH format: git@github.com:owner/repo → github.com/owner/repo
		url = strings.TrimPrefix(url, "git@")
		url = strings.Replace(url, ":", "/", 1)
	}

	// Strip trailing slash
	url = strings.TrimSuffix(url, "/")

	return url
}

// MatchRegistry finds the registry that matches a skill identity by prefix matching.
// It normalizes each registry URL and checks if the skill identity starts with
// the normalized prefix followed by "/". Returns the clone URL and path-in-repo.
//
// Registries are sorted by priority (lower number = higher priority).
// Returns an error if no registry matches.
//
// Note on overlapping registries: If two registries have overlapping prefixes
// (e.g., "github.com/org/repo" and "github.com/org/repo/subdir"), the one with
// higher priority (lower number) wins. This is by design — users control matching
// via priority ordering.
//
// Examples:
//   - identity="github.com/anthropics/skills/skills/logger", registry URL="https://github.com/anthropics/skills"
//     → cloneURL="https://github.com/anthropics/skills", pathInRepo="skills/logger"
//   - identity="github.com/company/utils/tools/logger", registry URL="git@github.com:company/utils"
//     → cloneURL="git@github.com:company/utils", pathInRepo="tools/logger"
func MatchRegistry(skillIdentity string, registries []Registry) (cloneURL, pathInRepo string, err error) {
	if len(registries) == 0 {
		return "", "", fmt.Errorf("no registries configured for skill: %s", skillIdentity)
	}

	// Sort registries by priority (lower number = higher priority)
	sorted := make([]Registry, len(registries))
	copy(sorted, registries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority < sorted[j].Priority
	})

	// Try each registry in priority order
	for _, reg := range sorted {
		normalized := NormalizeRegistryURL(reg.URL)
		if normalized == "" {
			continue
		}

		// Check if identity starts with normalized prefix followed by "/"
		// This prevents false positives (e.g., "github.com/anthropics/skills-extra"
		// should NOT match registry "https://github.com/anthropics/skills")
		prefix := normalized + "/"
		if strings.HasPrefix(skillIdentity, prefix) {
			pathInRepo = strings.TrimPrefix(skillIdentity, prefix)
			if pathInRepo == "" {
				continue // identity IS the registry prefix, no path-in-repo
			}
			return reg.URL, pathInRepo, nil
		}
	}

	return "", "", fmt.Errorf("no registry found for skill: %s", skillIdentity)
}

// MatchesRegistry checks if a skill identity belongs to a specific registry.
// Returns true if the identity starts with the normalized registry URL prefix followed by "/".
func MatchesRegistry(skillIdentity string, reg Registry) bool {
	normalized := NormalizeRegistryURL(reg.URL)
	if normalized == "" {
		return false
	}
	return strings.HasPrefix(skillIdentity, normalized+"/")
}
