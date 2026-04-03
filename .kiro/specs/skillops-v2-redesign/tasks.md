# Implementation Plan: SkillOps v2 Redesign

## Overview

Implement the v2 redesign in five phases: trim the default IDE list and add local config foundation, rewrite core commands (`init`, `add`, `remove`), add visibility commands (`status`, `sync`), add enterprise registry support, then clean up deprecated code.

All property-based tests use `pgregory.net/rapid`. Filesystem tests use `t.TempDir()` to isolate state.

## Tasks

- [x] 1. Phase 1 — Foundation: trim default IDEs and add local config

  - [x] 1.1 Trim `defaultAgentics` in `internal/config/config.go` to exactly 9 entries
    - Replace the current 35+ entry map with only: `claude-code`, `cursor`, `windsurf`, `kiro`, `gemini-cli`, `goose`, `github-copilot`, `opencode`, `antigravity`
    - Comment out (do not delete) all removed entries so they can be re-enabled
    - Verify `EnsureConfig()` logic is unchanged — it only adds missing keys, never removes existing ones
    - _Requirements: 1.1, 1.2, 1.3, 1.4_

  - [ ]* 1.2 Write unit tests for trimmed default agentics
    - `TestDefaultAgentics_Count` — assert `len(defaultAgentics) == 9`
    - `TestDefaultAgentics_Paths` — assert each of the 9 tools maps to its canonical path
    - _Requirements: 1.1, 1.2_

  - [ ]* 1.3 Write property test: EnsureConfig preserves existing entries
    - **Property 1: EnsureConfig preserves existing entries**
    - Generator: `rapid.MapOf(rapid.StringMatching(`[a-z][a-z0-9-]*`), rapid.StringMatching(`[a-z./]+`))` for arbitrary existing tool→path entries
    - Assert all original entries are present and unchanged after `EnsureConfig()` runs
    - **Validates: Requirements 1.4**

  - [x] 1.4 Create `internal/config/localconfig.go` with `LocalConfig` struct and all required functions
    - Define `LocalConfig` struct: `Version string`, `Tools map[string][]string`
    - Implement `LocalConfigPath()` using `os.Getwd()` + `filepath.Join(".skillops", "config.json")`
    - Implement `ReadLocalConfig()` — wrap `os.ErrNotExist` gracefully
    - Implement `WriteLocalConfig()` — `os.MkdirAll` for `.skillops/`, JSON with 2-space indent, version always `"1"`
    - Implement `GetActiveTools()`, `GetToolSkills()`, `AddSkillToTool()`, `RemoveSkillFromTool()`, `SetActiveTools()`
    - `AddSkillToTool` is a no-op if identity already present; `RemoveSkillFromTool` is a no-op if not present
    - `SetActiveTools` preserves skill entries for tools that remain, deletes entries for removed tools
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6_

  - [ ]* 1.5 Write unit tests for `localconfig.go`
    - `TestLocalConfigPath` — assert path ends with `.skillops/config.json`
    - `TestReadLocalConfig_NotFound` — assert error wraps `os.ErrNotExist`
    - `TestAddSkillToTool_Idempotent` — adding same identity twice results in one entry
    - `TestRemoveSkillFromTool_NoOp` — removing absent identity returns nil
    - _Requirements: 2.1, 2.2, 2.6_

  - [ ]* 1.6 Write property test: LocalConfig round-trip
    - **Property 2: LocalConfig round-trip**
    - Generator: random `LocalConfig` with valid tool names and `"repo/skill"` identity strings
    - Assert `WriteLocalConfig` then `ReadLocalConfig` produces a value equal to the original
    - **Validates: Requirements 2.2, 2.3**

