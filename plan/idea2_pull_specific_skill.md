# Plan: Idea 2 - Pull Specific Skill

## Objective
Extend `skillops pull <url>` to support a `--skill <skill-name>` option to pull only a single skill from a repository.

## Proposed Changes

### CLI Interface
- `skillops pull <url> --skill <name>`
- Shorthand: `-s`

### Logic
1. If the `--skill` flag is provided:
    - Clone the repository to a temporary directory.
    - Search for the skill folder using several search strategies:
        - **Strategy 1**: Direct subdirectory matching `<skill-name>`.
        - **Strategy 2**: Subdirectory inside a `skills/` folder matching `<skill-name>`.
        - **Strategy 3**: The root directory if it contains `SKILL.md` and the name matches.
    - Once the specific skill folder is found:
        - Identify the repository name from the URL.
        - Create the target structure in the global skills directory: `~/.skillops/skills/<repo-name>/<skill-name>/`.
        - Move the content of the identified skill folder into this target.
    - Cleanup: Remove the temporary clone.

## Case Handling
- **Case 1**: `multiple/skills/repo/skill1/SKILL.md` -> Move `skill1/` to `~/.skillops/skills/repo/skill1/`.
- **Case 2**: `skills/folder/repo/skills/skillA/SKILL.md` -> Move `skills/skillA/` to `~/.skillops/skills/repo/skillA/`.
- **Case 3**: `one/skill/repo/SKILL.md` -> Move root `SKILL.md` (and related files) to `~/.skillops/skills/repo/SKILL.md`.
