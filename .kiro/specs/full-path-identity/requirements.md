# Requirements Document: Full-Path Identity Support

## Introduction

This feature fundamentally redesigns skillops skill identity format to use full paths including host and repository information. The current 2-level `repo/skill` format causes critical issues:

1. **Repository collision**: Multiple organizations can have repositories with the same name (e.g., `company-a/common-utils` vs `company-b/common-utils`)
2. **Nested path limitation**: Cannot represent skills in nested directory structures (e.g., `repo/backend/services/api/skills/auth`)
3. **Team collaboration friction**: No mechanism to share registry URLs, requiring manual setup for each team member
4. **Source ambiguity**: Cannot determine which git host (GitHub, GitLab, Bitbucket, self-hosted) a skill comes from

The new identity format `<host>/<repo-path>/<path-to-skill>` solves all these issues while enabling:
- Zero-config team onboarding (registries in project config)
- Multi-source skill management (GitHub, GitLab, Bitbucket, self-hosted)
- Arbitrary nesting depth support
- Collision-free skill identification
- Full support for multi-level groups (GitLab subgroups)

**Note:** This is a breaking change. There are no existing users, so no migration or backward compatibility is required.

## Key Design Decision: Registry-Based Repo Boundary Detection

The identity string `<host>/<repo-path>/<path-to-skill>` does NOT encode where the repository ends and the skill path begins. This boundary is determined by **registry URL prefix matching** at runtime.

**Rationale:** Git repository URLs can have arbitrary depth (e.g., `gitlab.com/group/subgroup/project`). The last component of the clone URL is always the repository name. Rather than trying to encode this boundary in the identity string (which would require a special separator), we use the registry URL to determine it.

**Example:**
- Registry URL: `https://gitlab.common.datumhq.com/datumhq-consulting-vn/management/datum-skills/software-skills`
- Identity: `gitlab.common.datumhq.com/datumhq-consulting-vn/management/datum-skills/software-skills/skills/logger`
- Match: identity starts with registry prefix → repo boundary known
- Path in repo: `skills/logger` (remainder after stripping registry prefix from identity)

## Glossary

- **Skill_Identity**: The full path identifier for a skill in the format `<host>/<repo-path>/<path-to-skill>` (e.g., `github.com/anthropics/skills/skills/skill-creator`). The boundary between repo-path and path-to-skill is determined by registry matching, not by parsing.
- **Host**: The git hosting platform domain (e.g., `github.com`, `gitlab.com`, `bitbucket.org`, `gitlab.company.internal`)
- **Repo_Path**: The full path from host to repository, including any groups/subgroups and the repo name itself (e.g., `anthropics/skills`, `datumhq-consulting-vn/management/datum-skills/software-skills`)
- **Path_In_Repo**: The path from repository root to skill folder (e.g., `skills/skill-creator`, `backend/services/api/skills/auth`). Determined by stripping the registry prefix from the identity.
- **Short_Name**: The final component of the identity path used as the symlink filename (e.g., `skill-creator`, `logger`)
- **Global_Store**: The directory `~/.skillops/skills/` organized by the full identity path
- **Symlink_Path**: The path where a skill symlink is created in a project's IDE directory (e.g., `.kiro/skills/skill-creator`)
- **Local_Config**: The `.skillops/config.json` file (version 2) that stores skill identities, registries, and custom symlink names per project
- **Registry**: A clone URL for a skill repository (e.g., `https://github.com/anthropics/skills`, `https://gitlab.common.datumhq.com/datumhq-consulting-vn/management/datum-skills/software-skills`). Points to the exact repository. No trailing slash.
- **Skill_Metadata**: Per-skill metadata file `.so-skill-meta.json` containing repo URL, path_in_repo, timestamp, and commit hash
- **Repo_Metadata**: Per-repo metadata file `.so-repo-meta.json` for full repository pulls
- **Custom_Symlink_Name**: User-provided symlink name to resolve conflicts when multiple skills have the same short name

## Requirements

### Requirement 1: Parse Full-Path Skill Identities

**User Story:** As a developer, I want skillops to correctly parse skill identities with full path including host and repository path, so that I can use skills from multiple sources without collision.

#### Acceptance Criteria

