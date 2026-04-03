# SkillOps v2 Redesign Plan

## Mục tiêu cốt lõi

Giữ nguyên giá trị cốt lõi: centralize skills tại `~/.skillops/skills/` và symlink tới các IDE folder.
Cải thiện UX bằng cách giảm friction, làm rõ mental model, và thống nhất behavior giữa các lệnh.

---

## Phân tích vấn đề hiện tại

1. Mental model không rõ ràng — `agentic` + `agentic manage` là 2 bước tách biệt, user phải nhớ tên IDE
2. `remove` inconsistent — cùng tên nhưng một cái unlink, một cái xóa global
3. Danh sách IDE quá dài (30+), nhiều cái obscure, gây noise trong TUI
4. Không có local project state — không biết project đang dùng IDE/skill nào nếu không check thủ công
5. Không có lệnh xem trạng thái project hiện tại

---

## Mental model mới

```
# Setup (1 lần per machine)
skillops pull <url>        → tải skill repo về global store

# Setup (1 lần per project)
skillops init              → khai báo IDE nào dùng trong project này

# Hàng ngày
skillops add <skill>       → link skill vào IDE đã khai báo
skillops remove <skill>    → unlink skill khỏi IDE (không xóa global)
skillops status            → xem project đang dùng skill/IDE nào
skillops list              → xem tất cả skill đã pull về global
```

---

## Bảng thay đổi lệnh (đầy đủ)

| Lệnh hiện tại | Flags | Trạng thái | Lệnh v2 | Ghi chú |
|---|---|---|---|---|
| `skillops pull <url>` | `-s, --skill` | Giữ nguyên | `skillops pull <url>` | Không đổi |
| `skillops list` | _(none)_ | Giữ nguyên | `skillops list` | Không đổi — xem global store |
| `skillops update` | `-s, --skill` | Giữ nguyên | `skillops update` | Không đổi |
| `skillops version` | _(none)_ | Giữ nguyên | `skillops version` | Không đổi |
| `skillops config add-agentic` | `-n, -p` | Giữ nguyên tạm | `skillops config add-agentic` | Không phát triển thêm |
| `skillops config remove-agentic` | `-n` | Giữ nguyên tạm | `skillops config remove-agentic` | Không phát triển thêm |
| `skillops config update-agentic` | `-n, -p` | Giữ nguyên tạm | `skillops config update-agentic` | Không phát triển thêm |
| `skillops agentic` | _(none)_ | **Xóa** | `skillops init` | Rewrite hoàn toàn |
| `skillops agentic manage <ide>` | _(none)_ | **Xóa** | `skillops add` | Gộp vào add |
| `skillops agentic remove-skill <ide> <skill>` | _(none)_ | **Xóa** | `skillops remove <skill> --tool <ide>` | |
| `skillops agentic remove-skills <ide>` | _(none)_ | **Xóa** | `skillops remove --tool <ide> --all` | |
| `skillops remove` | `-s, --skill` (required) | **Rewrite** | `skillops remove [skill]` | Chỉ unlink, không xóa global |
| `skillops remove-all` | _(none)_ | **Xóa** | _(không có)_ | Không có lệnh tương đương |
| _(chưa có)_ | | **Mới** | `skillops init` | Khai báo IDE cho project |
| _(chưa có)_ | | **Mới** | `skillops add` | Link skill vào IDE |
| _(chưa có)_ | | **Mới** | `skillops status` | Xem trạng thái project |
| _(chưa có)_ | | **Mới** | `skillops sync` | Restore symlinks từ local config |

> Lệnh xóa global store (`skillops purge`) không có trong v2. Chưa có demand thực tế.
> Nếu cần, user tự xóa thư mục `~/.skillops/skills/<repo>` bằng tay.

---

## Chi tiết từng phần

---

### PHẦN 1: Trim danh sách IDE mặc định

Chỉ giữ 9 IDE phổ biến, comment out phần còn lại (không xóa):

```
claude-code    → .claude/skills
cursor         → .cursor/skills
windsurf       → .windsurf/skills
kiro           → .kiro/skills
gemini-cli     → .gemini/skills
goose          → .goose/skills
github-copilot → .github/skills
opencode       → .agents/skills
antigravity    → .agent/skills
```