- [x] 2. Checkpoint — ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 3. Phase 2 — Core commands: `init`, `add`, `remove`

  - [x] 3.1 Create `cmd/init.go` and update `internal/tui/tui.go` with a new `initModel`
    - Register `initCmd` with `GroupID: "project"`, no flags
    - Add `RunInit()` entry point in `internal/tui/tui.go` (or a new `internal/tui/init.go`)
    - Build `initModel` (new struct, separate from the existing `checklistModel`) with states: `CHECKLIST → CONFIRM → APPLY`
    - Pre-check logic reads from `ReadLocalConfig()` — tools already in local config are pre-checked
    - Confirm screen shows `+` for newly selected tools and `-` for deselected tools
    - `applyChanges()`: call `SetActiveTools(selected)`, `os.MkdirAll` for newly added tool skill dirs, enumerate and remove symlinks (not root dirs) for deselected tools
    - Safety check: validate removal path is within `<cwd>/<toolRootDir>/skills/` before any `os.Remove`
    - `quitting bool` field; `View()` returns `""` when quitting; final output via `fmt.Println` after `p.Run()`
    - Keyboard: `↑`/`↓` navigate, `space` toggle, `enter` confirm, `esc`/`ctrl+c` cancel
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7, 3.8, 9.1, 9.2, 9.3, 9.4, 9.5, 9.6_

  - [ ]* 3.2 Write property test: init apply sets exactly the selected tools
    - **Property 3: init apply sets exactly the selected tools**
    - Generator: random subset of the 9 known tool names
    - Call `applyChanges()` with the subset; assert `GetActiveTools()` returns exactly that set
    - **Validates: Requirements 3.3, 3.5**

  - [ ]* 3.3 Write property test: init creates skills directories for selected tools
    - **Property 4: init creates skills directories for selected tools**
    - Generator: random tool selection from the 9 defaults
    - After `applyChanges()`, assert each selected tool's skills dir exists on disk (use `t.TempDir()`)
    - **Validates: Requirements 3.4, 4.7**

  - [ ]* 3.4 Write property test: init does not delete root IDE directories
    - **Property 5: init does not delete root IDE directories**
    - Generator: random tool deselection; pre-create root IDE dirs in `t.TempDir()`
    - After `applyChanges()`, assert root dirs still exist
    - **Validates: Requirements 3.6**

  - [ ]* 3.5 Write property test: init is idempotent
    - **Property 6: init is idempotent**
    - Generator: random tool selection
    - Run `applyChanges()` twice with the same selection; assert local config and symlinks are identical after both runs
    - **Validates: Requirements 3.7**

  - [x] 3.6 Create `cmd/add.go` with TUI flow and non-TUI flag paths
    - Register `addCmd` with `GroupID: "project"`, optional positional arg `[skill]`, flags `--all` and `--tool string`
    - Guard: if `.skillops/config.json` not found, exit with `"Run 'skillops init' first"`
    - TUI flow (no args): screen 1 skill selection from `skills.Discover()` grouped by repo; screen 2 `toolSelectModel` checklist of active tools; screen 3 confirm showing `+ <short-name> → <tool>`
    - Non-TUI: `--all` links into all active tools; `--tool kiro,cursor` links into specified tools only
    - Conflict detection: `os.Lstat` the target path; if symlink exists pointing to different repo → warn and skip; if same path → skip silently
    - Short name derived as `strings.SplitN(identity, "/", 2)[1]`
    - Create tool skills dir with `os.MkdirAll` if missing
    - After creating symlink, call `AddSkillToTool(tool, repoSkill)`
    - `quitting bool`; `View()` returns `""` when quitting; final output after `p.Run()`
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6, 4.7, 4.8, 4.9, 4.10, 9.1–9.6_

  - [ ]* 3.7 Write property test: add targets exactly the specified tools
    - **Property 7: add targets exactly the specified tools**
    - Generator: random active tool set, random target subset
    - After add completes, assert `GetToolSkills(tool)` contains the identity for every targeted tool and not for non-targeted tools that didn't already have it
    - **Validates: Requirements 4.4, 4.5**

  - [ ]* 3.8 Write property test: add uses short name as symlink filename
    - **Property 8: add uses short name as symlink filename**
    - Generator: random `"repo/skill"` strings (valid path components)
    - Assert symlink filename on disk equals the portion after `"/"`
    - **Validates: Requirements 4.6**

  - [ ]* 3.9 Write property test: add round-trip — symlink and local config are consistent
    - **Property 9: add round-trip**
    - Generator: random skill identity + tool name
    - After add: assert symlink exists at `<cwd>/<toolPath>/<shortName>` AND `GetToolSkills(tool)` contains the full identity
    - **Validates: Requirements 4.8**

  - [ ]* 3.10 Write property test: add does not overwrite conflicting symlinks
    - **Property 10: add does not overwrite conflicting symlinks**
    - Generator: pre-existing symlink from repo-A; attempt to add same short name from repo-B
    - Assert existing symlink target is unchanged and repo-B identity is not in local config for that tool
    - **Validates: Requirements 4.9**

  - [x] 3.11 Rewrite `cmd/remove.go` — unlink-only, no global store deletion
    - Keep file, replace `removeCmd` and `removeAllCmd` implementations
    - Register `removeCmd` with `GroupID: "project"`, optional positional arg `[skill]`, flags `--all` and `--tool string`
    - Guard: if `.skillops/config.json` not found, exit with `"Run 'skillops init' first"`
    - TUI flow (no args): screen 1 shows only skills in local config; screen 2 shows only tools that have the selected skill; screen 3 confirm showing `- <short-name> from <tool>`
    - Non-TUI: `--all` unlinks from all active tools; `--tool` unlinks from specified tools only
    - Never touch `~/.skillops/skills/` — only remove symlinks and call `RemoveSkillFromTool`
    - If symlink doesn't exist on disk, skip without error (idempotent)
    - If target is not a symlink, skip with warning (do not `os.Remove` non-symlinks)
    - Remove `removeAllCmd` registration from `init()`
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8, 5.9, 8.2, 9.1–9.6_

  - [ ]* 3.12 Write unit test: remove is idempotent when symlink absent
    - `TestRemoveSymlink_Idempotent` — call remove on a tool that has no symlink; assert no error
    - _Requirements: 5.6_

  - [ ]* 3.13 Write property test: remove does not touch the global store
    - **Property 11: remove does not touch the global store**
    - Generator: random skill + tool; pre-populate `t.TempDir()` as fake global store
    - After remove, assert the skill directory in the fake global store is unchanged
    - **Validates: Requirements 5.1**

  - [ ]* 3.14 Write property test: remove targets exactly the specified tools
    - **Property 12: remove targets exactly the specified tools**
    - Generator: random active tool set, random target subset
    - After remove, assert identity absent from targeted tools and still present in non-targeted tools
    - **Validates: Requirements 5.4, 5.5**

  - [ ]* 3.15 Write property test: remove round-trip — symlink and local config are consistent
    - **Property 13: remove round-trip**
    - Generator: random skill identity + tool name; pre-create symlink and local config entry
    - After remove: assert no symlink at expected path AND `GetToolSkills(tool)` does not contain the identity
    - **Validates: Requirements 5.7**

