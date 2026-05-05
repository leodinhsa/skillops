# Design Document: Nested Skill Paths Support

## Overview

This design extends skillops to support arbitrary nesting depth in skill directory structures while maintaining backward compatibility with the existing 2-level `repo/skill` format. The core change is introducing a consistent identity parsing strategy that extracts the **short name** (final path component) for symlink creation while preserving the **full skill path** for global store lookups.

### Current Limitation

The existing implementation assumes all skill identities follow the pattern `repo_name/skill_name` and uses a simple `strings.SplitN(identity, "/", 2)` approach. When encountering nested paths like `skills/skills/skill-creator`, the system incorrectly attempts to create nested symlink directories (`skills/skill-creator`) instead of using only the final component (`skill-creator`).

### Solution Approach

We introduce a **filepath.Base()** extraction pattern applied consistently across all commands that handle skill identities. This ensures:

1. **Symlinks remain flat**: Only the final path component becomes the symlink filename
2. **Full paths preserved**: The complete identity is stored in local config for accurate global store lookups
3. **Backward compatibility**: 2-level identities continue to work identically
4. **Conflict detection**: Multiple skills with the same short name are detected and handled gracefully

## Architecture

### High-Level Data Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    Skill Identity Flow                       │
└─────────────────────────────────────────────────────────────┘

Input: "skills/skills/skill-creator"
   │
   ├─> Identity Parser
   │   ├─> Repository: "skills" (first component)
   │   ├─> Skill Path: "skills/skill-creator" (remaining components)
   │   └─> Short Name: "skill-creator" (filepath.Base of skill path)
   │
   ├─> Global Store Path Construction
   │   └─> ~/.skillops/skills/skills/skills/skill-creator
   │
   ├─> Symlink Path Construction
   │   └─> <project>/.kiro/skills/skill-creator
   │
   └─> Local Config Storage
       └─> "skills/skills/skill-creator" (full identity preserved)


### System Components

```
┌──────────────────────────────────────────────────────────────┐
│                     Component Architecture                    │
└──────────────────────────────────────────────────────────────┘

┌─────────────────┐
│  CLI Commands   │  (cmd/*.go)
│  - sync.go      │  Uses filepath.Base() for symlink creation
│  - remove.go    │  Uses filepath.Base() for symlink lookup
│  - status.go    │  Uses filepath.Base() for symlink checking
│  - add.go       │  Delegates to TUI
└────────┬────────┘
         │
         ├─────────────────────────────────────────┐
         │                                         │
         ▼                                         ▼
┌─────────────────┐                      ┌─────────────────┐
│   TUI Layer     │                      │  Config Layer   │
│ (internal/tui)  │                      │(internal/config)│
│  - add.go       │◄────────────────────►│ localconfig.go  │
│  - remove.go    │  Reads/writes full   │                 │
│                 │  skill identities    │ Stores full     │
│ Uses Base() for │                      │ identities in   │
│ display & links │                      │ config.json     │
└────────┬────────┘                      └─────────────────┘
         │
         ▼
┌─────────────────┐
│ Discovery Layer │
│(internal/skills)│
│  - skills.go    │  Discovers skills recursively
│                 │  Builds full path identities
│  - extract.go   │  Handles skill extraction
└─────────────────┘
```

## Components and Interfaces

### 1. Identity Parser (Inline Logic)

**Location**: Applied consistently across `cmd/sync.go`, `cmd/remove.go`, `cmd/status.go`, `internal/tui/add.go`

**Algorithm**:
```go
// Parse skill identity into components
identity := "skills/skills/skill-creator"  // Full identity from config

// Split into repo and skill path
parts := strings.SplitN(identity, "/", 2)
if len(parts) != 2 {
    return error("invalid skill identity")
}

repoName := parts[0]      // "skills"
skillPath := parts[1]     // "skills/skill-creator"

// Extract short name for symlink
shortName := filepath.Base(skillPath)  // "skill-creator"

// Construct global path
globalPath := filepath.Join(config.SkillsDir, repoName, skillPath)
// Result: ~/.skillops/skills/skills/skills/skill-creator

// Construct symlink path
symlinkPath := filepath.Join(toolDir, shortName)
// Result: <project>/.kiro/skills/skill-creator
```

