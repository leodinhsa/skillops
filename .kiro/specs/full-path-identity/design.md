# Design Document: Full-Path Identity Support

## Overview

This design fundamentally redesigns skillops to use full-path skill identities in the format `<host>/<owner>/<repo>/<path-to-skill>`, replacing the current 2-level `repo/skill` format. This change solves critical issues:

1. **Repository collision prevention**: Skills from `github.com/company-a/utils` and `github.com/company-b/utils` can coexist
2. **Arbitrary nesting support**: Skills can be located at any depth (e.g., `github.com/company/monorepo/backend/services/api/skills/auth`)
3. **Multi-source management**: Support for GitHub, GitLab, Bitbucket, and self-hosted git platforms
4. **Zero-config team collaboration**: Project config includes registries, enabling automatic skill resolution

### Key Design Principles

1. **Full-path identities**: Every skill is identified by its complete path from host to skill folder
2. **Flat symlinks**: Symlinks always use short names (final path component) to keep IDE directories organized
3. **Conflict resolution**: When multiple skills have the same short name, users provide custom symlink names via TUI
4. **Self-contained config**: Project `.skillops/config.json` includes registries for team collaboration
5. **Per-skill metadata**: Each skill has `.so-skill-meta.json` for traceability and updates

## Architecture

### High-Level Data Flow

```
┌─────────────────────────────────────────────────────────────┐
│                    Skill Identity Flow                       │
└─────────────────────────────────────────────────────────────┘

Input: "github.com/anthropics/skills/skills/skill-creator"
   │
   ├─> Identity Parser
   │   ├─> Host: "github.com"
   │   ├─> Owner: "anthropics"
   │   ├─> Repo: "skills"
   │   ├─> Path in repo: "skills/skill-creator"
   │   └─> Short Name: "skill-creator" (filepath.Base)
   │
   ├─> Global Store Path Construction
   │   └─> ~/.skillops/skills/github.com/anthropics/skills/skills/skill-creator
   │
   ├─> Symlink Path Construction
   │   └─> <project>/.kiro/skills/skill-creator
   │       (or custom name from symlink_names mapping)
   │
   └─> Local Config Storage
       └─> "github.com/anthropics/skills/skills/skill-creator" (full identity)
```

### System Components

```
┌──────────────────────────────────────────────────────────────┐
│                     Component Architecture                    │
└──────────────────────────────────────────────────────────────┘

┌─────────────────┐
│  CLI Commands   │  (cmd/*.go)
│  - pull.go      │  Organizes by host/owner/repo
│  - sync.go      │  Uses registries + metadata
│  - update.go    │  Reads metadata for source
│  - add.go       │  Auto-populates registries
│  - remove.go    │  Handles custom symlink names
│  - status.go    │  Displays full identities
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
│  - conflict.go  │  identities +        │ Stores:         │
│    (NEW)        │  custom names        │ - registries    │
│                 │                      │ - tools         │
│ Conflict TUI    │                      │ - symlink_names │
│ for renaming    │                      └─────────────────┘
└────────┬────────┘
         │
         ▼
┌─────────────────┐                      ┌─────────────────┐
│ Discovery Layer │                      │ Registry Layer  │
│(internal/skills)│                      │(internal/config)│
│  - skills.go    │  Discovers skills    │ - registry.go   │
│                 │  with full paths     │   (NEW)         │
│  - extract.go   │  Handles extraction  │                 │
│                 │  with path_in_repo   │ Matches skills  │
│  - metadata.go  │  Per-skill metadata  │ to registries   │
│    (NEW)        │  management          │                 │
└─────────────────┘                      └─────────────────┘
```

## Data Models

### Skill Identity Structure

