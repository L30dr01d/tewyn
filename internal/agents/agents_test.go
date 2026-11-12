package agents_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"tewyn/internal/agents"
)

func TestLinksListsAgentsAlphabetically(t *testing.T) {
	got := agents.Links()
	want := []agents.Link{
		{Name: "Claude", Rel: ".claude/skills"},
		{Name: "Codex / Goose", Rel: ".agents/skills"},
		{Name: "Crush", Rel: ".crush/skills"},
		{Name: "Cursor", Rel: ".cursor/skills"},
		{Name: "Grok Build", Rel: ".grok/skills"},
		{Name: "OpenCode", Rel: ".opencode/skills"},
		{Name: "Pi", Rel: ".pi/skills"},
	}
	if len(got) != len(want) {
		t.Fatalf("%+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%+v", got)
		}
	}
}

func TestApplyLinksOpenCodeCrushAndPi(t *testing.T) {
	dir := t.TempDir()
	if err := agents.Apply(dir, map[string]bool{"OpenCode": true, "Crush": true, "Pi": true, "Grok Build": true}); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{".opencode/skills", ".crush/skills", ".pi/skills", ".grok/skills"} {
		got, err := os.Readlink(filepath.Join(dir, rel))
		if err != nil || got != "../.tewyn/skills" {
			t.Fatalf("%s %q %v", rel, got, err)
		}
	}
}

func TestApplyCreatesRelativeSymlink(t *testing.T) {
	dir := t.TempDir()
	if err := agents.Apply(dir, map[string]bool{"Claude": true, "Cursor": false, "Codex / Goose": true}); err != nil {
		t.Fatal(err)
	}
	got, err := os.Readlink(filepath.Join(dir, ".claude", "skills"))
	if err != nil || got != "../.tewyn/skills" {
		t.Fatalf("link %q %v", got, err)
	}
	state, err := agents.State(dir)
	if err != nil || !state["Claude"] || state["Cursor"] || !state["Codex / Goose"] {
		t.Fatal(state, err)
	}
	if err := agents.Apply(dir, map[string]bool{"Claude": false}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(dir, ".claude", "skills")); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestRealDirectoryIsLeftAlone(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".claude", "skills")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "keep.txt"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".cursor"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/somewhere/else", filepath.Join(dir, ".cursor", "skills")); err != nil {
		t.Fatal(err)
	}
	err := agents.Apply(dir, map[string]bool{"Claude": true, "Cursor": false, "Codex / Goose": true})
	var linkErr agents.LinkError
	if !errors.As(err, &linkErr) || linkErr.Agent != "Claude" {
		t.Fatal(err)
	}
	body, err := fs.ReadFile(os.DirFS(skillDir), "keep.txt")
	if err != nil || string(body) != "keep" {
		t.Fatal("claude dir removed")
	}
	other, err := os.Readlink(filepath.Join(dir, ".cursor", "skills"))
	if err != nil || other != "/somewhere/else" {
		t.Fatalf("cursor link changed: %s", other)
	}
	codex, err := os.Readlink(filepath.Join(dir, ".agents", "skills"))
	if err != nil || codex != "../.tewyn/skills" {
		t.Fatal(codex)
	}
}