**Key Change**: Replace `shortName := parts[1]` with `shortName := filepath.Base(parts[1])`

### 2. Sync Command (cmd/sync.go)

**Current Implementation Issue**:
```go
// Line 67-68 (BEFORE)
repoName := parts[0]
skillName := parts[1]  // ❌ For "skills/skills/skill-creator", this is "skills/skill-creator"

globalSkillPath := filepath.Join(config.SkillsDir, repoName, skillName)
symlinkPath := filepath.Join(skillsDir, skillName)  // ❌ Creates nested directory
```

**Fixed Implementation**:
```go
// Line 67-70 (AFTER)
repoName := parts[0]
skillPath := parts[1]  // Full path: "skills/skill-creator"
shortName := filepath.Base(skillPath)  // ✓ Extract final component: "skill-creator"

globalSkillPath := filepath.Join(config.SkillsDir, repoName, skillPath)  // ✓ Correct global path
symlinkPath := filepath.Join(skillsDir, shortName)  // ✓ Flat symlink
```

**Before/After Example**:

```go
// BEFORE: Identity "skills/skills/skill-creator"
repoName := "skills"
skillName := "skills/skill-creator"
globalSkillPath := "~/.skillops/skills/skills/skills/skill-creator"  // ✓ Correct
symlinkPath := ".kiro/skills/skills/skill-creator"  // ❌ Wrong - nested!

// AFTER: Identity "skills/skills/skill-creator"
repoName := "skills"
skillPath := "skills/skill-creator"
shortName := "skill-creator"  // ✓ filepath.Base(skillPath)
globalSkillPath := "~/.skillops/skills/skills/skills/skill-creator"  // ✓ Correct
symlinkPath := ".kiro/skills/skill-creator"  // ✓ Correct - flat!
```

### 3. Remove Command (cmd/remove.go)

**Current Implementation Issue**:
```go
// Line 52-58 (BEFORE)
parts := strings.SplitN(identity, "/", 2)
shortName := parts[1]  // ❌ For nested paths, this includes intermediate directories
```

**Fixed Implementation**:
```go
// Line 52-59 (AFTER)
parts := strings.SplitN(identity, "/", 2)
if len(parts) != 2 {
    return error("invalid skill identity")
}
shortName := filepath.Base(parts[1])  // ✓ Extract final component only
```

**Impact**: The `UnlinkSkillFromTool` function in `internal/tui/remove.go` receives the correct short name for symlink deletion.

### 4. Status Command (cmd/status.go)

**Current Implementation Issues** (2 locations):

**Location 1: Line 54-60 (Tool not in global config path)**
```go
// BEFORE
parts := strings.SplitN(identity, "/", 2)
shortName := identity
repoName := ""
if len(parts) == 2 {
    repoName = parts[0]
    shortName = parts[1]  // ❌ Nested path issue
}
```

**Location 2: Line 72-78 (Tool in global config path)**
```go
// BEFORE
parts := strings.SplitN(identity, "/", 2)
shortName := identity
repoName := ""
if len(parts) == 2 {
    repoName = parts[0]
    shortName = parts[1]  // ❌ Nested path issue
}
```

**Fixed Implementation** (both locations):
```go
// AFTER
parts := strings.SplitN(identity, "/", 2)
shortName := identity
repoName := ""
if len(parts) == 2 {
    repoName = parts[0]
    shortName = filepath.Base(parts[1])  // ✓ Extract final component
}
```

### 5. Add TUI (internal/tui/add.go)

**Current Implementation Issues** (2 locations):

**Location 1: Line 267-273 (viewSkillSelect display)**
```go
// BEFORE
shortName := strings.SplitN(r.item.identity, "/", 2)
displayName := r.item.identity
if len(shortName) == 2 {
    displayName = shortName[1]  // ❌ Shows "skills/skill-creator" instead of "skill-creator"
}
```