```go
// Full identity format: <host>/<owner>/<repo>/<path-to-skill>
type SkillIdentity string

// Parsed components
type ParsedIdentity struct {
    Full       string  // "github.com/anthropics/skills/skills/logger"
    Host       string  // "github.com"
    Owner      string  // "anthropics"
    Repo       string  // "skills"
    PathInRepo string  // "skills/logger"
    ShortName  string  // "logger" (filepath.Base of PathInRepo)
}

// Parse function
// See Algorithm Specifications section for complete validation logic
func ParseIdentity(identity string) (*ParsedIdentity, error) {
    parts := strings.Split(identity, "/")
    if len(parts) < 4 {
        return nil, fmt.Errorf("invalid identity: need at least host/owner/repo/skill")
    }
    
    // Validate all components for path traversal (see algorithm spec)
    for _, part := range parts {
        if part == "" || part == "." || part == ".." {
            return nil, fmt.Errorf("invalid identity: component cannot be empty, '.', or '..'")
        }
    }
    
    pathInRepo := strings.Join(parts[3:], "/")
    
    return &ParsedIdentity{
        Full:       identity,
        Host:       parts[0],
        Owner:      parts[1],
        Repo:       parts[2],
        PathInRepo: pathInRepo,
        ShortName:  filepath.Base(pathInRepo),
    }, nil
}
```

### Local Config Schema V2

```go
type LocalConfig struct {
    Version      string              `json:"version"`       // "2"
    Registries   []Registry          `json:"registries"`
    Tools        map[string][]string `json:"tools"`         // tool -> []skill_identity
    SymlinkNames map[string]string   `json:"symlink_names,omitempty"` // identity -> custom_name
}

type Registry struct {
    URL      string `json:"url"`
    Name     string `json:"name"`
    Priority int    `json:"priority"`
}
```

**Example:**
```json
{
  "version": "2",
  "registries": [
    {
      "url": "https://github.com/anthropics",
      "name": "Anthropic Skills",
      "priority": 1
    }
  ],
  "tools": {
    "kiro": [
      "github.com/anthropics/skills/skills/logger",
      "github.com/company-a/utils/tools/logger"
    ]
  },
  "symlink_names": {
    "github.com/company-a/utils/tools/logger": "logger-utils"
  }
}
```

**Note:** `symlink_names` only contains custom names (conflicts). If not present, use short name.

### Skill Metadata Schema

```go
type SkillMetadata struct {
    RepoURL    string    `json:"repo_url"`
    PathInRepo string    `json:"path_in_repo"`
    PulledAt   time.Time `json:"pulled_at"`
    CommitHash string    `json:"commit_hash"`
}
```

**File location:** `<skill-folder>/.so-skill-meta.json`

**Example:**
```json
{
  "repo_url": "https://github.com/anthropics/skills",
  "path_in_repo": "skills/logger",
  "pulled_at": "2026-05-06T10:30:00Z",
  "commit_hash": "abc123def456"
}
```

### Repo Metadata Schema (for full pulls)

```go
type RepoMetadata struct {
    RepoURL    string    `json:"repo_url"`
    PulledAt   time.Time `json:"pulled_at"`
    CommitHash string    `json:"commit_hash"`
}
```

**File location:** `<repo-root>/.so-repo-meta.json`

## Algorithm Specifications

### 1. Identity Parsing Algorithm

```
FUNCTION ParseSkillIdentity(identity: string) -> (ParsedIdentity, error)
  INPUT: Full skill identity (e.g., "github.com/anthropics/skills/skills/logger")
  OUTPUT: Parsed components or error
  
  STEPS:
    1. Split identity on "/"
       parts = strings.Split(identity, "/")
    
    2. Validate minimum components
       IF len(parts) < 4 THEN
         RETURN error("invalid identity: need at least host/owner/repo/skill")
       END IF
    
    3. Validate all components for path traversal
       FOR EACH part IN parts DO
         IF part == "" OR part == "." OR part == ".." THEN
           RETURN error("invalid identity: component cannot be empty, '.', or '..'")
         END IF
       END FOR
    
    4. Extract components
       host = parts[0]
       owner = parts[1]
       repo = parts[2]
       pathInRepo = strings.Join(parts[3:], "/")
       shortName = filepath.Base(pathInRepo)
    
    5. Validate short name
       IF shortName == "" OR shortName == "." OR shortName == ".." THEN
         RETURN error("invalid skill path")
       END IF
    
    6. RETURN ParsedIdentity{
         Full: identity,
         Host: host,
         Owner: owner,
         Repo: repo,
         PathInRepo: pathInRepo,
         ShortName: shortName,
       }
END FUNCTION
```

### 2. Registry Matching Algorithm