1. WHEN a skill identity is provided, THE Identity_Parser SHALL extract the host (first component) and the short name (final component)
2. WHEN a skill identity contains nested paths (e.g., `gitlab.common.datumhq.com/datumhq-consulting-vn/management/datum-skills/software-skills/skills/logger`), THE Identity_Parser SHALL correctly validate all components
3. THE Identity_Parser SHALL extract the base name (final component) of the path as the short name for symlink creation
4. WHEN parsing any valid skill identity, THE Identity_Parser SHALL produce a short name that contains no path separators
5. THE Identity_Parser SHALL validate that the identity has at least 3 path components (host/repo-or-group/skill minimum)
6. THE Identity_Parser SHALL validate that no component is empty, ".", or ".." to prevent path traversal attacks
7. THE Identity_Parser SHALL NOT attempt to determine the boundary between repo-path and path-to-skill (this is determined by registry matching)

### Requirement 2: Prevent Repository Collision

**User Story:** As a developer, I want skills from different organizations with the same repository name to coexist, so that I can use skills from multiple sources without conflicts.

#### Acceptance Criteria

1. WHEN two skills have identities `github.com/company-a/utils/logger` and `github.com/company-b/utils/logger`, THE System SHALL store them in separate directories
2. THE Global_Store SHALL organize skills by `<host>/<owner>/<repo>/<path-to-skill>` structure
3. WHEN pulling a skill, THE System SHALL create directories matching the full path structure
4. THE System SHALL NOT allow one repository to overwrite another with the same name but different owner
5. WHEN displaying skills, THE System SHALL show enough context to distinguish between similarly named repositories

### Requirement 3: Support Multi-Registry Configuration

**User Story:** As a team lead, I want to configure multiple skill registries in my project config, so that team members can automatically pull skills from the correct sources without manual setup.

#### Acceptance Criteria

1. THE Local_Config SHALL support a `registries` array containing registry objects with `url`, `name`, and `priority` fields
2. WHEN adding a skill to a project, THE System SHALL automatically populate the registries array from skill metadata
3. WHEN syncing a project, THE System SHALL use registries from local config to resolve and pull missing skills
4. THE System SHALL support registries from multiple git hosts (GitHub, GitLab, Bitbucket, self-hosted)
5. WHEN multiple registries are configured, THE System SHALL try them in priority order
6. THE Registry.URL SHALL be a full repository clone URL (e.g., `https://github.com/anthropics/skills`, `https://gitlab.common.datumhq.com/datumhq-consulting-vn/management/datum-skills/software-skills`) without trailing slash
7. THE System SHALL determine the path-to-skill by stripping the normalized registry URL prefix from the skill identity

### Requirement 4: Create Flat Symlinks Using Short Names

**User Story:** As a developer, I want symlinks to use only the final skill name component, so that my IDE skill directories remain flat and organized.

#### Acceptance Criteria

1. WHEN creating a symlink for any skill identity, THE Symlink_Creator SHALL use only the short name (final path component)
2. THE Symlink_Creator SHALL NOT create nested directories within the tool's skills directory (tool's skills dir stays flat; global store has nested structure)
3. WHEN the skill has a nested path, THE Symlink_Creator SHALL construct the global path using the full path structure
4. THE Symlink_Creator SHALL create symlinks that point to valid skill directories containing SKILL.md files
5. WHEN creating symlinks, THE System SHALL verify the target exists in the global store

### Requirement 5: Store Per-Skill Metadata

**User Story:** As a developer, I want each skill to have metadata about its source, so that the system can update skills correctly and I can trace their origin.

#### Acceptance Criteria

1. WHEN pulling a skill, THE System SHALL create a `.so-skill-meta.json` file in the skill directory
2. THE Skill_Metadata SHALL contain `repo_url`, `path_in_repo`, `pulled_at`, and `commit_hash` fields
3. WHEN updating a skill, THE System SHALL read metadata to determine the source repository and path
4. THE Metadata SHALL be stored in JSON format with human-readable formatting
5. WHEN metadata is missing or corrupted, THE System SHALL provide a clear error message indicating the skill cannot be updated

### Requirement 6: Enable Zero-Config Team Onboarding