**Location 2: Line 323-328 (viewConfirm display)**
```go
// BEFORE
shortName := strings.SplitN(skill.identity, "/", 2)
name := skill.identity
if len(shortName) == 2 {
    name = shortName[1]  // ❌ Shows "skills/skill-creator" instead of "skill-creator"
}
```

**Fixed Implementation** (both locations):
```go
// AFTER
parts := strings.SplitN(r.item.identity, "/", 2)
displayName := r.item.identity
if len(parts) == 2 {
    displayName = filepath.Base(parts[1])  // ✓ Extract final component for display
}
```

**Location 3: Line 356-362 (LinkSkillToTool function)**
```go
// BEFORE
parts := strings.SplitN(identity, "/", 2)
if len(parts) != 2 {
    return "", fmt.Errorf("invalid skill identity: %s", identity)
}
shortName := parts[1]  // ❌ Nested path issue
```

**Fixed Implementation**:
```go
// AFTER
parts := strings.SplitN(identity, "/", 2)
if len(parts) != 2 {
    return "", fmt.Errorf("invalid skill identity: %s", identity)
}
shortName := filepath.Base(parts[1])  // ✓ Extract final component
```

## Data Models

### Skill Identity Structure

```go
// Full identity stored in local config
type SkillIdentity string  // Format: "repo_name/path/to/skill"

// Parsed components
type ParsedIdentity struct {
    Full      string  // "skills/skills/skill-creator"
    Repo      string  // "skills"
    SkillPath string  // "skills/skill-creator"
    ShortName string  // "skill-creator" (filepath.Base of SkillPath)
}

// Example parsing function (conceptual - implemented inline)
func ParseIdentity(identity string) (ParsedIdentity, error) {
    parts := strings.SplitN(identity, "/", 2)
    if len(parts) != 2 {
        return ParsedIdentity{}, fmt.Errorf("invalid identity: %s", identity)
    }
    
    return ParsedIdentity{
        Full:      identity,
        Repo:      parts[0],
        SkillPath: parts[1],
        ShortName: filepath.Base(parts[1]),
    }, nil
}
```

### Local Config Schema (Unchanged)

```json
{
  "version": "1",
  "tools": {
    "kiro": [
      "skills/skills/skill-creator",
      "my-repo/logger"
    ]
  }
}
```

**Key Point**: Full identities are preserved in config. Short name extraction happens at runtime.

### Symlink Structure

```
<project>/
  .kiro/
    skills/
      skill-creator → ~/.skillops/skills/skills/skills/skill-creator
      logger → ~/.skillops/skills/my-repo/logger
```

**Key Point**: Symlink directory remains flat regardless of nesting depth.

## Algorithm Specifications

### Identity Parsing Algorithm

```
FUNCTION ParseSkillIdentity(identity: string) -> (repo, skillPath, shortName, error)
  INPUT: Full skill identity (e.g., "skills/skills/skill-creator")
  OUTPUT: Repository name, skill path, short name, or error
  
  STEPS:
    1. Split identity on "/" with limit 2
       parts = strings.SplitN(identity, "/", 2)
    
    2. Validate split result
       IF len(parts) != 2 THEN
         RETURN error("invalid skill identity: must contain repo/path")
       END IF
    
    3. Extract components
       repo = parts[0]
       skillPath = parts[1]
       shortName = filepath.Base(skillPath)
    
    4. Validate short name
       IF shortName == "" OR shortName == "." OR shortName == ".." THEN
         RETURN error("invalid skill path")
       END IF
    
    5. RETURN (repo, skillPath, shortName, nil)
END FUNCTION
```

### Symlink Creation Algorithm