```
FUNCTION MatchRegistry(skillIdentity: string, registries: []Registry) -> (string, error)
  INPUT: 
    - skillIdentity: Full skill identity
    - registries: List of configured registries
  OUTPUT: Clone URL or error
  
  STEPS:
    1. Parse skill identity
       parsed = ParseSkillIdentity(skillIdentity)
    
    2. Sort registries by priority
       sortedRegs = SortByPriority(registries)
    
    3. Try each registry
       FOR EACH reg IN sortedRegs DO
         IF MatchesRegistry(reg, parsed.Host, parsed.Owner) THEN
           cloneURL = reg.URL + "/" + parsed.Repo
           RETURN cloneURL, nil
         END IF
       END FOR
    
    4. No match found
       RETURN "", error("no registry found for " + parsed.Host + "/" + parsed.Owner)
END FUNCTION

FUNCTION MatchesRegistry(reg: Registry, host: string, owner: string) -> bool
  STEPS:
    1. Normalize registry URL
       normalized = NormalizeURL(reg.URL)
       // Remove protocol (https://, git@), replace : with /
       // Example: "git@github.com:anthropics" -> "github.com/anthropics"
    
    2. Build expected path
       expectedPath = host + "/" + owner
    
    3. Check exact match or prefix match
       // Exact match: registry is exactly host/owner
       IF normalized == expectedPath THEN
         RETURN true
       END IF
       
       // Prefix match: registry is host/owner/... (more specific)
       IF strings.HasPrefix(normalized, expectedPath + "/") THEN
         RETURN true
       END IF
       
       RETURN false
END FUNCTION
```

### 3. Symlink Creation Algorithm

```
FUNCTION CreateSkillSymlink(identity: string, tool: string, localConfig: LocalConfig) -> (wasCreated: bool, error)
  INPUT:
    - identity: Full skill identity
    - tool: IDE name (e.g., "kiro")
    - localConfig: Project configuration
  OUTPUT: wasCreated flag and error if creation fails
  
  STEPS:
    1. Parse identity
       parsed = ParseSkillIdentity(identity)
    
    2. Determine symlink name
       symlinkName = localConfig.SymlinkNames[identity]
       IF symlinkName == "" THEN
         symlinkName = parsed.ShortName  // Use default
       END IF
    
    3. Construct paths
       globalPath = filepath.Join(
         config.SkillsDir,
         parsed.Host,
         parsed.Owner,
         parsed.Repo,
         parsed.PathInRepo,
       )
       
       toolDir = GetToolSkillsDir(tool)  // e.g., .kiro/skills
       symlinkPath = filepath.Join(toolDir, symlinkName)
    
    4. Verify global skill exists
       IF NOT exists(globalPath + "/SKILL.md") THEN
         RETURN false, error("skill not found in global store: " + identity)
       END IF
    
    5. Check for conflicts
       IF exists(symlinkPath) THEN
         existingTarget = readlink(symlinkPath)
         IF existingTarget != globalPath THEN
           RETURN false, error("symlink conflict: " + symlinkName + " already exists")
         ELSE
           RETURN false, nil  // Already linked correctly (idempotent, not created)
         END IF
       END IF
    
    6. Create symlink
       err = os.Symlink(globalPath, symlinkPath)
       IF err != nil THEN
         RETURN false, error("failed to create symlink: " + err)
       END IF
    
    7. RETURN true, nil  // Successfully created
END FUNCTION
```

### 4. Conflict Detection and Resolution Algorithm

```
FUNCTION DetectConflicts(identities: []string, localConfig: LocalConfig) -> []Conflict
  INPUT: List of skill identities to be linked
  OUTPUT: List of conflicts
  
  STEPS:
    1. Build symlink name map
       symlinkMap = map[string][]string{}
       
       FOR EACH identity IN identities DO
         parsed = ParseSkillIdentity(identity)
         
         // Check if has custom name
         symlinkName = localConfig.SymlinkNames[identity]
         IF symlinkName == "" THEN
           symlinkName = parsed.ShortName
         END IF
         
         symlinkMap[symlinkName] = append(symlinkMap[symlinkName], identity)
       END FOR
    
    2. Identify conflicts
       conflicts = []
       FOR EACH symlinkName, identityList IN symlinkMap DO
         IF len(identityList) > 1 THEN
           conflicts = append(conflicts, Conflict{
             SymlinkName: symlinkName,
             Identities:  identityList,
           })
         END IF
       END FOR
    
    3. RETURN conflicts
END FUNCTION

FUNCTION ResolveConflictsTUI(conflicts: []Conflict) -> (map[string]string, error)
  INPUT: List of conflicts
  OUTPUT: Map of identity -> custom_symlink_name
  
  STEPS:
    1. Initialize TUI model
       model = ConflictResolutionModel{
         conflicts: conflicts,
         customNames: map[string]string{},
       }
    
    2. For each conflict, show input form
       FOR EACH conflict IN conflicts DO
         FOR EACH identity IN conflict.Identities DO
           // Display identity
           // Show input field for custom name
           // Validate: no path separators, not empty, not "." or ".."
           // Check for conflicts with existing names
         END FOR
       END FOR
    
    3. User provides custom names
       // TUI handles input, validation, navigation
    
    4. RETURN customNames map
END FUNCTION
```

