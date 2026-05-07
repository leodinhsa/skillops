# Implementation Tasks: Full-Path Identity Support

## Overview

This document breaks down the implementation of full-path identity support into discrete, testable tasks. Tasks are organized by component and ordered by dependency.

**Total estimated tasks:** 28 tasks across 8 phases

---

## Phase 1: Core Data Structures & Parsing (Foundation)

### Task 1.1: Update ParsedIdentity struct
**File:** `internal/skills/skills.go`
**Requirements:** Req 1
**Description:** 
- Add `ParsedIdentity` struct with fields: `Full`, `Host`, `Path`, `ShortName`
- Add `ParseIdentity(identity string) (*ParsedIdentity, error)` function
- Implement validation: min 3 components, no empty/"."/"..` components
- Note: No `Owner` or `Repo` fields — repo boundary is determined by registry URL prefix matching

**Acceptance:**
- [x] Struct defined with all fields (Full, Host, Path, ShortName)
- [x] ParseIdentity validates component count (minimum 3)
- [x] ParseIdentity validates all components for path traversal
- [x] Returns error for invalid identities

**Tests:**
- Valid 3-component identity (minimum)
- Valid 4+ component identities
- Valid multi-level groups (GitLab deep paths)
- Invalid: < 3 components
- Invalid: empty component
- Invalid: ".." component
- Invalid: "." component

---

### Task 1.2: Implement git.ParseRepoURL
**File:** `internal/git/git.go`
**Requirements:** Req 2, Req 13
**Design:** Algorithm Specification section 8
**Description:**
- Add `ParseRepoURL(repoURL string) (host, repoPath string, error)` function
- Support HTTPS, SSH, self-hosted formats
- Support multi-level groups (full path preserved as repoPath)
- Strip `.git` suffix
- Validate all components
- Return normalized identity prefix: `host + "/" + repoPath`

**Acceptance:**
- [x] Parses HTTPS URLs correctly
- [x] Parses SSH URLs correctly
- [x] Handles multi-level groups (GitLab) — full path preserved
- [x] Strips `.git` suffix
- [x] Validates components
- [x] Returns identity prefix usable for registry matching

**Tests:**
- `https://github.com/anthropics/skills.git` → host=`github.com`, repoPath=`anthropics/skills`
- `git@github.com:owner/repo.git` → host=`github.com`, repoPath=`owner/repo`
- `https://gitlab.com/group/subgroup/project` → host=`gitlab.com`, repoPath=`group/subgroup/project`
- `https://gitlab.common.datumhq.com/datumhq-consulting-vn/management/datum-skills/software-skills` → host=`gitlab.common.datumhq.com`, repoPath=`datumhq-consulting-vn/management/datum-skills/software-skills`
- Invalid: missing path after host
- Invalid: path traversal in components

---

### Task 1.3: Update LocalConfig schema to V2
**File:** `internal/config/localconfig.go`
**Requirements:** Req 3, Req 16
**Description:**
- Add `Version` field (always "2")
- Add `Registries []Registry` field
- Add `SymlinkNames map[string]string` field (optional)
- Update `ReadLocalConfig` to validate version
- Update `WriteLocalConfig` to always set version "2"

**Acceptance:**
- [x] LocalConfig struct has all V2 fields
- [x] ReadLocalConfig fails on V1 config with clear error
- [x] WriteLocalConfig always sets version "2"
- [x] SymlinkNames is optional (omitempty)

**Tests:**
- Read valid V2 config
- Read V1 config (should error)
- Read config with no version field (should error)
- Write config sets version "2"
- SymlinkNames omitted when empty

---

## Phase 2: Global Store & Metadata

### Task 2.1: Update global store structure
**File:** `cmd/pull.go`
**Requirements:** Req 2, Req 13
**Description:**
- Use `ParseRepoURL` to extract host/owner/repo
- Create directory: `~/.skillops/skills/<host>/<owner>/<repo>`
- Handle multi-level owners (create nested directories)

**Acceptance:**
- [ ] Pull creates correct directory structure
- [ ] Multi-level owners create nested dirs
- [ ] Existing directories not overwritten