**User Story:** As a new team member, I want to clone a project and run `skillops sync` to automatically get all required skills, without manually configuring registries.

#### Acceptance Criteria

1. WHEN a developer clones a project with `.skillops/config.json`, THE Config SHALL contain all necessary registry URLs
2. WHEN running `skillops sync`, THE System SHALL pull missing skills from registries in the config
3. THE System SHALL NOT require developers to manually edit `~/.skillops/config/settings.yaml`
4. WHEN a skill cannot be pulled from any configured registry, THE System SHALL display a clear error message and skip that skill
5. THE Sync_Command SHALL report the number of skills auto-pulled from registries

### Requirement 7: Detect and Resolve Symlink Conflicts

**User Story:** As a developer, I want to be able to use multiple skills with the same short name by providing custom symlink names, so that I can work with skills from different sources without conflicts.

#### Acceptance Criteria

1. WHEN two skill identities have different full paths but the same short name, THE Conflict_Detector SHALL identify this as a symlink conflict
2. WHEN a conflict is detected during `add` in a TTY environment, THE System SHALL launch an interactive TUI allowing the user to rename symlinks
3. WHEN a conflict is detected in a non-TTY environment (CI, SSH, piped), THE System SHALL fail with a descriptive error listing the conflicts and suggesting manual resolution in config.json
4. THE Conflict_Resolution_TUI SHALL display the full identity of each conflicting skill
5. THE Conflict_Resolution_TUI SHALL allow users to provide custom symlink names for each conflicting skill
6. WHEN a user provides a custom name, THE System SHALL validate it (no path separators, not empty, not "." or "..")
7. THE System SHALL store the custom symlink name mapping in local config under `symlink_names` field
8. WHEN syncing with custom symlink names, THE System SHALL recreate symlinks using the custom names
9. THE System SHALL prevent users from creating custom names that conflict with existing symlinks

### Requirement 8: Update Sync Command for Full-Path Identities

**User Story:** As a developer, I want `skillops sync` to correctly restore symlinks using full-path identities and registries, so that my project setup is consistent across machines.

#### Acceptance Criteria

1. WHEN `skillops sync` processes a full-path identity from local config, THE Sync_Command SHALL construct the correct global skill path
2. WHEN a skill is not found in the global store, THE Sync_Command SHALL attempt to pull from configured registries
3. WHEN no registry matches the skill identity, THE Sync_Command SHALL display an error and skip that skill (no fallback to metadata)
4. WHEN `skillops sync` creates a symlink, THE Sync_Command SHALL use the short name or custom name from config
5. THE Sync_Command SHALL maintain idempotent behavior (running twice produces the same result)
6. WHEN sync completes, THE Sync_Command SHALL report: symlinks created, skills auto-pulled, and any errors

### Requirement 9: Update Remove Command for Full-Path Identities

**User Story:** As a developer, I want `skillops remove` to correctly unlink skills using full-path identities, so that I can cleanly remove skills regardless of their path structure.

#### Acceptance Criteria

1. WHEN removing a skill, THE Remove_Command SHALL extract the symlink name (short name or custom name) for locating the symlink
2. WHEN removing a skill, THE Remove_Command SHALL delete the symlink at `<tool_dir>/skills/<symlink_name>`
3. WHEN removing a skill, THE Remove_Command SHALL remove the full identity and any custom name mapping from local config
4. THE Remove_Command SHALL support both symlink name and full identity for skill selection
5. WHEN a user provides a symlink name that matches multiple skills, THE Remove_Command SHALL launch a TUI to disambiguate, displaying full identities for each match

### Requirement 10: Update Status Command for Full-Path Identities

**User Story:** As a developer, I want `skillops status` to correctly display full-path identities, so that I can see which skills are linked and their sources.

#### Acceptance Criteria

1. WHEN displaying a skill, THE Status_Command SHALL show the symlink name and the full identity
2. WHEN checking if a symlink exists, THE Status_Command SHALL look for the symlink using the symlink name from config
3. THE Status_Command SHALL correctly identify linked vs not-linked status for all skills
4. THE Status_Command SHALL display the registry source for each skill
5. WHEN displaying skills, THE Status_Command SHALL group them by tool (IDE)

### Requirement 11: Update Add TUI for Full-Path Identities