- [x] 4. Checkpoint — ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 5. Phase 3 — Visibility: `status` and `sync`

  - [x] 5.1 Create `cmd/status.go` — static lipgloss panel, no interactive TUI
    - Register `statusCmd` with `GroupID: "project"`, no flags
    - Guard: if `.skillops/config.json` not found, exit with `"No local config found. Run 'skillops init' first"`
    - Read local config, sort tools alphabetically
    - For each tool: call `GetToolSkills`, `GetAgenticPath`; for each `"repo/skill"` derive short name, `os.Lstat` symlink path → `◉` if exists, `○` if missing
    - If tool has no skills, render `— no skills linked`
    - Footer: count active tools and symlinks where `Lstat` succeeded
    - Never render the `~/.skillops/skills/...` path — show only `(repo-name)` as context
    - Use only styles from `internal/tui/styles.go`; no one-off lipgloss styles in the command file
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7, 9.4_

  - [ ]* 5.2 Write unit tests for status rendering
    - `TestStatusRender_NoSkillsLinked` — tool with empty skills list renders `"— no skills linked"`
    - `TestStatusRender_NoLocalConfig` — missing config exits with correct message
    - `TestStatusRender_NoGlobalPaths` — rendered output does not contain `~/.skillops/skills/`
    - _Requirements: 6.4, 6.5, 6.7_

  - [ ]* 5.3 Write property test: status renders correct symlink indicators without exposing global paths
    - **Property 14: status renders correct symlink indicators without exposing global paths**
    - Generator: random `LocalConfig` + random boolean per skill (symlink present or absent)
    - Assert `◉` iff symlink exists; assert rendered string does not contain `~/.skillops/skills/`
    - **Validates: Requirements 6.3, 6.5**

  - [ ]* 5.4 Write property test: status renders correct structure for all tools
    - **Property 15: status renders correct structure for all tools**
    - Generator: random `LocalConfig` with 1–9 tools
    - Assert output contains a section for every active tool; footer count matches actual counts
    - **Validates: Requirements 6.2, 6.6**

  - [x] 5.5 Create `cmd/sync.go`
    - Register `syncCmd` with `GroupID: "project"`, no flags
    - Guard: if `.skillops/config.json` not found, exit with `"No local config found. Run 'skillops init' first"`
    - Call `config.ReadSettings()` at start; if `registries` non-empty, use `PullSkillFromURL` for auto-pull
    - For each tool in local config: `GetAgenticPath`, `os.MkdirAll` skills dir
    - For each `"repo/skill"`: follow the full sync flowchart from design.md — check global store → registry fallback via `PullSkillFromURL` → create symlink
    - If skill missing from global store and no registries: record warning `"skill '<repo>/<skill>' not found locally, run 'skillops pull'"`
    - If skill missing and registries configured: try each registry in order via `PullSkillFromURL`; on first success record "auto-pulled"; if all fail record warning `"skill '<repo>/<skill>' not found in any configured registry"`
    - Never remove existing symlinks
    - Render TUI summary: created count + auto-pulled count (omit if 0) + warnings (omit if 0)
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6, 7.7, 7.8, 9.4_

  - [ ]* 5.6 Write unit test: sync no-ops when config missing
    - `TestSyncRender_NoLocalConfig` — missing config exits with correct message
    - _Requirements: 7.8_

  - [ ]* 5.7 Write property test: sync creates all missing symlinks
    - **Property 16: sync creates all missing symlinks**
    - Generator: random `LocalConfig` where all declared skills exist in a fake global store (`t.TempDir()`)
    - After sync, assert every declared symlink exists on disk
    - **Validates: Requirements 7.1, 7.2**

  - [ ]* 5.8 Write property test: sync emits warnings for missing global skills
    - **Property 17: sync emits warnings for missing global skills**
    - Generator: random `LocalConfig` with some skills absent from the fake global store
    - Assert a warning containing the full `"repo/skill"` identity is produced for each missing skill
    - **Validates: Requirements 7.3**

  - [ ]* 5.9 Write property test: sync does not remove existing symlinks
    - **Property 18: sync does not remove existing symlinks**
    - Generator: random pre-existing symlinks in `t.TempDir()`; run sync
    - Assert all pre-existing symlinks still exist after sync completes
    - **Validates: Requirements 7.4**

