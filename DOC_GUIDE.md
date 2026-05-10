# SkillOps: Hướng Dẫn Sử Dụng

SkillOps là CLI tool viết bằng Go giúp quản lý các "skills" (script/capability hỗ trợ AI) cho nhiều Agentic IDE (Claude Code, Kiro, Cursor, Windsurf, v.v.) thông qua cơ chế **symlink**.

Thay vì sao chép thủ công skill vào từng thư mục IDE, SkillOps lưu trữ tập trung tại `~/.skillops/skills/` và tạo symlink vào đúng vị trí trong dự án.

---

## Cài đặt

### Homebrew (macOS & Linux)
```bash
brew tap leodinhsa/skillops
brew install skillops
```

### Từ mã nguồn (yêu cầu Go 1.25+)
```bash
go build -o skillops .
mv skillops /usr/local/bin/
```

---

## Khái niệm cốt lõi

- **Skill**: Thư mục chứa file `SKILL.md`, được nhận diện theo định dạng `repo_name/skill_name` (ví dụ: `my-repo/logger`).
- **Agentic**: IDE/môi trường agent với đường dẫn skill riêng (ví dụ: `kiro` → `.kiro/skills`).
- **Global store**: `~/.skillops/skills/` — nơi lưu tất cả repo skill đã pull về.
- **Local config**: `.skillops/config.json` trong thư mục dự án — nguồn sự thật cho project, nên commit vào git.

---

## Quy trình làm việc cơ bản

### 1. Khai báo IDE cho dự án (`init`)
Chạy ở thư mục gốc của dự án để chọn các IDE muốn hỗ trợ:
```bash
skillops init
```
TUI hiện ra danh sách IDE. Dùng `Space` để chọn/bỏ chọn, `Enter` để xác nhận. SkillOps sẽ tạo thư mục tương ứng và cập nhật `.skillops/config.json`.

### 2. Pull skill từ GitHub (`pull`)
```bash
# Pull toàn bộ repo
skillops pull https://github.com/user/my-skills

# Pull một skill cụ thể (tiết kiệm dung lượng)
skillops pull https://github.com/user/my-skills --skill logger
```

### 3. Gắn skill vào IDE (`add`)
```bash
# TUI tương tác — chọn skill và IDE
skillops add

# Gắn skill cụ thể vào tất cả IDE đang active
skillops add logger --all

# Gắn vào một hoặc nhiều IDE cụ thể
skillops add logger --tool kiro
skillops add logger --tool kiro,claude-code
```
Có thể truyền tên ngắn (`logger`) hoặc full identity (`my-repo/logger`).

### 4. Xem trạng thái dự án (`status`)
```bash
skillops status
```
Hiển thị tất cả IDE đang active, skill nào đã được link (◉) và chưa được link (○).

### 5. Khôi phục symlink (`sync`)
Dùng khi clone dự án về máy mới hoặc symlink bị mất:
```bash
skillops sync
```
Đọc `.skillops/config.json` và tạo lại toàn bộ symlink. Nếu skill chưa có trong global store và đã cấu hình registry, sẽ tự động pull về.

---

## Quản lý skill

### Xem danh sách skill đã pull (`list`)
```bash
skillops list
```
Giao diện TUI duyệt danh sách repo và skill. Dùng `Space` để copy tên skill vào clipboard.

### Cập nhật skill (`update`)
```bash
# Cập nhật tất cả
skillops update

# Cập nhật một skill cụ thể
skillops update --skill logger
```

### Gỡ skill khỏi IDE (`remove`)
```bash
# TUI tương tác
skillops remove

# Gỡ khỏi tất cả IDE
skillops remove logger --all

# Gỡ khỏi IDE cụ thể
skillops remove logger --tool kiro
```

---

## Cấu hình toàn cục (`config`)

Cấu hình lưu tại `~/.skillops/config/agentics.yaml` — ánh xạ tên IDE sang đường dẫn thư mục skill tương đối.

```bash
# Thêm IDE mới
skillops config add-agentic -n my-ide -p .my-ide/skills

# Cập nhật đường dẫn
skillops config update-agentic -n my-ide -p .new-path/skills

# Xóa IDE
skillops config remove-agentic -n my-ide
```

---

## Xem phiên bản

```bash
skillops version
```

---

## Xử lý sự cố

**Lỗi "No local config found"**: Chạy `skillops init` trước, sau đó `skillops sync` để khôi phục link.

**Lỗi "open /dev/tty"**: SkillOps dùng TUI tương tác, cần chạy trong terminal thực (không chạy được trong môi trường non-interactive như CI).

**Symlink bị hỏng**: Chạy `skillops sync` để tạo lại, hoặc `skillops add` để gắn lại từng skill.

**Conflict tên skill**: Nếu hai repo có skill cùng tên ngắn, SkillOps sẽ cảnh báo và bỏ qua — không ghi đè thầm lặng.

---

*Cập nhật theo phiên bản hiện tại của codebase.*
