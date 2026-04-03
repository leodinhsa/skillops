package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty string", "", true},
		{"path traversal", "../etc/passwd", true},
		{"double dot in middle", "foo/../bar", true},
		{"absolute unix path", "/etc/passwd", true},
		{"absolute windows path", `\etc\passwd`, true},
		{"valid simple name", "my-skill", false},
		{"valid with underscore", "my_skill", false},
		{"valid with dot", "skill.v2", false},
		{"valid repo/skill", "repo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestCopyDir(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")

	// Create source structure
	if err := os.MkdirAll(filepath.Join(src, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(src, "SKILL.md"):         "# Skill",
		filepath.Join(src, "subdir", "foo.txt"): "hello",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := CopyDir(src, dst); err != nil {
		t.Fatalf("CopyDir failed: %v", err)
	}

	// Verify files exist with correct content
	for path, content := range files {
		rel, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, rel)
		got, err := os.ReadFile(dstPath)
		if err != nil {
			t.Errorf("missing file %s: %v", dstPath, err)
			continue
		}
		if string(got) != content {
			t.Errorf("file %s: got %q, want %q", dstPath, got, content)
		}
	}
}

func TestCopyDir_NestedDirs(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")

	if err := os.MkdirAll(filepath.Join(src, "a", "b", "c"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "a", "b", "c", "deep.txt"), []byte("deep"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := CopyDir(src, dst); err != nil {
		t.Fatalf("CopyDir failed: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dst, "a", "b", "c", "deep.txt"))
	if err != nil {
		t.Fatalf("deep file not copied: %v", err)
	}
	if string(got) != "deep" {
		t.Errorf("got %q, want %q", got, "deep")
	}
}