```
FUNCTION CreateSkillSymlink(identity, globalSkillsDir, projectToolDir) -> error
  INPUT: 
    - identity: Full skill identity (e.g., "skills/skills/skill-creator")
    - globalSkillsDir: ~/.skillops/skills/
    - projectToolDir: <project>/.kiro/skills/
  
  OUTPUT: Error if creation fails, nil on success
  
  STEPS:
    1. Parse identity
       (repo, skillPath, shortName, err) = ParseSkillIdentity(identity)
       IF err != nil THEN RETURN err
    
    2. Construct global skill path
       globalPath = filepath.Join(globalSkillsDir, repo, skillPath)
       // Example: ~/.skillops/skills/skills/skills/skill-creator
    
    3. Verify global skill exists
       IF NOT exists(globalPath + "/SKILL.md") THEN
         RETURN error("skill not found in global store")
       END IF
    
    4. Construct symlink path
       symlinkPath = filepath.Join(projectToolDir, shortName)
       // Example: <project>/.kiro/skills/skill-creator
    
    5. Check for conflicts
       IF exists(symlinkPath) THEN
         existingTarget = readlink(symlinkPath)
         IF existingTarget != globalPath THEN
           RETURN error("conflict: symlink already exists pointing to different skill")
         ELSE
           RETURN nil  // Already linked correctly (idempotent)
         END IF
       END IF
    
    6. Create symlink
       err = os.Symlink(globalPath, symlinkPath)
       IF err != nil THEN
         RETURN error("failed to create symlink: " + err)
       END IF
    
    7. RETURN nil
END FUNCTION
```

### Conflict Detection Algorithm

```
FUNCTION DetectSymlinkConflicts(identities: []string) -> []Conflict
  INPUT: List of skill identities to be linked
  OUTPUT: List of conflicts where multiple identities map to same short name
  
  STEPS:
    1. Initialize short name map
       shortNameMap = map[string][]string{}
    
    2. Build short name mapping
       FOR EACH identity IN identities DO
         (_, _, shortName, err) = ParseSkillIdentity(identity)
         IF err != nil THEN CONTINUE
         
         shortNameMap[shortName] = append(shortNameMap[shortName], identity)
       END FOR
    
    3. Identify conflicts
       conflicts = []
       FOR EACH shortName, identityList IN shortNameMap DO
         IF len(identityList) > 1 THEN
           conflicts = append(conflicts, Conflict{
             ShortName: shortName,
             Identities: identityList,
           })
         END IF
       END FOR
    
    4. RETURN conflicts
END FUNCTION
```

## Error Handling

### Error Categories

1. **Invalid Identity Format**
   - **Trigger**: Identity doesn't contain exactly one "/"
   - **Message**: `"invalid skill identity '%s': must be in format repo/path"`
   - **Recovery**: User must correct the identity in local config

2. **Skill Not Found in Global Store**
   - **Trigger**: Global path doesn't exist or lacks SKILL.md
   - **Message**: `"skill '%s' not found locally, run 'skillops pull'"`
   - **Recovery**: Auto-pull from registry (sync command) or manual pull

3. **Symlink Conflict**
   - **Trigger**: Symlink exists pointing to different skill
   - **Message**: `"conflict: %s already linked to a different skill in %s (skipping)"`
   - **Recovery**: Skip conflicting skill, continue with others

4. **Filesystem Errors**
   - **Trigger**: Permission denied, disk full, etc.
   - **Message**: `"failed to create symlink for '%s' in %s: %v"`
   - **Recovery**: Display warning, continue with other skills

### Error Handling Strategy

```go
// Graceful degradation pattern used in sync command
for _, identity := range cfg.Tools[tool] {
    // Parse identity
    parts := strings.SplitN(identity, "/", 2)
    if len(parts) != 2 {
        warnings = append(warnings, fmt.Sprintf("invalid skill identity '%s', skipping", identity))
        continue  // ✓ Continue processing other skills
    }
    
    // ... attempt symlink creation ...
    
    if err := os.Symlink(globalSkillPath, symlinkPath); err != nil {
        warnings = append(warnings, fmt.Sprintf("failed to create symlink for '%s': %v", identity, err))
        continue  // ✓ Continue processing other skills
    }
    
    created++
}

// Display summary with warnings
fmt.Println(renderSyncSummary(created, autoPulled, warnings))
```