File: `internal/config/config.go` — thu gọn `defaultAgentics` map, comment out phần còn lại.

User đã có config cũ sẽ không bị ảnh hưởng — `EnsureConfig()` chỉ thêm key mới, không xóa key cũ.

---

### PHẦN 2: Local project config (`internal/config/localconfig.go`)

File `.skillops/config.json` tại project root. Nên commit vào git.

**Schema** (Option A — lưu cả skill list per tool để `sync` hoạt động đầy đủ):

```json
{
  "version": "1",
  "tools": {
    "claude-code": ["repo-a/auth-agent", "repo-a/logging-agent"],
    "kiro": ["repo-a/auth-agent"]
  }
}
```

Skill được lưu theo format `repo/skill` đầy đủ để tránh nhầm lẫn khi có skill trùng tên từ nhiều repo khác nhau. Khi tạo symlink, dùng short name (phần sau `/`) làm tên symlink. Nếu 2 skill từ 2 repo khác nhau có cùng short name → báo conflict rõ ràng, không silently overwrite.

Local config là source of truth. Symlink trên disk là derived state.

**Functions** (cùng package `config`, không tạo package mới):

```go
func LocalConfigPath() string
func ReadLocalConfig() (LocalConfig, error)
func WriteLocalConfig(cfg LocalConfig) error
func GetActiveTools() ([]string, error)
func GetToolSkills(tool string) ([]string, error)
func AddSkillToTool(tool, repoSkill string) error   // gọi khi `add`
func RemoveSkillFromTool(tool, repoSkill string) error // gọi khi `remove`
func SetActiveTools(tools []string) error
```

---

### PHẦN 3: `skillops init`

Khai báo IDE nào được dùng trong project hiện tại.

**Behavior**:
1. Đọc global config → lấy danh sách IDE có thể chọn (9 IDE đã trim)
2. Nếu `.skillops/config.json` đã tồn tại → pre-check các IDE đã lưu
3. TUI checklist chọn/bỏ chọn IDE (tái dùng `checklistModel` trong `tui.go`)
4. Confirm screen hiện summary: IDE nào được thêm, IDE nào bị bỏ
5. Apply:
   - Ghi danh sách IDE vào `.skillops/config.json`
   - Tạo thư mục skills cho IDE mới thêm (e.g., `.kiro/skills/`)
   - Với IDE bị bỏ: xóa symlinks trong thư mục skills của nó, xóa entry khỏi local config
   - Không xóa thư mục IDE (`.kiro/`, `.claude/`) — chỉ dọn symlinks

Idempotent. Chạy lại bất cứ lúc nào cũng an toàn.

**Files**: `cmd/init.go` (mới), cập nhật `internal/tui/tui.go` (`checklistModel.applyChanges`)

---

### PHẦN 4: `skillops add`

Link skill vào các IDE đang active trong project.

Prerequisite: `.skillops/config.json` phải tồn tại. Nếu không → `"Run 'skillops init' first"`.

**Modes**:
```bash
skillops add                              # TUI: chọn skill → chọn IDE
skillops add <skill-name>                 # TUI: chỉ chọn IDE (skill đã biết)
skillops add <skill-name> --all           # Link vào tất cả IDE active
skillops add <skill-name> --tool kiro
skillops add <skill-name> --tool kiro,claude-code
```

**TUI flow** (không có args):
- Màn hình 1: Checklist chọn skill từ global store
- Màn hình 2: Checklist chọn IDE targets (chỉ hiện IDE active theo local config)
- Màn hình confirm: summary trước khi apply

**Logic**:
1. Đọc local config → lấy active tools
2. Với mỗi tool → lấy path từ global config
3. Tạo symlink: `<cwd>/<tool-path>/<skill-short-name>` → `~/.skillops/skills/<repo>/<skill>`
4. Cập nhật local config: thêm `repo/skill` vào tool entry
5. Tạo thư mục nếu chưa có

**Conflict handling**: Nếu short name đã tồn tại (từ repo khác) → báo lỗi rõ, không overwrite.

**File**: `cmd/add.go` (mới)

---

### PHẦN 5: `skillops remove` (rewrite)

Chỉ unlink symlink. Không xóa global store.