- [x] 6. Checkpoint — ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 7. Phase 4 — Enterprise Registry support

  - [x] 7.0 Refactor skill extraction into `internal/skills/extract.go`
    - Create `internal/skills/extract.go` (package `skills`)
    - Implement `PullSkillFromURL(repoURL, skillName, destSkillDir string) error`:
      - Clone `repoURL` into temp dir via `git.Clone`
      - Find skill folder using 3-rule discovery (root `SKILL.md` / `skills/<name>/SKILL.md` / `<name>/SKILL.md`)
      - If not found → return error `"skill '<skillName>' not found in repository"`
      - `os.MkdirAll(filepath.Dir(destSkillDir))`
      - `utils.CopyDir(skillPath, destSkillDir)`
      - `SaveMetadata(filepath.Dir(destSkillDir), RepoMetadata{URL: repoURL, SkillName: skillName})`
      - `defer os.RemoveAll(tempDir)`
    - Refactor `cmd/pull.go` `--skill` path to call `skills.PullSkillFromURL` instead of inlining
    - Verify `skillops pull <url> --skill <name>` behavior is unchanged
    - _Requirements: 11.8 (shared code path)_

  - [x] 7.1 Create `internal/config/settings.go`
    - Define `Registry` struct: `URL string`, `Name string` (yaml tags: `url`, `name,omitempty`)
    - Define `Settings` struct: `Registries []Registry` (yaml tag: `registries`)
    - Implement `SettingsPath()` — returns `filepath.Join(ConfigDir, "settings.yaml")`
    - Implement `ReadSettings()` — if file absent return empty `Settings{}` with no error; if malformed log warning to stderr and return empty `Settings{}`
    - Implement `WriteSettings(s Settings) error`
    - Add `ensureSettings()` helper called from `EnsureConfig()` in `config.go` — writes `settings.yaml` with empty registries list if file does not exist
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5, 11.6_

  - [ ]* 7.2 Write unit tests for `settings.go`
    - `TestSettingsPath` — assert path ends with `config/settings.yaml`
    - `TestReadSettings_FileAbsent` — assert returns empty `Settings{}` with no error
    - `TestReadSettings_Malformed` — assert returns empty `Settings{}` and logs warning (no panic)
    - `TestReadSettings_MultipleRegistries` — assert all entries parsed in order
    - _Requirements: 11.3, 11.5, 11.6_

  - [ ]* 7.3 Write property test: Settings YAML round-trip
    - **Property 20 (unit): Settings YAML round-trip**
    - Generator: random `Settings` with 0–5 registries, each with random URL and optional name
    - Assert `WriteSettings` then `ReadSettings` produces a value equal to the original
    - _Requirements: 11.3_

  - [x] 7.4 Update `cmd/status.go` to show registry section
    - Call `config.ReadSettings()` after reading local config
    - If `settings.Registries` is non-empty, append a `Registries:` line to the status panel
    - Display each registry's `name` field; if name is empty display `"registry-<index+1>"`
    - Never display registry URLs
    - _Requirements: 11.13_

  - [ ]* 7.5 Write unit tests for registry display in status
    - `TestStatusRender_RegistriesSection` — assert `Registries:` line appears when registries configured
    - `TestStatusRender_NoRegistriesSection` — assert `Registries:` line absent when settings empty
    - `TestStatusRender_RegistryURLNotShown` — assert rendered output does not contain any `http` substring from registry URLs
    - _Requirements: 11.13_

  - [ ]* 7.6 Write property test: sync auto-pulls from registry when skill missing locally
    - **Property 20: sync auto-pulls from registry when skill missing locally**
    - Generator: random skill absent from fake global store; mock `PullSkillFromURL` to succeed for registry-1
    - After sync, assert skill dir exists in fake global store and symlink exists at expected project path
    - **Validates: Requirements 11.8, 11.9**

  - [ ]* 7.7 Write property test: sync tries registries in order and stops at first success
    - **Property 21: sync tries registries in order and stops at first success**
    - Generator: two registries; first always fails, second always succeeds (mock `PullSkillFromURL`)
    - Assert exactly one clone succeeded (from registry-2) and skill is available
    - **Validates: Requirements 11.10**

  - [ ]* 7.8 Write property test: sync with no registries configured behaves identically to current sync behavior
    - **Property 22: sync with no registries configured behaves identically to current sync behavior**
    - Generator: random `LocalConfig` with some skills absent; empty `Settings{}`
    - Assert warning messages match `"skill '<repo>/<skill>' not found locally, run 'skillops pull'"` exactly
    - **Validates: Requirements 11.5, 11.12**

