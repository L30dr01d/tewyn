package source

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitErrorTextIsStderr(t *testing.T) {
	_, err := git(context.Background(), t.TempDir(), "rev-parse", "--verify", "tewyn-missing")
	var exit *exec.ExitError
	if err == nil || !errors.As(err, &exit) {
		t.Fatal(err)
	}
	if err.Error() != strings.TrimSpace(string(exit.Stderr)) || err.Error() == "" {
		t.Fatalf("%q stderr %q", err.Error(), exit.Stderr)
	}
}

func TestReplaceRestoresOldWhenSwapFails(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "skill")
	old := dest + ".old"
	if err := os.Mkdir(old, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "SKILL.md"), []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := replace(filepath.Join(dir, "missing"), dest); err == nil {
		t.Fatal("expected error")
	}
	body, err := os.ReadFile(filepath.Clean(filepath.Join(dest, "SKILL.md")))
	if err != nil || string(body) != "old\n" {
		t.Fatalf("%q %v", body, err)
	}
	if _, err = os.Stat(old); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}