```bash
skillops remove                              # TUI: chọn skill → chọn IDE để unlink
skillops remove <skill-name>                 # TUI: chọn IDE để unlink
skillops remove <skill-name> --all           # Unlink khỏi tất cả IDE active
skillops remove <skill-name> --tool kiro
skillops remove <skill-name> --tool kiro,claude-code
```

**Logic**:
1. Xóa symlink tại `<cwd>/<tool-path>/<skill-short-name>`
2. Cập nhật local config: xóa `repo/skill` khỏi tool entry
3. Idempotent — nếu symlink không tồn tại thì bỏ qua

**File**: `cmd/remove.go` (rewrite hoàn toàn)

---

### PHẦN 6: `skillops status`

Xem trạng thái project hiện tại. TUI đẹp, không cần nhanh.

```bash
skillops status
```

**Output mẫu** (TUI với lipgloss, không phải plain text):

```
╭─────────────────────────────────────────╮
│           PROJECT STATUS                │
│  /path/to/my-project                    │
├─────────────────────────────────────────┤
│                                         │
│  claude-code                            │
│    ◉ auth-agent        (repo-a)         │
│    ◉ logging-agent     (repo-a)         │
│                                         │
│  kiro                                   │
│    ◉ auth-agent        (repo-a)         │
│    ○ logging-agent     not linked       │
│                                         │
│  cursor                                 │
│    — no skills linked                   │
│                                         │
│  2 tools active • 3 skills linked       │
╰─────────────────────────────────────────╯
```

Không hiển thị đường dẫn symlink thực tế (`~/.skillops/skills/...`). Chỉ hiện repo name và skill name.

`◉` = đang linked, `○` = có trong local config nhưng symlink bị mất (cần `sync`), `—` = IDE active nhưng chưa có skill nào.

**File**: `cmd/status.go` (mới), thêm TUI model mới trong `internal/tui/`

---

### PHẦN 7: `skillops sync`

Restore symlinks theo local config. Dùng sau khi clone repo mới về máy.

```bash
skillops sync
```

**Behavior**:
1. Đọc `.skillops/config.json` → lấy tools và skill list per tool
2. Với mỗi tool → với mỗi `repo/skill` trong list:
   - Kiểm tra skill có tồn tại trong `~/.skillops/skills/` không
   - Nếu có → tạo symlink (nếu chưa có)
   - Nếu không → báo warning: `"skill 'repo/auth-agent' not found locally, run 'skillops pull'"` 
3. Không xóa symlinks thừa (đó là việc của `remove`)
4. Output: TUI đẹp hiển thị kết quả

**Không trigger `skillops update`**. Sync chỉ tạo symlinks, không pull code mới.

**File**: `cmd/sync.go` (mới)

---

## Thứ tự triển khai

```
Phase 1 — Foundation
  1.1  internal/config/config.go       trim defaultAgentics (9 IDE)
  1.2  internal/config/localconfig.go  local config R/W, schema, helper functions

Phase 2 — Core commands
  2.1  cmd/init.go                     + cập nhật tui.go checklistModel
  2.2  cmd/add.go                      + TUI 2-màn-hình
  2.3  cmd/remove.go                   rewrite hoàn toàn

Phase 3 — Visibility
  3.1  cmd/status.go                   + TUI model mới
  3.2  cmd/sync.go

Phase 4 — Cleanup
  4.1  cmd/agentic.go                  xóa file
  4.2  cmd/remove.go (remove-all)      đã xóa trong 2.3
```

---

## Files thay đổi

```
NEW:
  cmd/init.go
  cmd/add.go
  cmd/status.go
  cmd/sync.go
  internal/config/localconfig.go

MODIFIED:
  internal/config/config.go     trim defaultAgentics
  internal/tui/tui.go           cập nhật checklistModel.applyChanges cho init
  cmd/remove.go                 rewrite hoàn toàn

DELETED:
  cmd/agentic.go

UNCHANGED:
  cmd/pull.go
  cmd/list.go
  cmd/update.go
  cmd/version.go
  cmd/config.go
  internal/skills/skills.go
  internal/symlink/symlink.go
  internal/utils/utils.go
  internal/git/git.go
  internal/tui/styles.go
  internal/tui/list.go
```