**Tests:**
- Pull `github.com/owner/repo` creates correct path
- Pull `gitlab.com/group/subgroup/project` creates nested path
- Pull same repo twice is idempotent

---

### Task 2.2: Implement SkillMetadata
**File:** `internal/skills/metadata.go` (NEW)
**Requirements:** Req 5
**Description:**
- Create `SkillMetadata` struct: `RepoURL`, `PathInRepo`, `PulledAt`, `CommitHash`
- Implement `SaveSkillMetadata(skillPath, metadata)`
- Implement `LoadSkillMetadata(skillPath) -> metadata`
- Implement `HasMetadata(skillPath) -> bool`
- Metadata file: `.so-skill-meta.json`

**Acceptance:**
- [ ] Struct defined with all fields
- [ ] SaveSkillMetadata writes JSON with formatting
- [ ] LoadSkillMetadata reads and parses JSON
- [ ] HasMetadata checks file existence

**Tests:**
- Save and load metadata
- Load missing metadata returns error
- HasMetadata returns true/false correctly
- JSON is human-readable (indented)

---

### Task 2.3: Implement RepoMetadata
**File:** `internal/skills/metadata.go`
**Requirements:** Req 13
**Description:**
- Create `RepoMetadata` struct: `RepoURL`, `PulledAt`, `CommitHash`
- Implement `SaveRepoMetadata(repoPath, metadata)`
- Implement `LoadRepoMetadata(repoPath) -> metadata`
- Metadata file: `.so-repo-meta.json`

**Acceptance:**
- [ ] Struct defined with all fields
- [ ] SaveRepoMetadata writes JSON
- [ ] LoadRepoMetadata reads JSON

**Tests:**
- Save and load repo metadata
- Load missing metadata returns error

---

### Task 2.4: Implement PullSkillFromURL
**File:** `internal/skills/extract.go`
**Requirements:** Req 6, Req 13
**Design:** Algorithm Specification section 7
**Description:**
- Implement `PullSkillFromURL(repoURL, pathInRepo, destSkillDir) -> error`
- Shallow clone (`--depth 1`)
- Extract skill from pathInRepo
- Atomic copy (temp-then-rename)
- Save skill metadata
- Cleanup temp directories

**Acceptance:**
- [ ] Clones repository to temp
- [ ] Extracts skill from pathInRepo
- [ ] Copies atomically (temp-then-rename)
- [ ] Saves metadata with commit hash
- [ ] Cleans up temp on success and error

**Tests:**
- Pull skill from valid repo
- Pull skill with nested path
- Pull skill not found in repo (error)
- Pull with network error (cleanup temp)
- Metadata saved correctly

---

## Phase 3: Registry Matching

### Task 3.1: Implement Registry struct
**File:** `internal/config/localconfig.go`
**Requirements:** Req 3
**Description:**
- Create `Registry` struct: `URL`, `Name`, `Priority`
- Add validation for Registry.URL (no trailing slash)

**Acceptance:**
- [ ] Struct defined with all fields
- [ ] URL validation implemented

**Tests:**
- Valid registry URL
- Invalid: trailing slash

---

### Task 3.2: Implement RegistryMatcher
**File:** `internal/config/registry.go` (NEW)
**Requirements:** Req 3, Req 8
**Design:** Algorithm Specification section 2
**Description:**
- Implement `MatchRegistry(skillIdentity, registries) -> (cloneURL, pathInRepo, error)`
- Implement `NormalizeRegistryURL(registryURL) -> identityPrefix`
- Normalize: strip protocol (`https://`, `git@`), replace `:` with `/` for SSH, strip trailing `/`
- Match: skill identity starts with normalized registry prefix
- Path in repo = identity minus matched prefix (strip leading `/`)
- Sort registries by priority

**Acceptance:**
- [ ] MatchRegistry finds correct registry by prefix matching
- [ ] NormalizeRegistryURL handles HTTPS and SSH formats
- [ ] Respects priority order
- [ ] Returns error when no match
- [ ] Correctly extracts pathInRepo as remainder after prefix
- [ ] Handles multi-level group URLs correctly