### 5. Sync Algorithm with Registry Resolution

```
FUNCTION SyncSkills(localConfig: LocalConfig) -> (created: int, autoPulled: int, errors: []string)
  INPUT: Local project configuration
  OUTPUT: Statistics and errors
  
  STEPS:
    1. Initialize counters
       created = 0
       autoPulled = 0
       errors = []
    
    2. For each tool and its skills
       FOR EACH tool, identities IN localConfig.Tools DO
         toolDir = GetToolSkillsDir(tool)
         os.MkdirAll(toolDir, 0755)
         
         FOR EACH identity IN identities DO
           // Parse identity
           parsed = ParseSkillIdentity(identity)
           IF error THEN
             errors = append(errors, "invalid identity: " + identity)
             CONTINUE
           END IF
           
           // Construct global path
           globalPath = filepath.Join(
             config.SkillsDir,
             parsed.Host,
             parsed.Owner,
             parsed.Repo,
             parsed.PathInRepo,
           )
           
           // Check if skill exists
           IF exists(globalPath) THEN
             // Create symlink (returns wasCreated bool)
             wasCreated, err = CreateSkillSymlink(identity, tool, localConfig)
             IF err != nil THEN
               errors = append(errors, err.Error())
             ELSE IF wasCreated THEN
               created++  // Only count if actually created (not idempotent no-op)
             END IF
           ELSE
             // Try to pull from registries
             cloneURL, err = MatchRegistry(identity, localConfig.Registries)
             IF err != nil THEN
               errors = append(errors, "no registry for " + identity)
               CONTINUE
             END IF
             
             // Pull skill
             err = PullSkillFromURL(cloneURL, parsed.PathInRepo, globalPath)
             IF err != nil THEN
               errors = append(errors, "failed to pull " + identity + ": " + err)
               CONTINUE
             END IF
             
             autoPulled++
             
             // Create symlink
             wasCreated, err = CreateSkillSymlink(identity, tool, localConfig)
             IF err != nil THEN
               errors = append(errors, err.Error())
             ELSE IF wasCreated THEN
               created++
             END IF
           END IF
         END FOR
       END FOR
    
    3. RETURN created, autoPulled, errors
END FUNCTION
```

### 6. Update Algorithm with Metadata

```
FUNCTION UpdateSkill(skillPath: string) -> error
  INPUT: Absolute path to skill in global store
  OUTPUT: Error if update fails
  
  STEPS:
    1. Load metadata
       metaPath = filepath.Join(skillPath, ".so-skill-meta.json")
       meta = LoadSkillMetadata(metaPath)
       IF error THEN
         RETURN error("no metadata found, cannot update")
       END IF
    
    2. Clone repository to temp
       tempDir = os.MkdirTemp("", "skillops-update-*")
       defer os.RemoveAll(tempDir)
       
       err = git.Clone(meta.RepoURL, tempDir)
       IF err != nil THEN
         RETURN error("failed to clone: " + err)
       END IF
    
    3. Extract skill from path_in_repo
       skillSource = filepath.Join(tempDir, meta.PathInRepo)
       IF NOT exists(skillSource + "/SKILL.md") THEN
         RETURN error("skill not found at " + meta.PathInRepo)
       END IF
    
    4. Replace skill content
       os.RemoveAll(skillPath)
       utils.CopyDir(skillSource, skillPath)
    
    5. Update metadata
       meta.PulledAt = time.Now()
       meta.CommitHash = getLatestCommit(tempDir)
       SaveSkillMetadata(metaPath, meta)
    
    6. RETURN nil
END FUNCTION
```

### 7. Pull Skill From URL Algorithm

