package update_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"tewyn/internal/library"
	"tewyn/internal/source"
	"tewyn/internal/update"
)

func TestUpdateCopiesCurrentAndSkipsDownload(t *testing.T) {
	proj := t.TempDir()
	libPath := filepath.Join(t.TempDir(), "library.json")
	commit := strings.Repeat("b", 40)
	if err := library.Save(libPath, library.Library{Groups: []library.Group{{
		Name: "ponytail",
		Skills: []library.Skill{{
			Name:   "ponytail",
			URL:    "https://github.com/DietrichGebert/ponytail/tree/main/skills/ponytail",
			Commit: commit,
		}},
	}}}); err != nil {
		t.Fatal(err)
	}
	writeSkill(t, filepath.Join(library.SkillsDir(libPath), "ponytail"), "fresh")
	projSkill := filepath.Join(proj, ".tewyn", "skills", "ponytail")
	writeSkill(t, projSkill, "stale")
	tree := &scriptTree{byDir: map[string]listResult{
		"skills/ponytail": {commit: commit},
	}}
	var out bytes.Buffer
	if err := update.Run(context.Background(), proj, libPath, tree, &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "copying ponytail\n" {
		t.Fatalf("%q", out.String())
	}
	if body := readSkill(t, projSkill); body != "fresh" {
		t.Fatalf("%q", body)
	}
	if len(tree.downloads) != 0 {
		t.Fatal(tree.downloads)
	}
	if _, err := os.Stat(filepath.Join(proj, ".tewyn", "config.json")); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestUpdateDownloadsWhenCacheFolderMissing(t *testing.T) {
	proj := t.TempDir()
	libPath := filepath.Join(t.TempDir(), "library.json")
	commit := strings.Repeat("b", 40)
	if err := library.Save(libPath, library.Library{Groups: []library.Group{{
		Name: "ponytail",
		Skills: []library.Skill{{
			Name:   "ponytail",
			URL:    "https://github.com/DietrichGebert/ponytail/tree/main/skills/ponytail",
			Commit: commit,
		}},
	}}}); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(library.SkillsDir(libPath), "ponytail")
	projSkill := filepath.Join(proj, ".tewyn", "skills", "ponytail")
	writeSkill(t, projSkill, "stale")
	tree := &scriptTree{byDir: map[string]listResult{
		"skills/ponytail": {commit: commit, body: "fresh"},
	}}
	var out bytes.Buffer
	if err := update.Run(context.Background(), proj, libPath, tree, &out); err != nil {
		t.Fatal(err)
	}
	if out.String() != "downloading ponytail\ncopying ponytail\n" {
		t.Fatalf("%q", out.String())
	}
	if len(tree.downloads) != 1 {
		t.Fatal(tree.downloads)
	}
	if body := readSkill(t, cache); body != "fresh" {
		t.Fatalf("cache %q", body)
	}
	if body := readSkill(t, projSkill); body != "fresh" {
		t.Fatalf("project %q", body)
	}
	lib, err := library.Load(libPath)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := lib.Find("ponytail")
	if !ok || got.Commit != commit {
		t.Fatalf("%+v %v", got, ok)
	}
}

func TestUpdateWarnsAndContinues(t *testing.T) {
	proj := t.TempDir()
	libPath := filepath.Join(t.TempDir(), "library.json")
	oldCommit := strings.Repeat("a", 40)
	newCommit := strings.Repeat("c", 40)
	if err := library.Save(libPath, library.Library{Groups: []library.Group{{
		Name: "skills",
		Skills: []library.Skill{
			{Name: "broken", URL: "https://not-github/broken", Commit: oldCommit},
			{Name: "writing-plans", URL: "https://github.com/obra/superpowers/tree/main/skills/writing-plans", Commit: oldCommit},
			{Name: "brainstorming", URL: "https://github.com/obra/superpowers/tree/main/skills/brainstorming", Commit: oldCommit},
		},
	}}}); err != nil {
		t.Fatal(err)
	}
	skills := library.SkillsDir(libPath)
	writeSkill(t, filepath.Join(skills, "writing-plans"), "keep")
	writeSkill(t, filepath.Join(proj, ".tewyn", "skills", "missing"), "keep-missing")
	writeSkill(t, filepath.Join(proj, ".tewyn", "skills", "broken"), "keep-broken")
	writeSkill(t, filepath.Join(proj, ".tewyn", "skills", "writing-plans"), "keep")
	writeSkill(t, filepath.Join(proj, ".tewyn", "skills", "brainstorming"), "old")
	tree := &scriptTree{byDir: map[string]listResult{
		"skills/writing-plans": {err: source.ErrPathChanged},
		"skills/brainstorming": {commit: newCommit, body: "new"},
	}}
	var out bytes.Buffer
	if err := update.Run(context.Background(), proj, libPath, tree, &out); err != nil {
		t.Fatal(err)
	}
	want := "downloading brainstorming\ncopying brainstorming\nwarnings\n  broken  invalid url\n  missing  not in library\n  writing-plans  path changed\n"
	if out.String() != want {
		t.Fatalf("%q", out.String())
	}
	cache := filepath.Join(skills, "brainstorming")
	if len(tree.downloads) != 1 || filepath.Dir(tree.downloads[0]) != filepath.Dir(cache) {
		t.Fatalf("dests %v", tree.downloads)
	}
	if _, err := os.Stat(filepath.Join(skills, "missing")); !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if body := readSkill(t, filepath.Join(proj, ".tewyn", "skills", "missing")); body != "keep-missing" {
		t.Fatalf("missing %q", body)
	}
	if body := readSkill(t, filepath.Join(proj, ".tewyn", "skills", "broken")); body != "keep-broken" {
		t.Fatalf("broken %q", body)
	}
	if body := readSkill(t, filepath.Join(skills, "writing-plans")); body != "keep" {
		t.Fatalf("cache %q", body)
	}
	if body := readSkill(t, filepath.Join(proj, ".tewyn", "skills", "writing-plans")); body != "keep" {
		t.Fatalf("project %q", body)
	}
	if body := readSkill(t, cache); body != "new" {
		t.Fatalf("cache %q", body)
	}
	if body := readSkill(t, filepath.Join(proj, ".tewyn", "skills", "brainstorming")); body != "new" {
		t.Fatalf("project %q", body)
	}
	lib, err := library.Load(libPath)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := lib.Find("brainstorming")
	if !ok || got.Commit != newCommit {
		t.Fatalf("%+v %v", got, ok)
	}
	if got, ok = lib.Find("writing-plans"); !ok || got.Commit != oldCommit {
		t.Fatalf("%+v", got)
	}
}

func TestUpdateDownloadErrorLeavesFolders(t *testing.T) {
	proj := t.TempDir()
	libPath := filepath.Join(t.TempDir(), "library.json")
	oldCommit := strings.Repeat("a", 40)
	newCommit := strings.Repeat("c", 40)
	if err := library.Save(libPath, library.Library{Groups: []library.Group{{
		Name: "skills",
		Skills: []library.Skill{
			{Name: "ponytail", URL: "https://github.com/DietrichGebert/ponytail/tree/main/skills/ponytail", Commit: oldCommit},
			{Name: "writing-plans", URL: "https://github.com/obra/superpowers/tree/main/skills/writing-plans", Commit: oldCommit},
		},
	}}}); err != nil {
		t.Fatal(err)
	}
	skills := library.SkillsDir(libPath)
	writeSkill(t, filepath.Join(skills, "writing-plans"), "keep")
	writeSkill(t, filepath.Join(skills, "ponytail"), "old")
	writeSkill(t, filepath.Join(proj, ".tewyn", "skills", "writing-plans"), "keep")
	writeSkill(t, filepath.Join(proj, ".tewyn", "skills", "ponytail"), "stale")
	tree := &scriptTree{byDir: map[string]listResult{
		"skills/ponytail":      {commit: newCommit, body: "fresh"},
		"skills/writing-plans": {commit: newCommit, body: "partial", dlErr: errors.New("download failed")},
	}}
	var out bytes.Buffer
	if err := update.Run(context.Background(), proj, libPath, tree, &out); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, part := range []string{"downloading writing-plans\n", "copying ponytail\n", "warnings\n  writing-plans  download failed\n"} {
		if !strings.Contains(got, part) {
			t.Fatalf("missing %q in %q", part, got)
		}
	}
	if body := readSkill(t, filepath.Join(skills, "writing-plans")); body != "keep" {
		t.Fatalf("cache %q", body)
	}
	if body := readSkill(t, filepath.Join(proj, ".tewyn", "skills", "writing-plans")); body != "keep" {
		t.Fatalf("project %q", body)
	}
	if body := readSkill(t, filepath.Join(proj, ".tewyn", "skills", "ponytail")); body != "fresh" {
		t.Fatalf("ponytail %q", body)
	}
}

func writeSkill(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readSkill(t *testing.T, dir string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

type listResult struct {
	commit string
	err    error
	body   string
	dlErr  error
}

type scriptTree struct {
	byDir     map[string]listResult
	downloads []string
}

func (s *scriptTree) List(_ context.Context, _, _, dir string) (string, []string, error) {
	r := s.byDir[dir]
	return r.commit, nil, r.err
}

func (s *scriptTree) Download(_ context.Context, _, _, dir, dest string) error {
	s.downloads = append(s.downloads, dest)
	r := s.byDir[dir]
	if err := os.MkdirAll(dest, 0o750); err != nil {
		return err
	}
	body := r.body
	if body == "" {
		body = "new"
	}
	if err := os.WriteFile(filepath.Join(dest, "SKILL.md"), []byte(body), 0o600); err != nil {
		return err
	}
	return r.dlErr
}