**Tests:**
- Match with single registry (HTTPS)
- Match with single registry (SSH)
- Match with multiple registries (priority)
- No match returns error
- Multi-level group URL matching
- Path in repo correctly extracted
- Substring false positive prevented (e.g., `github.com/anthropics/skills-extra` should NOT match registry `https://github.com/anthropics/skills`)

---

## Phase 4: Symlink Management

### Task 4.1: Update CreateSkillSymlink
**File:** `internal/symlink/symlink.go`
**Requirements:** Req 4, Req 8
**Design:** Algorithm Specification section 3
**Description:**
- Update signature: `CreateSkillSymlink(identity, tool, localConfig) -> (wasCreated bool, error)`
- Use custom symlink name from config if exists
- Return `wasCreated=false` for idempotent no-ops
- Construct global path with full identity

**Acceptance:**
- [ ] Uses custom name from config.SymlinkNames
- [ ] Falls back to short name
- [ ] Returns wasCreated=true only when created
- [ ] Returns wasCreated=false when already exists
- [ ] Validates global path exists

**Tests:**
- Create with default short name
- Create with custom name
- Idempotent (second call returns wasCreated=false)
- Conflict detection
- Global path not found (error)

---

### Task 4.2: Implement conflict detection
**File:** `internal/tui/add.go`
**Requirements:** Req 7
**Design:** Algorithm Specification section 4
**Description:**
- Implement `DetectConflicts(identities, localConfig) -> []Conflict`
- Check for duplicate symlink names (considering custom names)
- Return list of conflicts with identities

**Acceptance:**
- [ ] Detects conflicts between default short names
- [ ] Considers existing custom names
- [ ] Returns all conflicts

**Tests:**
- No conflicts
- Two skills same short name
- Three skills same short name
- Custom name prevents conflict

---

### Task 4.3: Implement Conflict Resolution TUI
**File:** `internal/tui/conflict.go` (NEW)
**Requirements:** Req 7
**Design:** Algorithm Specification section 4
**Description:**
- Create `ConflictResolutionModel` (bubbletea)
- Display conflicting identities
- Input fields for custom names
- Validate custom names (no path separators, not empty/"."/"..")
- Return map of identity -> custom name

**Acceptance:**
- [ ] TUI displays all conflicts
- [ ] Input fields for each skill
- [ ] Real-time validation
- [ ] Navigation between fields
- [ ] Returns custom names map

**Tests:**
- Manual TUI testing (no unit tests for bubbletea)
- Validation prevents invalid names

---

### Task 4.4: Handle non-TTY conflict detection
**File:** `internal/tui/add.go`
**Requirements:** Req 7 AC 3
**Description:**
- Detect non-TTY environment
- When conflicts detected in non-TTY, fail with descriptive error
- List all conflicting identities in error message
- Suggest manual resolution in config.json

**Acceptance:**
- [ ] Detects non-TTY (check `os.Stdin.Fd()` and `term.IsTerminal()`)
- [ ] Fails with clear error message
- [ ] Lists all conflicts
- [ ] Suggests manual fix

**Tests:**
- TTY environment launches TUI
- Non-TTY environment returns error
- Error message contains all conflicts

---

### Task 4.5: Implement Remove Disambiguation TUI
**File:** `internal/tui/remove.go`
**Requirements:** Req 9 AC 5
**Design:** Component Specifications section 4
**Description:**
- Create disambiguation TUI for when symlink name matches multiple skills
- Display full identities for each matching skill
- Allow user to select which skill to remove
- Use bubbletea list component for selection

**Acceptance:**
- [ ] TUI displays all matching skills with full identities
- [ ] User can navigate with arrow keys
- [ ] User can select with space/enter
- [ ] Returns selected skill identity
- [ ] Handles ESC to cancel

**Tests:**
- Manual TUI testing (no unit tests for bubbletea)
- Two skills with same symlink name
- Three+ skills with same symlink name
- Cancel operation works

---

## Phase 5: Command Updates