```
FUNCTION PullSkillFromURL(repoURL: string, pathInRepo: string, destSkillDir: string) -> error
  INPUT:
    - repoURL: Full git repository URL (e.g., "https://github.com/anthropics/skills")
    - pathInRepo: Path from repo root to skill (e.g., "skills/logger")
    - destSkillDir: Destination path in global store (e.g., "~/.skillops/skills/github.com/anthropics/skills/skills/logger")
  OUTPUT: Error if pull fails
  
  PURPOSE: Clone a repository, extract a specific skill subdirectory, and save metadata.
          Used by both Pull Command (--skill flag) and Sync Command (auto-pull).
  
  STEPS:
    1. Create temporary directory
       tempDir = os.MkdirTemp("", "skillops-pull-*")
       defer os.RemoveAll(tempDir)
    
    2. Clone repository (shallow clone for efficiency)
       err = git.Clone(repoURL, tempDir, "--depth", "1")
       IF err != nil THEN
         RETURN error("failed to clone repository: " + err)
       END IF
    
    3. Construct skill source path
       skillSource = filepath.Join(tempDir, pathInRepo)
    
    4. Verify skill exists
       IF NOT exists(skillSource + "/SKILL.md") THEN
         RETURN error("skill not found at path: " + pathInRepo)
       END IF
    
    5. Create destination parent directories
       err = os.MkdirAll(filepath.Dir(destSkillDir), 0755)
       IF err != nil THEN
         RETURN error("failed to create destination parent: " + err)
       END IF
    
    6. Copy skill directory atomically
       // Use temp-then-rename for atomicity
       tempDest = destSkillDir + ".tmp"
       err = utils.CopyDir(skillSource, tempDest)
       IF err != nil THEN
         os.RemoveAll(tempDest)
         RETURN error("failed to copy skill: " + err)
       END IF
       
       // Atomic rename
       err = os.Rename(tempDest, destSkillDir)
       IF err != nil THEN
         os.RemoveAll(tempDest)
         RETURN error("failed to finalize skill: " + err)
       END IF
    
    7. Get commit hash from cloned repo
       commitHash = getLatestCommit(tempDir)
    
    8. Save skill metadata
       meta = SkillMetadata{
         RepoURL:    repoURL,
         PathInRepo: pathInRepo,
         PulledAt:   time.Now(),
         CommitHash: commitHash,
       }
       
       metaPath = filepath.Join(destSkillDir, ".so-skill-meta.json")
       err = SaveSkillMetadata(metaPath, meta)
       IF err != nil THEN
         // Non-fatal: warn but don't fail the pull
         log.Warn("failed to save metadata: " + err)
       END IF
    
    9. RETURN nil
END FUNCTION
```

