# SkillOps - Product Overview

`skillops` is a CLI tool for managing AI agent "skills" (modular capabilities/scripts) across multiple Agentic IDEs (Cursor, Windsurf, Kiro, Roo, Claude Code, etc.).

## Core Problem
Each IDE stores skills in a different directory. Manually syncing a shared skill repository across IDEs is tedious and error-prone.

## Solution
- A central global skills directory (`~/.skillops/skills/`) stores all pulled skill repos
- Symlinks map individual skills into the correct IDE-specific paths within a project
- A YAML config (`~/.skillops/config/agentics.yaml`) maps IDE names to their relative skill folder paths

## Key Concepts
- **Skill**: A directory containing a `SKILL.md` file, discovered from pulled repos
- **Agentic**: An IDE/agent environment with a known relative path for skills (e.g., `kiro` → `.kiro/skills`)
- **Symlink**: The mechanism linking a global skill into a project's agentic directory
- Skills are identified as `repo_name/skill_name` (e.g., `my-repo/logger`)
