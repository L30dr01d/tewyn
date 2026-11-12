package project_test

import (
	"os"
	"path/filepath"
	"testing"
	"tewyn/internal/project"
)

func TestCopyDirReplacesOnlyAfterSuccess(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	if err := os.MkdirAll(filepath.Join(src, "scripts"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "scripts", "run.sh"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dst, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "SKILL.md"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := project.CopyDir(filepath.Join(root, "missing"), dst); err == nil {
		t.Fatal("missing src copied")
	}
	old, err := os.ReadFile(filepath.Join(dst, "SKILL.md"))
	if err != nil || string(old) != "old" {
		t.Fatal(string(old), err)
	}
	if err = project.CopyDir(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dst, "scripts", "run.sh"))
	if err != nil || string(got) != "new" {
		t.Fatal(string(got), err)
	}
	if _, err = os.Stat(filepath.Join(dst, "SKILL.md")); !os.IsNotExist(err) {
		t.Fatal("old file remains", err)
	}
	info, err := os.Lstat(dst)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatal(info, err)
	}
}