**User Story:** As a developer, I want the `skillops add` TUI to handle full-path identities and automatically configure registries, so that I can interactively add skills with minimal manual configuration.

#### Acceptance Criteria

1. WHEN displaying skills in the selection list, THE Add_TUI SHALL show the symlink name (short name or custom) for each skill
2. WHEN displaying skills, THE Add_TUI SHALL show the full identity as secondary information
3. WHEN a user selects skills, THE Add_TUI SHALL detect conflicts before proceeding to confirmation
4. WHEN conflicts are detected, THE Add_TUI SHALL launch the Conflict_Resolution_TUI
5. THE Conflict_Resolution_TUI SHALL allow users to provide custom symlink names for conflicting skills
6. WHEN confirming additions, THE Add_TUI SHALL display both symlink names and full identities
7. WHEN adding skills, THE Add_TUI SHALL automatically populate registries in local config from skill metadata
8. THE Add_TUI SHALL validate custom symlink names before accepting them

### Requirement 12: Update Discovery Logic for Full-Path Identities

**User Story:** As a developer, I want skillops to discover skills at any depth and construct full-path identities, so that all valid skills are available with proper identification.

#### Acceptance Criteria

1. WHEN discovering skills, THE Skill_Discovery SHALL scan the global store recursively for SKILL.md files
2. WHEN a SKILL.md file is found, THE Skill_Discovery SHALL construct the full identity from the file path
3. THE Skill_Discovery SHALL extract host, owner, repo, and path_in_repo from the global store path
4. THE Skill_Discovery SHALL build skill objects containing the full identity
5. THE Skill_Discovery SHALL handle skills at any nesting depth within repositories

### Requirement 13: Update Pull Command for Full-Path Structure

**User Story:** As a developer, I want `skillops pull` to organize skills in the global store using full-path structure, so that skills from different sources are properly isolated.

#### Acceptance Criteria

1. WHEN pulling a repository, THE Pull_Command SHALL extract host and repo-path from the repository URL
2. THE Pull_Command SHALL create the global store path as `~/.skillops/skills/<host>/<repo-path>` (preserving the full URL path structure including any groups/subgroups)
3. WHEN pulling a full repository, THE Pull_Command SHALL create `.so-repo-meta.json` at the repository root
4. WHEN pulling a specific skill with `--skill` flag, THE Pull_Command SHALL create `.so-skill-meta.json` in the skill directory
5. THE Pull_Command SHALL save `repo_url`, `path_in_repo`, `pulled_at`, and `commit_hash` in metadata
6. THE Pull_Command SHALL support multi-level group URLs (e.g., `https://gitlab.com/group/subgroup/project`)

### Requirement 14: Update Update Command for Metadata-Based Updates

**User Story:** As a developer, I want `skillops update` to use skill metadata to update from the correct source, so that skills remain synchronized with their origins.

#### Acceptance Criteria

1. WHEN updating a skill, THE Update_Command SHALL read `.so-skill-meta.json` to determine the source
2. THE Update_Command SHALL clone the repository from `metadata.repo_url`
3. THE Update_Command SHALL extract the skill from `metadata.path_in_repo`
4. THE Update_Command SHALL update `metadata.pulled_at` and `metadata.commit_hash` after successful update
5. WHEN metadata is missing, THE Update_Command SHALL display an error and skip that skill (no fallback)

### Requirement 15: Validate Full-Path Identities

**User Story:** As a developer, I want skillops to validate full-path identities and provide clear error messages, so that I can quickly identify and fix configuration issues.

#### Acceptance Criteria

1. WHEN a skill identity has fewer than 3 path components, THE Path_Validator SHALL return a descriptive error message
2. WHEN a skill identity contains invalid characters or path traversal attempts ("..", "."), THE Path_Validator SHALL reject the identity
3. WHEN a global skill path does not exist, THE System SHALL provide a clear error indicating the missing skill
4. THE Path_Validator SHALL validate that all components are not empty
5. WHEN validation fails, THE System SHALL NOT create partial symlinks or corrupt the local config

### Requirement 16: Support Local Config Schema V2

**User Story:** As a developer, I want my project config to include registries and custom symlink mappings, so that my team can collaborate without manual configuration.

