# Requirements Document: Nested Skill Paths Support

## Introduction

This feature extends skillops to support nested skill directory structures beyond the current 2-level `repo/skill` format. Currently, skillops assumes all skill identities follow the pattern `repo_name/skill_name`, which causes issues when skill repositories have nested directory structures like `repo/path/to/skill`. The system incorrectly attempts to create nested symlink directories instead of using only the final skill name component.

This enhancement will allow skillops to handle arbitrary nesting depth in skill paths while maintaining backward compatibility with existing 2-level identities.

## Glossary

- **Skill_Identity**: The full path identifier for a skill in the format `repo_name/path/to/skill_name` (e.g., `skills/skills/skill-creator`)
- **Skill_Path**: The portion of the identity after the repository name (e.g., `skills/skill-creator` from `skills/skills/skill-creator`)
- **Short_Name**: The final component of the skill path used as the symlink filename (e.g., `skill-creator` from `skills/skills/skill-creator`)
- **Global_Skill_Path**: The absolute path to a skill in the global store `~/.skillops/skills/repo_name/skill_path`
- **Symlink_Path**: The path where a skill symlink is created in a project's IDE directory (e.g., `.kiro/skills/skill-creator`)
- **Local_Config**: The `.skillops/config.json` file that stores skill identities per tool
- **Symlink_Conflict**: When two different skill identities resolve to the same short name

## Requirements

### Requirement 1: Parse Nested Skill Identities

**User Story:** As a developer, I want skillops to correctly parse skill identities with arbitrary nesting depth, so that I can use skills from repositories with nested directory structures.

#### Acceptance Criteria

1. WHEN a skill identity contains 2 path components (e.g., `repo/skill`), THE Identity_Parser SHALL extract the repository name as the first component and the skill path as the second component
2. WHEN a skill identity contains 3 or more path components (e.g., `repo/path/to/skill`), THE Identity_Parser SHALL extract the repository name as the first component and the remaining components as the skill path
3. THE Identity_Parser SHALL preserve the full skill path for constructing the global skill path
4. THE Identity_Parser SHALL extract the base name (final component) of the skill path as the short name for symlink creation
5. WHEN parsing any valid skill identity, THE Identity_Parser SHALL produce a short name that contains no path separators

### Requirement 2: Create Symlinks Using Short Names

**User Story:** As a developer, I want symlinks to use only the final skill name component, so that my IDE skill directories remain flat and organized.

#### Acceptance Criteria

1. WHEN creating a symlink for a skill with identity `repo/skill`, THE Symlink_Creator SHALL create the symlink at `<tool_dir>/skills/skill`
2. WHEN creating a symlink for a skill with identity `repo/path/to/skill`, THE Symlink_Creator SHALL create the symlink at `<tool_dir>/skills/skill`
3. THE Symlink_Creator SHALL NOT create nested directories within the tool's skills directory
4. WHEN the symlink target is a nested skill path, THE Symlink_Creator SHALL construct the global path as `~/.skillops/skills/repo_name/full/skill/path`
5. THE Symlink_Creator SHALL create symlinks that point to valid skill directories containing SKILL.md files

### Requirement 3: Maintain Backward Compatibility

**User Story:** As an existing skillops user, I want my current 2-level skill identities to continue working without any changes, so that I don't need to modify my configuration.

#### Acceptance Criteria

1. WHEN processing a 2-level skill identity (e.g., `my-repo/logger`), THE System SHALL behave identically to the current implementation
2. WHEN reading existing local config files with 2-level identities, THE System SHALL process them without errors
3. WHEN running `skillops sync` on a project with 2-level identities, THE System SHALL create the same symlinks as before this feature
4. THE System SHALL NOT require users to modify existing `.skillops/config.json` files
5. WHEN displaying skill names in TUI interfaces, THE System SHALL show the short name for both 2-level and nested identities

### Requirement 4: Detect Symlink Conflicts

**User Story:** As a developer, I want to be warned when two different skills would create the same symlink name, so that I can avoid conflicts and understand which skills are incompatible.

#### Acceptance Criteria

1. WHEN two skill identities have different full paths but the same short name (e.g., `repo-a/tools/logger` and `repo-b/utils/logger`), THE Conflict_Detector SHALL identify this as a symlink conflict
2. WHEN a symlink conflict is detected during `add` or `sync`, THE System SHALL display a warning message indicating the conflicting skill identities
3. WHEN a symlink conflict is detected, THE System SHALL skip creating the conflicting symlink and continue processing other skills
4. THE Conflict_Detector SHALL check for conflicts before creating any symlinks
5. WHEN a symlink already exists pointing to a different skill, THE System SHALL treat this as a conflict and skip the operation

### Requirement 5: Update Sync Command

**User Story:** As a developer, I want `skillops sync` to correctly restore symlinks for nested skill paths, so that my project setup is consistent across machines.

#### Acceptance Criteria

1. WHEN `skillops sync` processes a nested skill identity from local config, THE Sync_Command SHALL construct the correct global skill path using the full skill path
2. WHEN `skillops sync` creates a symlink for a nested skill, THE Sync_Command SHALL use only the short name as the symlink filename
3. WHEN a nested skill is not found in the global store, THE Sync_Command SHALL attempt to auto-pull from configured registries (existing behavior)
4. WHEN `skillops sync` completes, THE Sync_Command SHALL report the number of symlinks created including both 2-level and nested skills
5. THE Sync_Command SHALL maintain idempotent behavior (running twice produces the same result)

### Requirement 6: Update Remove Command