- [x] 8. Checkpoint — ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [x] 9. Phase 5 — Cleanup: delete deprecated commands and verify build

  - [x] 9.1 Delete `cmd/agentic.go`
    - Remove the file entirely; this removes `agenticCmd`, `agenticManageCmd`, `agenticRemoveSkillCmd`, `agenticRemoveSkillsCmd` and their `init()` registration
    - _Requirements: 8.1, 8.3_

  - [x] 9.2 Remove `ManageAgentics`, `PerformAgenticAction`, `actionModel`, and `NewActionModel` from `internal/tui/tui.go`
    - These are only called from `cmd/agentic.go`; once that file is deleted they are dead code
    - Verify no other callers exist before deleting
    - _Requirements: 8.1_

  - [x] 9.3 Remove `removeAllCmd` from `cmd/remove.go` (if not already done in task 3.11)
    - Delete the `removeAllCmd` var and its `rootCmd.AddCommand(removeAllCmd)` line in `init()`
    - _Requirements: 8.2_

  - [ ]* 9.4 Write property test: quitting TUI models return empty string from View()
    - **Property 19: quitting TUI models return empty string from View()**
    - Generator: all new TUI model types (`initModel`, `toolSelectModel`, any confirm models)
    - Set `quitting = true`; assert `View()` returns `""`
    - **Validates: Requirements 9.2**

  - [x] 9.5 Verify the build compiles cleanly
    - Run `go build -o skillops .` and confirm zero errors
    - Run `go test ./...` and confirm all tests pass
    - _Requirements: 8.1, 8.2, 8.3_

- [x] 10. Final checkpoint — ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for a faster MVP
- Property tests use `pgregory.net/rapid`; each test runs a minimum of 100 iterations
- Filesystem property tests override `config.SkillsDir` and use `t.TempDir()` for isolation
- Tag format in test comments: `// Feature: skillops-v2-redesign, Property N: <property_text>`
- The `remove-all` command deletion (req 8.2) is handled in task 3.11 as part of the remove rewrite; task 9.3 is a safety check only
- The `PullSkillFromURL` refactor in task 7.0 is a prerequisite for the registry auto-pull in task 5.5 and the registry property tests in tasks 7.6–7.8