#### Acceptance Criteria

1. THE Local_Config SHALL use version "2" to indicate the new schema
2. THE Local_Config SHALL include a `registries` array with registry objects
3. THE Local_Config SHALL support optional `symlink_names` mapping for custom symlink names (only stores custom names, not defaults)
4. WHEN writing local config, THE Config_Manager SHALL preserve all fields in human-readable JSON format
5. THE Local_Config SHALL be committable to version control and shareable across team members

## Non-Functional Requirements

### Reliability

1. THE System SHALL handle edge cases including deeply nested paths (20+ levels), paths with special characters, and very long path names
2. THE System SHALL gracefully handle filesystem errors when accessing skill directories
3. THE System SHALL maintain data consistency between local config, global store, and symlinks
4. WHEN network errors occur during pull operations, THE System SHALL provide clear error messages and not corrupt existing data

### Usability

1. THE System SHALL provide clear error messages that include the full skill identity when operations fail
2. THE TUI SHALL display skill information in a way that makes conflicts and sources obvious to users
3. THE Conflict_Resolution_TUI SHALL provide intuitive controls for renaming symlinks
4. THE System SHALL provide helpful suggestions when errors occur (e.g., "add registry to config" when skill cannot be resolved)

### Compatibility

1. THE System SHALL work correctly on Unix-like systems (Linux, macOS) and Windows
2. THE System SHALL handle path separators correctly across different operating systems
3. THE System SHALL support git URLs in HTTPS and SSH formats
4. THE System SHALL support multiple git hosting platforms (GitHub, GitLab, Bitbucket, self-hosted)

## Appendix A: Example Local Config (config.json)

```json
{
  "version": "2",
  "registries": [
    {
      "url": "https://github.com/anthropics/skills",
      "name": "Anthropic Public Skills",
      "priority": 1
    },
    {
      "url": "git@github.com:company-private/enterprise-skills",
      "name": "Company Private Skills (SSH)",
      "priority": 2
    },
    {
      "url": "https://gitlab.com/devops-team/ci-helpers",
      "name": "DevOps Team",
      "priority": 3
    },
    {
      "url": "https://gitlab.company.internal/backend/api-skills",
      "name": "Internal Backend Skills",
      "priority": 4
    },
    {
      "url": "git@bitbucket.org:frontend-guild/react-skills",
      "name": "Frontend Guild",
      "priority": 5
    },
    {
      "url": "https://gitlab.common.datumhq.com/datumhq-consulting-vn/management/datum-skills/software-skills",
      "name": "Datum Software Skills",
      "priority": 6
    }
  ],
  "tools": {
    "kiro": [
      "github.com/anthropics/skills/skills/logger",
      "github.com/anthropics/skills/skills/auth",
      "github.com/company-private/enterprise-skills/api/rate-limiter",
      "gitlab.com/devops-team/ci-helpers/docker-builder",
      "gitlab.common.datumhq.com/datumhq-consulting-vn/management/datum-skills/software-skills/skills/code-review"
    ],
    "cursor": [
      "github.com/anthropics/skills/skills/logger",
      "bitbucket.org/frontend-guild/react-skills/components/form-handler"
    ],
    "windsurf": [
      "gitlab.company.internal/backend/api-skills/database/migrations"
    ]
  },
  "symlink_names": {
    "github.com/company-a/utils/tools/logger": "logger-utils",
    "github.com/company-b/helpers/services/logger": "logger-services"
  }
}
```

## Appendix B: Global Store Structure (~/.skillops/)

