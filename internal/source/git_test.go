package source_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"tewyn/internal/source"
)

func TestListAndDownloadFromLocalRepo(t *testing.T) {
	repo := initRepo(t, map[string]string{
		"skills/ponytail/SKILL.md":       "# ponytail\n",
		"skills/ponytail/scripts/run.sh": "echo hi\n",
	})
	tree := source.New()
	commit, paths, err := tree.List(context.Background(), repo, "main", "skills/ponytail")
	if err != nil || len(commit) != 40 || !contains(paths, "SKILL.md") || !contains(paths, "scripts/run.sh") {
		t.Fatalf("commit %q paths %v err %v", commit, paths, err)
	}
	dest := filepath.Join(t.TempDir(), "ponytail")
	if err := tree.Download(context.Background(), repo, commit, "skills/ponytail", dest); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Clean(filepath.Join(dest, "scripts", "run.sh")))
	if err != nil || string(body) != "echo hi\n" {
		t.Fatalf("%q %v", body, err)
	}
}

func TestDownloadFailureLeavesDest(t *testing.T) {
	repo := initRepo(t, map[string]string{"skills/ponytail/SKILL.md": "new\n"})
	dest := filepath.Join(t.TempDir(), "ponytail")
	if err := os.MkdirAll(dest, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "SKILL.md"), []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := source.New().Download(context.Background(), repo, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef", "skills/ponytail", dest)
	if err == nil {
		t.Fatal("expected error")
	}
	body, _ := os.ReadFile(filepath.Clean(filepath.Join(dest, "SKILL.md")))
	if string(body) != "old\n" {
		t.Fatalf("dest replaced: %q", body)
	}
}

func TestListCancelIsNotPathChanged(t *testing.T) {
	repo := initRepo(t, map[string]string{"skills/ponytail/SKILL.md": "x\n"})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	source.SetAfterGit(func(args []string) {
		if slices.Contains(args, "rev-parse") {
			cancel()
		}
	})
	t.Cleanup(func() { source.SetAfterGit(nil) })
	_, _, err := source.New().List(ctx, repo, "main", "skills/ponytail")
	if errors.Is(err, source.ErrPathChanged) || !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestDownloadCancelLeavesDest(t *testing.T) {
	repo := initRepo(t, map[string]string{"skills/ponytail/SKILL.md": "new\n"})
	commit, _, err := source.New().List(context.Background(), repo, "main", "skills/ponytail")
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "ponytail")
	if err := os.MkdirAll(dest, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "SKILL.md"), []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	source.SetAfterGit(func(args []string) {
		if slices.Contains(args, "archive") {
			cancel()
		}
	})
	t.Cleanup(func() { source.SetAfterGit(nil) })
	err = source.New().Download(ctx, repo, commit, "skills/ponytail", dest)
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(filepath.Clean(filepath.Join(dest, "SKILL.md")))
	if string(body) != "old\n" {
		t.Fatalf("dest replaced: %q", body)
	}
}

func TestListMissingPath(t *testing.T) {
	repo := initRepo(t, map[string]string{"README.md": "x\n"})
	_, _, err := source.New().List(context.Background(), repo, "main", "skills/missing")
	if !errors.Is(err, source.ErrPathChanged) {
		t.Fatal(err)
	}
}

func initRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.CommandContext(context.Background(), "git")
		cmd.Args = append(cmd.Args, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=tewyn",
			"GIT_AUTHOR_EMAIL=tewyn@example.com",
			"GIT_COMMITTER_NAME=tewyn",
			"GIT_COMMITTER_EMAIL=tewyn@example.com",
			"GIT_CONFIG_COUNT=1",
			"GIT_CONFIG_KEY_0=commit.gpgsign",
			"GIT_CONFIG_VALUE_0=false",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s: %v: %s", args, err, out)
		}
	}
	git("init", "-b", "main")
	for path, body := range files {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	git("add", ".")
	git("commit", "-m", "init")
	return dir
}

func contains(paths []string, want string) bool {
	return slices.Contains(paths, want)
}