### Task 5.1: Update sync command
**File:** `cmd/sync.go`
**Requirements:** Req 8
**Design:** Algorithm Specification section 5
**Description:**
- Parse full-path identities
- Use MatchRegistry for missing skills
- Use custom symlink names from config
- Track wasCreated for accurate reporting
- Error (not fallback) when no registry matches

**Acceptance:**
- [ ] Parses full-path identities
- [ ] Uses registries for auto-pull
- [ ] Uses custom symlink names
- [ ] Reports accurate counts (created, auto-pulled)
- [ ] Errors when no registry matches

**Tests:**
- Sync with existing skills
- Sync with missing skills (auto-pull)
- Sync with custom symlink names
- Sync with no registry match (error)
- Idempotent (second run reports 0 created)

---

### Task 5.2: Update add command
**File:** `cmd/add.go`, `internal/tui/add.go`
**Requirements:** Req 11
**Description:**
- Display full identities in TUI
- Detect conflicts before confirmation
- Launch conflict resolution TUI
- Auto-populate registries from metadata
- Save custom symlink names to config

**Acceptance:**
- [ ] TUI shows full identities
- [ ] Detects conflicts
- [ ] Launches conflict TUI when needed
- [ ] Reads metadata to get registry URL
- [ ] Adds registry to config if not exists
- [ ] Saves custom names to config.SymlinkNames

**Tests:**
- Add skill without conflict
- Add skills with conflict (TUI launched)
- Registry auto-populated
- Custom names saved

---

### Task 5.3: Update remove command
**File:** `cmd/remove.go`, `internal/tui/remove.go`
**Requirements:** Req 9
**Dependencies:** Task 4.5 (Remove Disambiguation TUI)
**Description:**
- Support symlink name or full identity
- Launch disambiguation TUI for multi-match (uses Task 4.5)
- Remove from config.Tools and config.SymlinkNames

**Acceptance:**
- [ ] Accepts symlink name
- [ ] Accepts full identity
- [ ] Disambiguates multi-match with TUI
- [ ] Removes from both Tools and SymlinkNames

**Tests:**
- Remove by symlink name (unique)
- Remove by full identity
- Remove by symlink name (multi-match, TUI)
- Custom name removed from config

---

### Task 5.4: Update status command
**File:** `cmd/status.go`
**Requirements:** Req 10
**Description:**
- Display symlink name and full identity
- Check symlink using symlink name from config
- Show registry source
- Group by tool

**Acceptance:**
- [ ] Shows symlink name
- [ ] Shows full identity
- [ ] Shows linked/not-linked status
- [ ] Groups by tool

**Tests:**
- Status with default names
- Status with custom names
- Status shows correct link status

---

### Task 5.5: Update pull command
**File:** `cmd/pull.go`
**Requirements:** Req 13
**Description:**
- Use ParseRepoURL for host/owner/repo
- Create nested directory structure
- For `--skill` flag, use PullSkillFromURL
- For full pull, save RepoMetadata

**Acceptance:**
- [ ] Creates correct directory structure
- [ ] --skill uses PullSkillFromURL
- [ ] Full pull saves RepoMetadata
- [ ] Multi-level groups supported

**Tests:**
- Pull full repo
- Pull specific skill
- Pull with multi-level group
- Pull creates correct paths

---

### Task 5.6: Update update command
**File:** `cmd/update.go`
**Requirements:** Req 14
**Design:** Algorithm Specification section 6
**Description:**
- Read skill metadata
- Clone from metadata.RepoURL
- Extract from metadata.PathInRepo
- Update metadata after pull

**Acceptance:**
- [ ] Reads metadata
- [ ] Clones from correct URL
- [ ] Extracts from correct path
- [ ] Updates metadata timestamps
- [ ] Errors when metadata missing

**Tests:**
- Update skill with metadata
- Update skill without metadata (error)
- Metadata updated after pull

---

### Task 5.7: Implement Config V1 Detection
**File:** `internal/config/localconfig.go`
**Requirements:** Design - Migration and Compatibility section
**Dependencies:** Task 1.3 (LocalConfig V2)
**Description:**
- Detect config with `"version": "1"` or missing version field
- Fail with clear error message
- Suggest running `skillops init` to migrate