**Key design decisions:**
- **Shallow clone** (`--depth 1`) for efficiency
- **Atomic copy** (temp-then-rename) to prevent partial state
- **Metadata save is non-fatal** (warn but don't fail)
- **Cleanup on error** (remove temp directories)

### 8. Parse Repository URL Algorithm

```
FUNCTION ParseRepoURL(repoURL: string) -> (host: string, owner: string, repo: string, error)
  INPUT: Git repository URL in various formats
  OUTPUT: Extracted host, owner (may contain /), and repo name
  
  PURPOSE: Extract host, owner, and repo from git URLs supporting:
           - HTTPS: https://github.com/owner/repo.git
           - SSH: git@github.com:owner/repo.git
           - Self-hosted: https://gitlab.company.internal/group/subgroup/repo
           - Multi-level groups (GitLab): owner can contain "/"
  
  STEPS:
    1. Normalize URL
       url = strings.TrimSpace(repoURL)
       url = strings.TrimSuffix(url, ".git")  // Remove .git suffix
    
    2. Detect URL format and extract path
       IF strings.HasPrefix(url, "git@") THEN
         // SSH format: git@github.com:owner/repo
         url = strings.TrimPrefix(url, "git@")
         parts = strings.SplitN(url, ":", 2)
         IF len(parts) != 2 THEN
           RETURN error("invalid SSH URL format")
         END IF
         host = parts[0]
         pathPart = parts[1]
       ELSE IF strings.HasPrefix(url, "https://") OR strings.HasPrefix(url, "http://") THEN
         // HTTPS format: https://github.com/owner/repo
         url = strings.TrimPrefix(url, "https://")
         url = strings.TrimPrefix(url, "http://")
         parts = strings.SplitN(url, "/", 2)
         IF len(parts) != 2 THEN
           RETURN error("invalid HTTPS URL format")
         END IF
         host = parts[0]
         pathPart = parts[1]
       ELSE
         RETURN error("unsupported URL format (must be HTTPS or SSH)")
       END IF
    
    3. Extract owner and repo from path
       // pathPart examples:
       // - "anthropics/skills" → owner="anthropics", repo="skills"
       // - "group/subgroup/project" → owner="group/subgroup", repo="project"
       
       pathComponents = strings.Split(pathPart, "/")
       IF len(pathComponents) < 2 THEN
         RETURN error("URL must contain at least owner/repo")
       END IF
       
       // Repo is always the last component
       repo = pathComponents[len(pathComponents)-1]
       
       // Owner is everything before repo (supports multi-level groups)
       owner = strings.Join(pathComponents[0:len(pathComponents)-1], "/")
    
    4. Validate components
       IF host == "" OR owner == "" OR repo == "" THEN
         RETURN error("invalid URL: empty component")
       END IF
       
       // Validate no path traversal in components
       allComponents = append(strings.Split(owner, "/"), repo)
       FOR EACH component IN allComponents DO
         IF component == "" OR component == "." OR component == ".." THEN
           RETURN error("invalid URL: component cannot be empty, '.', or '..'")
         END IF
       END FOR
    
    5. RETURN host, owner, repo, nil
END FUNCTION
```

**Examples:**

| Input URL | Host | Owner | Repo |
|-----------|------|-------|------|
| `https://github.com/anthropics/skills.git` | `github.com` | `anthropics` | `skills` |
| `git@github.com:company/utils.git` | `github.com` | `company` | `utils` |
| `https://gitlab.com/group/subgroup/project` | `gitlab.com` | `group/subgroup` | `project` |
| `https://gitlab.company.internal/team/backend/api` | `gitlab.company.internal` | `team/backend` | `api` |
| `git@bitbucket.org:org/repo` | `bitbucket.org` | `org` | `repo` |

**Multi-level group support:**
- Owner can contain "/" for GitLab nested groups
- Global store path: `~/.skillops/skills/<host>/<owner>/<repo>/`
- Example: `~/.skillops/skills/gitlab.com/group/subgroup/project/`
- Identity: `gitlab.com/group/subgroup/project/skills/auth`

## Component Specifications

### 1. Pull Command Changes

**File:** `cmd/pull.go`

**Changes:**
- Extract host and owner from repository URL using `git.ParseRepoURL`
- Create directory structure: `~/.skillops/skills/<host>/<owner>/<repo>`
- Save `.so-repo-meta.json` for full pulls
- Use `PullSkillFromURL` for specific skill pulls with `--skill` flag

**Key logic:**
```go
// Extract host/owner/repo from URL
// Supports multi-level groups (e.g., GitLab group/subgroup/project)
host, owner, repo := git.ParseRepoURL(url)

// Create nested directory
dest := filepath.Join(config.SkillsDir, host, owner, repo)

// For --skill flag, use PullSkillFromURL
if skillFlag != "" {
    pathInRepo := skillFlag  // e.g., "skills/logger"
    destSkillDir := filepath.Join(dest, pathInRepo)
    return PullSkillFromURL(url, pathInRepo, destSkillDir)
}

// For full repo pull
git.Clone(url, dest)
SaveRepoMetadata(dest, RepoMetadata{...})
```

**Note:** `PullSkillFromURL` is shared with Sync Command for auto-pull functionality. See Algorithm Specifications section for complete implementation details.

### 2. Sync Command Changes

**File:** `cmd/sync.go`

**Changes:**
- Parse full-path identities
- Use registry matching for missing skills
- Support custom symlink names from config
- Error (not fallback) when no registry matches

**Key logic:**
```go
// Parse identity
parsed := ParseSkillIdentity(identity)

// Get symlink name (custom or default)
symlinkName := localConfig.SymlinkNames[identity]
if symlinkName == "" {
    symlinkName = parsed.ShortName
}

// Try registry matching if skill missing
if !exists(globalPath) {
    cloneURL, err := MatchRegistry(identity, localConfig.Registries)
    if err != nil {
        return fmt.Errorf("no registry found for %s", identity)
    }
    PullSkillFromURL(cloneURL, parsed.PathInRepo, globalPath)
}
```

### 3. Add TUI Changes

**File:** `internal/tui/add.go`

**Changes:**
- Display full identities with short names
- Detect conflicts before confirmation
- Launch conflict resolution TUI when needed
- Auto-populate registries from skill metadata
- Save custom symlink names to config

**New file:** `internal/tui/conflict.go`

**Purpose:** Interactive TUI for resolving symlink conflicts

**Features:**
- Display conflicting identities
- Input fields for custom symlink names
- Real-time validation
- Navigation between fields

### 4. Remove Command Changes

**File:** `cmd/remove.go`

**Changes:**
- Support symlink name or full identity for selection
- When symlink name matches multiple skills, launch disambiguation TUI
- Display full identities in disambiguation TUI
- Remove both skill identity and custom symlink name from config

**Disambiguation TUI:**
```
┌─────────────────────────────────────────────────────────────┐
│  MULTIPLE SKILLS MATCH: logger                               │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Select which skill to remove:                               │
│                                                              │
│  [ ] github.com/company-a/utils/tools/logger                │
│  [ ] github.com/company-b/helpers/services/logger           │
│                                                              │
│  [↑↓] Navigate  [Space] Select  [Enter] Confirm             │
└─────────────────────────────────────────────────────────────┘
```

### 5. Discovery Changes

**File:** `internal/skills/skills.go`

**Changes:**
- Walk global store starting from `~/.skillops/skills/`
- Construct full-path identities from filesystem paths
- Extract host, owner, repo from directory structure
- Skip hidden directories (starting with ".")

**Key logic:**
```go
// Walk global store
filepath.WalkDir(config.SkillsDir, func(path string, d fs.DirEntry, err error) error {
    if err != nil {
        return err
    }
    
    // Skip hidden directories (.git, .so-*, etc.)
    if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
        return filepath.SkipDir
    }
    
    // Check for SKILL.md
    if d.IsDir() && exists(filepath.Join(path, "SKILL.md")) {
        // Found a skill
        relPath, _ := filepath.Rel(config.SkillsDir, path)
        identity := filepath.ToSlash(relPath)
        // identity = "github.com/anthropics/skills/skills/logger"
        
        skills = append(skills, Skill{
            Identity: identity,
            Path:     path,
        })
    }
    return nil
})
```

**Rationale for skipping hidden directories:**
- Avoids scanning `.git/` subdirectories (thousands of inodes per repo)
- Prevents false positives from crafted SKILL.md files in `.git/`
- Skips `.so-repo-meta.json` and `.so-skill-meta.json` parent directories
- Performance optimization for large global stores

### 6. New Registry Matcher

**File:** `internal/config/registry.go` (NEW)

**Purpose:** Match skill identities to registries

**Key functions:**
- `MatchRegistry(identity, registries) -> cloneURL`
- `MatchesRegistry(registry, host, owner) -> bool`
- `BuildCloneURL(registry, repo) -> string`

### 7. New Metadata Manager

**File:** `internal/skills/metadata.go` (NEW)

**Purpose:** Manage per-skill metadata

**Key functions:**
- `SaveSkillMetadata(skillPath, metadata)`
- `LoadSkillMetadata(skillPath) -> metadata`
- `HasMetadata(skillPath) -> bool`

## Error Handling

### Error Categories

1. **Invalid Identity Format**
   - **Trigger**: Identity has < 4 components
   - **Message**: `"invalid skill identity '%s': need at least host/owner/repo/skill"`
   - **Recovery**: User must fix identity in config

2. **No Registry Match**
   - **Trigger**: No registry matches skill's host/owner
   - **Message**: `"no registry found for %s/%s"`
   - **Recovery**: User must add registry to config

3. **Skill Not Found**
   - **Trigger**: Global path doesn't exist
   - **Message**: `"skill '%s' not found in global store"`
   - **Recovery**: Sync will attempt auto-pull if registry configured

4. **Symlink Conflict**
   - **Trigger**: Symlink name already used by different skill
   - **Message**: `"symlink conflict: '%s' already linked to different skill"`
   - **Recovery**: Launch conflict resolution TUI

5. **Metadata Missing**
   - **Trigger**: `.so-skill-meta.json` not found during update
   - **Message**: `"no metadata found for skill, cannot update"`
   - **Recovery**: User must re-pull skill

### Error Handling Strategy

**Graceful degradation:**
- Continue processing other skills when one fails
- Collect all errors and display summary at end
- Never corrupt config or leave partial state

**Clear error messages:**
- Include full skill identity in error
- Suggest recovery action
- Show context (which tool, which operation)

## Testing Strategy

### Unit Tests

**Identity Parsing:**
- Valid full-path identities
- Nested paths (5+ levels)
- Invalid formats (< 4 components, empty components)
- Edge cases (special characters, very long paths)

**Registry Matching:**
- Exact matches
- Multiple registries with priorities
- No match scenarios
- Different protocols (HTTPS, SSH)

**Conflict Detection:**
- No conflicts
- Two skills same short name
- Three+ skills same short name
- Custom names preventing conflicts

**Symlink Creation:**
- Default short names
- Custom names from config
- Idempotent behavior
- Conflict scenarios

### Integration Tests

**End-to-End Workflows:**

1. **Pull → Add → Sync**
   - Pull repo with nested skills
   - Add skills to project
   - Clone project on different machine
   - Sync successfully pulls from registries

2. **Conflict Resolution**
   - Add two skills with same short name
   - TUI prompts for custom names
   - Symlinks created with custom names
   - Sync recreates custom-named symlinks

3. **Multi-Registry**
   - Configure multiple registries
   - Skills from different sources
   - Priority ordering works correctly

4. **Update with Metadata**
   - Pull specific skill
   - Update skill
   - Metadata used for source
   - Skill updated correctly

## Migration and Compatibility

**Note:** There are no existing users, so no migration or backward compatibility is required.

**Design decision:** The system does not support 2-level identities (`repo/skill`). All identities must be full-path format (`host/owner/repo/skill`).

**Config version handling:** If a config file has `"version": "1"` or no version field, the system SHALL fail with an error message: `"Config version 1 detected. This version requires config v2. Please run: skillops init"`

**Rationale:**
- Clean break from old design
- Simpler implementation (no compatibility shims)
- No technical debt from supporting legacy format
- Clear error message guides users to resolution

## Performance Considerations

### Optimizations

1. **Identity Parsing**: O(n) where n is path length, negligible
2. **Registry Matching**: O(m) where m is number of registries, typically < 20
3. **Discovery**: O(s) where s is number of skills, optimized with early SKILL.md check
4. **Conflict Detection**: O(n) where n is number of skills being added

**Note:** Performance is not a primary concern for this feature. The system is designed for correctness and usability first.

## Security Considerations

### Path Traversal Prevention

**Validation at multiple levels:**
1. Identity parsing validates ALL components (not just shortName) for ".." and "."
2. Symlink name validation prevents path separators
3. Global path construction uses `filepath.Join` (safe)
4. Custom names validated before use

**Complete validation:**
```go
// Validate every component during parsing
for _, part := range parts {
    if part == "" || part == "." || part == ".." {
        return fmt.Errorf("invalid component: %s", part)
    }
}
```

### Symlink Safety

1. **Target validation**: Verify global path exists and contains SKILL.md
2. **Conflict detection**: Prevent overwriting existing symlinks
3. **Idempotent operations**: Safe to run multiple times
4. **Permission checks**: Fail gracefully on permission errors

### Registry Security

1. **URL validation**: Check for valid git URL format
2. **Protocol support**: HTTPS and SSH only
3. **No arbitrary code execution**: Only git clone operations
4. **Timeout handling**: Network operations have timeouts

## Summary

This design introduces a comprehensive identity system that:

1. **Prevents collisions** through full-path identities
2. **Supports arbitrary nesting** with any directory structure
3. **Enables team collaboration** via self-contained config
4. **Provides traceability** through per-skill metadata
5. **Handles conflicts gracefully** with interactive resolution

**Key changes:**
- Identity format: `<host>/<owner>/<repo>/<path-to-skill>`
- Global store: Organized by host/owner/repo
- Config v2: Includes registries and custom symlink names
- Metadata: Per-skill `.so-skill-meta.json` files
- Conflict resolution: Interactive TUI for custom naming

**Breaking changes:**
- Identity format completely changed
- Global store structure reorganized
- Config schema updated to v2
- No backward compatibility (no existing users)