**User Story:** As a developer, I want `skillops remove` to correctly unlink nested skill paths, so that I can cleanly remove skills regardless of their nesting depth.

#### Acceptance Criteria

1. WHEN removing a skill with a nested identity, THE Remove_Command SHALL extract the short name for locating the symlink
2. WHEN removing a skill, THE Remove_Command SHALL delete the symlink at `<tool_dir>/skills/<short_name>`
3. WHEN removing a skill, THE Remove_Command SHALL remove the full identity from the local config
4. THE Remove_Command SHALL support both positional argument (short name) and full identity for skill selection
5. WHEN a user provides a short name that matches multiple skills, THE Remove_Command SHALL use the TUI to disambiguate

### Requirement 7: Update Status Command

**User Story:** As a developer, I want `skillops status` to correctly display nested skill paths, so that I can see which skills are linked regardless of their nesting depth.

#### Acceptance Criteria

1. WHEN displaying a nested skill identity, THE Status_Command SHALL show the short name in the skill list
2. WHEN displaying skill details, THE Status_Command SHALL show the repository name in parentheses
3. WHEN checking if a symlink exists, THE Status_Command SHALL look for the symlink at `<tool_dir>/skills/<short_name>`
4. THE Status_Command SHALL correctly identify linked vs not-linked status for nested skills
5. THE Status_Command SHALL display nested skills with the same formatting as 2-level skills

### Requirement 8: Update Add TUI

**User Story:** As a developer, I want the `skillops add` TUI to correctly handle nested skill paths, so that I can interactively add skills with any directory structure.

#### Acceptance Criteria

1. WHEN displaying skills in the selection list, THE Add_TUI SHALL show the short name for each skill
2. WHEN a user selects a nested skill, THE Add_TUI SHALL use the full identity for creating symlinks and updating config
3. WHEN creating symlinks from the TUI, THE Add_TUI SHALL use the short name as the symlink filename
4. THE Add_TUI SHALL detect and display conflicts when multiple skills have the same short name
5. WHEN confirming additions, THE Add_TUI SHALL display the short name in the confirmation summary

### Requirement 9: Preserve Full Identity in Config

**User Story:** As a developer, I want the full skill identity stored in my local config, so that the system can correctly locate skills in the global store.

#### Acceptance Criteria

1. WHEN adding a nested skill to local config, THE Config_Manager SHALL store the complete identity (e.g., `repo/path/to/skill`)
2. WHEN reading identities from local config, THE Config_Manager SHALL preserve the full path structure
3. THE Config_Manager SHALL NOT modify or truncate skill identities during read/write operations
4. WHEN serializing local config to JSON, THE Config_Manager SHALL maintain the exact identity format
5. THE Local_Config SHALL remain human-readable with full skill paths visible

### Requirement 10: Update Discovery Logic

**User Story:** As a developer, I want skillops to discover skills at any nesting depth in pulled repositories, so that all valid skills are available regardless of directory structure.

#### Acceptance Criteria

1. WHEN discovering skills, THE Skill_Discovery SHALL scan repositories recursively for SKILL.md files
2. WHEN a SKILL.md file is found at path `repo/a/b/c/SKILL.md`, THE Skill_Discovery SHALL create an identity `repo/a/b/c`
3. THE Skill_Discovery SHALL construct the full skill path relative to the repository root
4. WHEN building the skill list, THE Skill_Discovery SHALL include the full path in the Skill struct
5. THE Skill_Discovery SHALL maintain compatibility with existing discovery rules (root skills, subfolder skills, container skills)

### Requirement 11: Handle Update Command

**User Story:** As a developer, I want `skillops update` to continue working correctly for repositories containing nested skills, so that I can keep my skills up to date.

#### Acceptance Criteria

1. WHEN updating a repository containing nested skills, THE Update_Command SHALL pull the latest changes for the entire repository
2. THE Update_Command SHALL NOT require special handling for nested vs flat skill structures
3. WHEN updating a specific skill pulled with `--skill` flag, THE Update_Command SHALL use the metadata to locate and update the correct repository
4. THE Update_Command SHALL preserve the skill path structure during updates
5. WHEN an update completes, THE Update_Command SHALL report success without errors related to nested paths

### Requirement 12: Validate Skill Paths

**User Story:** As a developer, I want skillops to validate skill paths and provide clear error messages, so that I can quickly identify and fix configuration issues.

#### Acceptance Criteria

1. WHEN a skill identity is malformed (e.g., missing repository name), THE Path_Validator SHALL return a descriptive error message
2. WHEN a skill path contains invalid characters, THE Path_Validator SHALL reject the identity
3. WHEN a global skill path does not exist, THE System SHALL provide a clear error indicating the missing skill and suggesting `skillops pull`
4. THE Path_Validator SHALL accept skill identities with 2 or more path components
5. WHEN validation fails, THE System SHALL NOT create partial symlinks or corrupt the local config

## Non-Functional Requirements

### Performance

1. THE System SHALL process nested skill identities with no measurable performance degradation compared to 2-level identities
2. THE Skill_Discovery SHALL complete within 2 seconds for repositories containing up to 100 nested skills

### Reliability

1. THE System SHALL handle edge cases including deeply nested paths (10+ levels), paths with special characters, and very long path names
2. THE System SHALL gracefully handle filesystem errors when accessing nested skill directories

### Usability

1. THE System SHALL provide clear error messages that include the full skill identity when operations fail
2. THE TUI SHALL display skill names in a way that makes conflicts obvious to users

### Compatibility

1. THE System SHALL work correctly on Unix-like systems (Linux, macOS) and Windows
2. THE System SHALL handle path separators correctly across different operating systems