**Acceptance:**
- [ ] Detects version "1" in config
- [ ] Detects missing version field
- [ ] Returns error with migration instructions
- [ ] Error message includes: "Config version 1 detected. This version requires config v2. Please run: skillops init"

**Tests:**
- Read config with version "1" (error)
- Read config with no version field (error)
- Read config with version "2" (success)
- Error message contains migration instructions

---

### Task 5.8: Auto-populate Registries in Add Command
**File:** `cmd/add.go`, `internal/config/localconfig.go`
**Requirements:** Req 11 AC 7, Req 3 AC 2
**Dependencies:** Task 2.2 (SkillMetadata), Task 3.1 (Registry struct)
**Description:**
- When adding skills, read `.so-skill-meta.json` from each skill
- Extract `repo_url` from metadata
- Parse URL to get host/owner (use `git.ParseRepoURL`)
- Check if registry already exists in config
- If not exists, add new registry with auto-generated name and priority
- Deduplicate registries (same URL)

**Acceptance:**
- [ ] Reads metadata from added skills
- [ ] Extracts repo_url
- [ ] Parses URL to get host/owner
- [ ] Checks for existing registry
- [ ] Adds new registry if not exists
- [ ] Generates registry name (e.g., "GitHub - anthropics")
- [ ] Assigns priority (max existing priority + 1)
- [ ] Deduplicates by URL

**Tests:**
- Add skill with new registry (registry added)
- Add skill with existing registry (no duplicate)
- Add multiple skills from same repo (one registry)
- Add skills from different repos (multiple registries)
- Registry name generation
- Priority assignment

---

## Phase 6: Discovery

### Task 6.1: Update discovery logic
**File:** `internal/skills/skills.go`
**Requirements:** Req 12
**Description:**
- Walk global store recursively
- Skip hidden directories (starting with ".")
- Construct full-path identities from filesystem
- Extract host/owner/repo from path

**Acceptance:**
- [ ] Walks global store
- [ ] Skips hidden directories
- [ ] Constructs correct identities
- [ ] Handles multi-level owners

**Tests:**
- Discover skills in flat structure
- Discover skills in nested structure
- Skip .git directories
- Skip .so-* files
- Multi-level owner paths

---

## Phase 7: Validation & Error Handling

### Task 7.1: Implement path validation
**File:** `internal/utils/utils.go`
**Requirements:** Req 15
**Description:**
- Validate skill identities (min 4 components)
- Validate no path traversal
- Validate components not empty
- Clear error messages

**Acceptance:**
- [ ] Validates component count
- [ ] Validates all components
- [ ] Returns descriptive errors

**Tests:**
- Valid identity passes
- < 4 components fails
- Empty component fails
- ".." component fails
- "." component fails

---

### Task 7.2: Add error messages
**Files:** All command files
**Requirements:** All requirements (error handling)
**Description:**
- Add clear error messages with full identity
- Suggest recovery actions
- Handle non-TTY environments

**Acceptance:**
- [ ] Errors include full identity
- [ ] Errors suggest actions
- [ ] Non-TTY errors are clear

---

### Task 7.3: Standardize Non-TTY Error Messages
**Files:** `cmd/add.go`, `cmd/remove.go`, `internal/tui/add.go`, `internal/tui/remove.go`
**Requirements:** Req 7 AC 3
**Dependencies:** Task 4.4 (Non-TTY conflict detection)
**Description:**
- Create standard error message format for non-TTY conflicts
- Include all conflicting identities
- Provide clear manual resolution steps
- Show example config.json snippet

**Acceptance:**
- [ ] Error message format is consistent across commands
- [ ] Lists all conflicting skills with full identities
- [ ] Provides step-by-step resolution instructions
- [ ] Shows example of adding to `symlink_names` in config.json
- [ ] Error is actionable (user knows exactly what to do)

