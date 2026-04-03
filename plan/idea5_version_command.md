# Idea 5: Version Command Implementation Plan

## Goal
Implement a `version` command and a `--version` and `-v` flags for the `skillops` CLI to help users track their installed version and stay updated with frequent releases.

## Proposed Changes

### 1. Define Version Variable
- **File**: `internal/config/config.go`
- **Action**: Add a `Version` variable (default `v1.1.0-dev`). Using a variable allows injecting the exact version during build time using `ldflags`.

### 2. Implement `version` Command
- **File**: `cmd/version.go` [NEW]
- **Action**: Create a new Cobra command that prints the version with a premium TUI style.

### 3. Add `--version` and `-v` Flags to Root
- **File**: `cmd/root.go`
- **Action**: Set `rootCmd.Version = config.Version`. Cobra automatically handles `--version` and `-v` if `Version` is set on the root command.

### 4. Update Homebrew Formula
- **File**: `Formula/skillops.rb`
- **Action**: Update the `install` block to use `ldflags` for version injection.
- **Code**:
  ```ruby
  def install
    system "go", "build", "-ldflags", "-X skillops/internal/config.Version=v#{version}", "-o", bin/"skillops", "main.go"
  end
  ```

## TUI Design
The version output should look premium:
```bash
 ⚙️ SKILLOPS VERSION 
                     
Current Version: v1.1.0
Latest Version:  v1.2.0 (Update available!)
```

## Verification Plan
1. Run `skillops version` and verify output.
2. Run `skillops --version` and verify output.
3. Run `skillops -v` and verify output.
4. Ensure the formatting matches the existing CLI style.