### Validation Rules

1. **Identity Format**: Must match `^[^/]+/.+$` (at least one "/" with content on both sides)
2. **Short Name**: Must not be empty, ".", or ".."
3. **Path Components**: Must not contain null bytes or other invalid filesystem characters
4. **Global Path**: Must exist and contain SKILL.md file
5. **Symlink Target**: Must be absolute path to valid skill directory

## Testing Strategy

### Unit Testing Approach

This feature involves primarily **data transformation logic** (parsing identities, constructing paths) and **filesystem operations** (creating symlinks). Property-based testing is **not applicable** here because:

1. **Filesystem operations** are side-effect-only (creating/removing symlinks)
2. **Path construction** is deterministic string manipulation
3. **Identity parsing** has a fixed format with limited valid variations

Instead, we use **example-based unit tests** with representative cases and **integration tests** for end-to-end workflows.

### Unit Test Coverage

#### 1. Identity Parsing Tests

**File**: `internal/skills/skills_test.go` (new test cases)

```go
func TestParseNestedIdentity(t *testing.T) {
    tests := []struct {
        name          string
        identity      string
        wantRepo      string
        wantSkillPath string
        wantShortName string
        wantErr       bool
    }{
        {
            name:          "2-level identity (backward compat)",
            identity:      "my-repo/logger",
            wantRepo:      "my-repo",
            wantSkillPath: "logger",
            wantShortName: "logger",
            wantErr:       false,
        },
        {
            name:          "3-level nested identity",
            identity:      "skills/skills/skill-creator",
            wantRepo:      "skills",
            wantSkillPath: "skills/skill-creator",
            wantShortName: "skill-creator",
            wantErr:       false,
        },
        {
            name:          "deeply nested identity",
            identity:      "repo/a/b/c/d/skill",
            wantRepo:      "repo",
            wantSkillPath: "a/b/c/d/skill",
            wantShortName: "skill",
            wantErr:       false,
        },
        {
            name:     "invalid - no slash",
            identity: "invalid",
            wantErr:  true,
        },
        {
            name:     "invalid - trailing slash",
            identity: "repo/skill/",
            wantErr:  true,
        },
        {
            name:     "invalid - empty repo",
            identity: "/skill",
            wantErr:  true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            parts := strings.SplitN(tt.identity, "/", 2)
            if len(parts) != 2 {
                if !tt.wantErr {
                    t.Errorf("unexpected parse error for %s", tt.identity)
                }
                return
            }
            
            repo := parts[0]
            skillPath := parts[1]
            shortName := filepath.Base(skillPath)
            
            if repo != tt.wantRepo {
                t.Errorf("repo = %v, want %v", repo, tt.wantRepo)
            }
            if skillPath != tt.wantSkillPath {
                t.Errorf("skillPath = %v, want %v", skillPath, tt.wantSkillPath)
            }
            if shortName != tt.wantShortName {
                t.Errorf("shortName = %v, want %v", shortName, tt.wantShortName)
            }
        })
    }
}
```

#### 2. Path Construction Tests

**File**: `cmd/sync_test.go` (new file)

```go
func TestGlobalPathConstruction(t *testing.T) {
    tests := []struct {
        name         string
        identity     string
        skillsDir    string
        wantGlobal   string
        wantSymlink  string
    }{
        {
            name:        "2-level identity",
            identity:    "my-repo/logger",
            skillsDir:   "/home/user/.skillops/skills",
            wantGlobal:  "/home/user/.skillops/skills/my-repo/logger",
            wantSymlink: "logger",
        },
        {
            name:        "nested identity",
            identity:    "skills/skills/skill-creator",
            skillsDir:   "/home/user/.skillops/skills",
            wantGlobal:  "/home/user/.skillops/skills/skills/skills/skill-creator",
            wantSymlink: "skill-creator",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            parts := strings.SplitN(tt.identity, "/", 2)
            repo := parts[0]
            skillPath := parts[1]
            shortName := filepath.Base(skillPath)
            
            globalPath := filepath.Join(tt.skillsDir, repo, skillPath)
            
            if globalPath != tt.wantGlobal {
                t.Errorf("globalPath = %v, want %v", globalPath, tt.wantGlobal)
            }
            if shortName != tt.wantSymlink {
                t.Errorf("shortName = %v, want %v", shortName, tt.wantSymlink)
            }
        })
    }
}
```