**Example error format:**
```
Error: Symlink conflicts detected (non-interactive mode)

The following skills have conflicting symlink names:

Symlink name: logger
  - github.com/company-a/utils/tools/logger
  - github.com/company-b/helpers/services/logger

To resolve, add custom symlink names to .skillops/config.json:

{
  "symlink_names": {
    "github.com/company-a/utils/tools/logger": "logger-utils",
    "github.com/company-b/helpers/services/logger": "logger-services"
  }
}

Then run: skillops sync
```

**Tests:**
- Add command in non-TTY with conflicts
- Remove command in non-TTY with multi-match
- Error message contains all required elements
- Error message is parseable by CI tools

---

## Phase 8: Testing & Documentation

### Task 8.1: Integration tests
**File:** `cmd/*_test.go`
**Description:**
- End-to-end workflow tests
- Pull → Add → Sync
- Conflict resolution
- Multi-registry
- Update with metadata

**Tests:**
- Full workflow on new machine
- Conflict resolution flow
- Multi-registry priority
- Update flow

---

### Task 8.2: Update documentation
**Files:** `README.md`, `DOC_GUIDE.md`
**Description:**
- Update examples to use full-path identities
- Document registry configuration
- Document conflict resolution
- Update command help text

**Acceptance:**
- [ ] All examples use full-path format
- [ ] Registry docs added
- [ ] Conflict resolution docs added
- [ ] Help text updated

---

## Task Dependencies

```
Phase 1 (Foundation)
├─ 1.1 ParsedIdentity
├─ 1.2 ParseRepoURL
└─ 1.3 LocalConfig V2

Phase 2 (Storage)
├─ 2.1 Global store (depends on 1.2)
├─ 2.2 SkillMetadata
├─ 2.3 RepoMetadata
└─ 2.4 PullSkillFromURL (depends on 2.2)

Phase 3 (Registry)
├─ 3.1 Registry struct (depends on 1.3)
└─ 3.2 RegistryMatcher (depends on 3.1, 1.1)

Phase 4 (Symlinks)
├─ 4.1 CreateSkillSymlink (depends on 1.1, 1.3)
├─ 4.2 Conflict detection (depends on 1.1, 1.3)
├─ 4.3 Conflict TUI (depends on 4.2)
├─ 4.4 Non-TTY handling (depends on 4.2)
└─ 4.5 Remove Disambiguation TUI (depends on 1.1)

Phase 5 (Commands)
├─ 5.1 Sync (depends on 3.2, 4.1, 2.4)
├─ 5.2 Add (depends on 4.2, 4.3, 2.2)
├─ 5.3 Remove (depends on 4.1, 4.5)
├─ 5.4 Status (depends on 4.1)
├─ 5.5 Pull (depends on 1.2, 2.4, 2.3)
├─ 5.6 Update (depends on 2.2)
├─ 5.7 Config V1 Detection (depends on 1.3)
└─ 5.8 Auto-populate Registries (depends on 2.2, 3.1)

Phase 6 (Discovery)
└─ 6.1 Discovery (depends on 1.1)

Phase 7 (Validation)
├─ 7.1 Path validation (depends on 1.1)
├─ 7.2 Error messages (depends on all)
└─ 7.3 Non-TTY error messages (depends on 4.4)

Phase 8 (Testing)
├─ 8.1 Integration tests (depends on all)
└─ 8.2 Documentation (depends on all)
```

## Estimated Effort

| Phase | Tasks | Estimated Time |
|-------|-------|----------------|
| Phase 1 | 3 | 1-2 days |
| Phase 2 | 4 | 2-3 days |
| Phase 3 | 2 | 1 day |
| Phase 4 | 5 | 2.5-3.5 days |
| Phase 5 | 8 | 4-5 days |
| Phase 6 | 1 | 0.5 day |
| Phase 7 | 3 | 1.5-2 days |
| Phase 8 | 2 | 1-2 days |
| **Total** | **28** | **14-20 days** |

## Success Criteria

- [ ] All 16 requirements have passing tests
- [ ] All algorithms from design.md are implemented
- [ ] Integration tests pass
- [ ] Documentation updated
- [ ] No backward compatibility (clean break)
- [ ] Config V2 enforced
- [ ] Multi-level groups supported
- [ ] Conflict resolution works in TTY and non-TTY