```
~/.skillops/
├── config/
│   ├── agentics.yaml              # Global IDE registry
│   └── settings.yaml              # Global registries (optional, fallback)
│
└── skills/                        # Global store (organized by full identity path)
    ├── github.com/
    │   ├── anthropics/
    │   │   └── skills/                          # ← This is the repo root
    │   │       ├── .so-repo-meta.json           # Repo metadata (if full pull)
    │   │       ├── .git/                        # Git repo (if full pull)
    │   │       ├── README.md
    │   │       └── skills/                      # Container folder (path-in-repo starts here)
    │   │           ├── logger/
    │   │           │   ├── SKILL.md
    │   │           │   └── .so-skill-meta.json
    │   │           ├── auth/
    │   │           │   ├── SKILL.md
    │   │           │   └── .so-skill-meta.json
    │   │           └── nested/
    │   │               └── advanced-logger/
    │   │                   ├── SKILL.md
    │   │                   └── .so-skill-meta.json
    │   │
    │   └── company-private/
    │       └── enterprise-skills/               # ← Repo root
    │           └── api/
    │               └── rate-limiter/
    │                   ├── SKILL.md
    │                   └── .so-skill-meta.json
    │
    ├── gitlab.com/
    │   └── devops-team/
    │       └── ci-helpers/                      # ← Repo root
    │           └── docker-builder/
    │               ├── SKILL.md
    │               └── .so-skill-meta.json
    │
    ├── gitlab.common.datumhq.com/
    │   └── datumhq-consulting-vn/
    │       └── management/
    │           └── datum-skills/
    │               └── software-skills/         # ← Repo root (multi-level groups)
    │                   ├── .so-repo-meta.json
    │                   └── skills/
    │                       ├── code-review/
    │                       │   ├── SKILL.md
    │                       │   └── .so-skill-meta.json
    │                       └── logger/
    │                           ├── SKILL.md
    │                           └── .so-skill-meta.json
    │
    └── bitbucket.org/
        └── frontend-guild/
            └── react-skills/                    # ← Repo root
                └── components/
                    └── form-handler/
                        ├── SKILL.md
                        └── .so-skill-meta.json
```

**Note:** The "repo root" is determined by the registry URL. The filesystem structure mirrors the full identity path exactly. There is no special marker in the directory structure to indicate where the repo ends — this is resolved by registry matching at runtime.

## Appendix C: Project Structure with Symlinks

```
my-project/
├── .skillops/
│   └── config.json                # Local config (commit to git)
│       # See Appendix A for full example
│
├── .kiro/
│   └── skills/
│       ├── logger -> ~/.skillops/skills/github.com/anthropics/skills/skills/logger
│       ├── auth -> ~/.skillops/skills/github.com/anthropics/skills/skills/auth
│       ├── rate-limiter -> ~/.skillops/skills/github.com/company-private/enterprise-skills/api/rate-limiter
│       └── docker-builder -> ~/.skillops/skills/gitlab.com/devops-team/ci-helpers/docker-builder
│
├── .cursor/
│   └── skills/
│       ├── logger -> ~/.skillops/skills/github.com/anthropics/skills/skills/logger
│       └── form-handler -> ~/.skillops/skills/bitbucket.org/frontend-guild/react-skills/components/form-handler
│
└── .windsurf/
    └── skills/
        └── migrations -> ~/.skillops/skills/gitlab.company.internal/backend/api-skills/database/migrations
```

## Appendix D: Conflict Resolution Example

### Scenario: Two skills with same short name

**Skills to add:**
- `github.com/company-a/utils/tools/logger`
- `github.com/company-b/helpers/services/logger`

Both have short name: `logger`

### TUI Flow:

```
┌─────────────────────────────────────────────────────────────┐
│  SYMLINK CONFLICT DETECTED                                   │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Multiple skills resolve to the same symlink name: logger   │
│                                                              │
│  Please provide custom names for each skill:                │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ github.com/company-a/utils/tools/logger              │  │
│  │ Symlink name: [logger-utils_______________]          │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ github.com/company-b/helpers/services/logger         │  │
│  │ Symlink name: [logger-services____________]          │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  [Tab] Next field  [Enter] Confirm  [Esc] Cancel            │
└─────────────────────────────────────────────────────────────┘
```

### Result in config.json:

```json
{
  "tools": {
    "kiro": [
      "github.com/company-a/utils/tools/logger",
      "github.com/company-b/helpers/services/logger"
    ]
  },
  "symlink_names": {
    "github.com/company-a/utils/tools/logger": "logger-utils",
    "github.com/company-b/helpers/services/logger": "logger-services"
  }
}
```

### Result symlinks:

```
.kiro/skills/
├── logger-utils -> ~/.skillops/skills/github.com/company-a/utils/tools/logger
└── logger-services -> ~/.skillops/skills/github.com/company-b/helpers/services/logger
```