#### 3. Conflict Detection Tests

**File**: `internal/tui/add_test.go` (new test cases)

```go
func TestSymlinkConflictDetection(t *testing.T) {
    tests := []struct {
        name        string
        identities  []string
        wantConflict bool
        conflictName string
    }{
        {
            name:        "no conflict - different short names",
            identities:  []string{"repo-a/logger", "repo-b/auth"},
            wantConflict: false,
        },
        {
            name:        "conflict - same short name different paths",
            identities:  []string{"repo-a/tools/logger", "repo-b/utils/logger"},
            wantConflict: true,
            conflictName: "logger",
        },
        {
            name:        "no conflict - same full identity",
            identities:  []string{"repo-a/logger", "repo-a/logger"},
            wantConflict: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            shortNames := make(map[string][]string)
            
            for _, id := range tt.identities {
                parts := strings.SplitN(id, "/", 2)
                if len(parts) == 2 {
                    shortName := filepath.Base(parts[1])
                    shortNames[shortName] = append(shortNames[shortName], id)
                }
            }
            
            hasConflict := false
            for name, ids := range shortNames {
                if len(ids) > 1 {
                    hasConflict = true
                    if tt.wantConflict && name != tt.conflictName {
                        t.Errorf("conflict on wrong name: got %v, want %v", name, tt.conflictName)
                    }
                }
            }
            
            if hasConflict != tt.wantConflict {
                t.Errorf("conflict detection = %v, want %v", hasConflict, tt.wantConflict)
            }
        })
    }
}
```

### Integration Testing

#### Test Scenarios

1. **End-to-End Sync with Nested Skills**
   - Setup: Create test repo with nested skill structure
   - Action: Run `skillops sync`
   - Verify: Symlinks created with correct short names, pointing to correct global paths

2. **Add Nested Skill via TUI**
   - Setup: Pull repo with nested skills
   - Action: Run `skillops add`, select nested skill
   - Verify: Symlink created correctly, local config updated with full identity

3. **Remove Nested Skill**
   - Setup: Project with linked nested skill
   - Action: Run `skillops remove <short-name>`
   - Verify: Symlink removed, local config updated

4. **Status Display for Nested Skills**
   - Setup: Project with mix of 2-level and nested skills
   - Action: Run `skillops status`
   - Verify: All skills displayed with correct short names and link status

5. **Conflict Handling**
   - Setup: Two repos with skills having same short name
   - Action: Attempt to link both
   - Verify: Second link skipped with warning, first remains intact

#### Manual Testing Checklist

- [ ] Backward compatibility: Existing 2-level identities work unchanged
- [ ] Nested skill discovery: `skillops list` shows nested skills correctly
- [ ] Sync creates flat symlinks for nested paths
- [ ] Add TUI displays short names correctly
- [ ] Remove command works with both short name and full identity
- [ ] Status command shows correct link status for nested skills
- [ ] Conflict warnings appear when appropriate
- [ ] Cross-platform: Works on Linux, macOS, and Windows

### Test Data Setup

```bash
# Create test repository with nested structure
mkdir -p ~/.skillops/skills/test-repo/tools/advanced/logger
echo "# Logger Skill" > ~/.skillops/skills/test-repo/tools/advanced/logger/SKILL.md

mkdir -p ~/.skillops/skills/test-repo/utils/logger
echo "# Utils Logger" > ~/.skillops/skills/test-repo/utils/logger/SKILL.md

# This creates a conflict scenario: both resolve to "logger" short name
```

