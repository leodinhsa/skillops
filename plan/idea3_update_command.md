# Plan: Idea 3 - Update Command

## Objective
Implement `skillops update` to refresh installed skills and `skillops update --skill <name>` for specific updates.

## Proposed Changes

### New Command: `update`
- `skillops update`: Updates all repositories in the global skills directory.
- `skillops update --skill <name>`: Updates only the repository that contains the specified skill.

### Implementation Strategy
1. **Identify Repositories**:
    - Iterate over each directory in `~/.skillops/skills/`.
    - Verify if it's a Git repository by checking for a `.git` subfolder.
2. **Execute Update**:
    - For each valid repo, change directory to the repo root and execute `git pull --rebase`.
    - If `--skill` is provided:
        - Use the existing discovery logic to find which repository folder owns the skill.
        - Perform the `git pull` only in that specific folder.
3. **Feedback**:
    - Provide a TUI progress bar or a simple list showing the update status of each repo.

## Note on URL Persistence
- Since `skillops pull` clones the original repository, the remote URL is automatically stored in the `.git/config` of each folder. No additional metadata storage is strictly necessary, but we should ensure `git pull` works for both SSH and HTTPS clones.