## Migration and Backward Compatibility

### Backward Compatibility Guarantee

**All existing 2-level identities continue to work identically:**

```go
// 2-level identity: "my-repo/logger"
parts := strings.SplitN("my-repo/logger", "/", 2)
// parts[0] = "my-repo"
// parts[1] = "logger"
shortName := filepath.Base("logger")  // "logger"

// Result: Identical behavior to current implementation
```

**No migration required**: Existing `.skillops/config.json` files work without modification.

### Validation

Users can verify backward compatibility by:

1. Running `skillops status` before and after upgrade
2. Confirming symlink paths remain unchanged for existing skills
3. Running `skillops sync` to verify idempotent behavior

### Rollback Strategy

If issues arise:
1. Revert to previous skillops version
2. Symlinks remain valid (no data corruption)
3. Local config unchanged (no format changes)

## Performance Considerations

### Impact Analysis

1. **Identity Parsing**: `filepath.Base()` is O(n) where n is path length, negligible overhead
2. **Symlink Creation**: No change in filesystem operations
3. **Discovery**: No change in recursive directory scanning
4. **Memory**: No additional data structures required

### Benchmarks

Expected performance (no measurable degradation):

- Parse 1000 identities: < 1ms
- Create 100 symlinks: < 100ms (filesystem-bound)
- Discover 100 nested skills: < 2s (existing performance)

## Security Considerations

### Path Traversal Prevention

**Existing validation remains in place:**

```go
// internal/utils/utils.go - ValidateName function
func ValidateName(name string) error {
    if name == "" || name == "." || name == ".." {
        return fmt.Errorf("invalid name: %s", name)
    }
    if strings.Contains(name, string(filepath.Separator)) {
        return fmt.Errorf("name cannot contain path separators")
    }
    return nil
}
```

**Applied to short names before symlink creation:**

```go
shortName := filepath.Base(skillPath)
if err := utils.ValidateName(shortName); err != nil {
    return fmt.Errorf("invalid skill name: %w", err)
}
```

### Symlink Safety

1. **Target validation**: Verify global path exists and contains SKILL.md
2. **Conflict detection**: Prevent overwriting existing symlinks
3. **Path sanitization**: Use `filepath.Clean()` for all constructed paths
4. **Permission checks**: Fail gracefully on permission errors

## Deployment and Rollout

### Release Strategy

1. **Version**: Include in next minor release (e.g., v1.3.0)
2. **Changelog**: Document nested path support and backward compatibility
3. **Testing**: Run full integration test suite before release
4. **Documentation**: Update README with nested path examples

### User Communication

**Release Notes**:
```markdown
## v1.3.0 - Nested Skill Paths Support

### New Features
- ✨ Support for arbitrary nesting depth in skill paths
- Skills can now be organized in nested directories (e.g., `repo/tools/advanced/logger`)
- Symlinks remain flat using the final path component

### Backward Compatibility
- All existing 2-level skill identities continue to work unchanged
- No migration required for existing projects
- Local config format unchanged

### Examples
```bash
# Pull a repo with nested skills
skillops pull https://github.com/user/skills

# Skills discovered at any depth:
# - skills/skills/skill-creator
# - skills/tools/logger
# - skills/utils/auth

# Symlinks created as:
# .kiro/skills/skill-creator
# .kiro/skills/logger
# .kiro/skills/auth
```
```

## Summary

This design introduces minimal, surgical changes to support nested skill paths:

1. **Core Change**: Replace `shortName := parts[1]` with `shortName := filepath.Base(parts[1])` in 5 locations
2. **Affected Files**: 4 files (sync.go, remove.go, status.go, add.go)
3. **Backward Compatible**: 100% compatible with existing 2-level identities
4. **No Schema Changes**: Local config format unchanged
5. **Graceful Degradation**: Conflicts detected and handled with warnings

The implementation maintains the existing architecture while extending capability to handle arbitrary nesting depth, ensuring a smooth upgrade path for all users.
