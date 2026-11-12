package main

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
)

type fakeTree struct {
	paths []string
	dests *[]string
}

func (f fakeTree) List(_ context.Context, _, _, _ string) (string, []string, error) {
	return strings.Repeat("a", 40), f.paths, nil
}

func (f fakeTree) Download(_ context.Context, _, _, _, dest string) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dest, "SKILL.md"), []byte("cached"), 0o644); err != nil {
		return err
	}
	if f.dests != nil {
		*f.dests = append(*f.dests, dest)
	}
	return nil
}

// failSecondDownload writes every dest, then fails on the second call.
type failSecondDownload struct {
	paths []string
	calls int
}

func (f *failSecondDownload) List(context.Context, string, string, string) (string, []string, error) {
	return strings.Repeat("a", 40), f.paths, nil
}

func (f *failSecondDownload) Download(_ context.Context, _, _, _, dest string) error {
	f.calls++
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dest, "SKILL.md"), []byte("partial"), 0o644); err != nil {
		return err
	}
	if f.calls > 1 {
		return errors.New("download failed")
	}
	return nil
}

// parentRequiredTree fails the way git Download does when SkillsDir is missing.
type parentRequiredTree struct {
	paths []string
}

func (f parentRequiredTree) List(context.Context, string, string, string) (string, []string, error) {
	return strings.Repeat("a", 40), f.paths, nil
}

func (parentRequiredTree) Download(_ context.Context, _, _, _, dest string) error {
	parent := filepath.Dir(dest)
	if _, err := os.Stat(parent); err != nil {
		return err
	}
	if err := os.Mkdir(dest, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dest, "SKILL.md"), []byte("cached"), 0o644)
}

type errTree struct{}

func (errTree) List(context.Context, string, string, string) (string, []string, error) {
	return "", nil, source.ErrPathChanged
}

func (errTree) Download(context.Context, string, string, string, string) error {
	return nil
}

func TestAddSkillsWritesGroup(t *testing.T) {
	home := t.TempDir()
	getenv := func(k string) string {
		if k == "HOME" {
			return home
		}
		return ""
	}
	tree := fakeTree{paths: []string{"brainstorming/SKILL.md", "writing-plans/SKILL.md"}}
	var out, errb bytes.Buffer
	code := Main([]string{"--add-skills", "https://github.com/obra/superpowers/tree/main/skills"}, t.TempDir(), &out, &errb, getenv, tree)
	if code != 0 || errb.Len() != 0 {
		t.Fatalf("code %d stderr %q", code, errb.String())
	}
	want := "downloading brainstorming\ndownloading writing-plans\nadded group superpowers\n  brainstorming\n  writing-plans\nsaved to ~/.config/tewyn/library.json\n"
	if out.String() != want {
		t.Fatalf("%q", out.String())
	}
	lib, err := library.Load(home + "/.config/tewyn/library.json")
	if err != nil || lib.Groups[0].Skills[1].URL != "https://github.com/obra/superpowers/tree/main/skills/writing-plans" {
		t.Fatal(lib, err)
	}
	commit := strings.Repeat("a", 40)
	for _, skill := range lib.Groups[0].Skills {
		if skill.Commit != commit {
			t.Fatalf("commit %s %q", skill.Name, skill.Commit)
		}
	}
	cached, err := os.ReadFile(home + "/.config/tewyn/skills/brainstorming/SKILL.md")
	if err != nil || string(cached) != "cached" {
		t.Fatalf("cache %q %v", cached, err)
	}
	code = Main([]string{"--add-skills", "https://github.com/obra/superpowers/tree/main/skills"}, t.TempDir(), &out, &errb, getenv, tree)
	if code == 0 || errb.String() != "error: group superpowers already exists\n" {
		t.Fatalf("code %d stderr %q", code, errb.String())
	}
}

func TestAddSkillsCreatesCacheDirBeforeDownload(t *testing.T) {
	home := t.TempDir()
	getenv := func(k string) string {
		if k == "HOME" {
			return home
		}
		return ""
	}
	tree := parentRequiredTree{paths: []string{"brainstorming/SKILL.md"}}
	var out, errb bytes.Buffer
	code := Main([]string{"--add-skills", "https://github.com/obra/superpowers/tree/main/skills"}, t.TempDir(), &out, &errb, getenv, tree)
	if code != 0 || errb.Len() != 0 {
		t.Fatalf("code %d stderr %q out %q", code, errb.String(), out.String())
	}
	cached, err := os.ReadFile(filepath.Join(home, ".config", "tewyn", "skills", "brainstorming", "SKILL.md"))
	if err != nil || string(cached) != "cached" {
		t.Fatalf("cache %q %v", cached, err)
	}
}

func TestAddSkillsNoSkills(t *testing.T) {
	home := t.TempDir()
	getenv := func(k string) string {
		if k == "HOME" {
			return home
		}
		return ""
	}
	var out, errb bytes.Buffer
	code := Main([]string{"--add-skills", "https://github.com/obra/superpowers"}, t.TempDir(), &out, &errb, getenv, fakeTree{})
	if code == 0 || errb.String() != "error: no skills at that path\n" || out.Len() != 0 {
		t.Fatalf("code %d out %q err %q", code, out.String(), errb.String())
	}
	if _, err := os.Stat(home + "/.config/tewyn/library.json"); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestAddSkillsDuplicateName(t *testing.T) {
	home := t.TempDir()
	getenv := func(k string) string {
		if k == "HOME" {
			return home
		}
		return ""
	}
	path := home + "/.config/tewyn/library.json"
	lib := library.Library{Groups: []library.Group{{
		Name: "other",
		Skills: []library.Skill{{
			Name:   "brainstorming",
			URL:    "https://github.com/example/other/tree/main/skills/brainstorming",
			Commit: strings.Repeat("b", 40),
		}},
	}}}
	if err := library.Save(path, lib); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var dests []string
	tree := fakeTree{paths: []string{"brainstorming/SKILL.md", "writing-plans/SKILL.md"}, dests: &dests}
	var out, errb bytes.Buffer
	code := Main([]string{"--add-skills", "https://github.com/obra/superpowers/tree/main/skills"}, t.TempDir(), &out, &errb, getenv, tree)
	if code != 1 || errb.String() != "error: skill brainstorming appears twice\n" {
		t.Fatalf("code %d stderr %q", code, errb.String())
	}
	if len(dests) != 0 {
		t.Fatalf("download called: %v", dests)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(before) {
		t.Fatalf("library changed: %s", after)
	}
}

func TestAddSkillsDownloadErrorRemovesNewFolders(t *testing.T) {
	home := t.TempDir()
	getenv := func(k string) string {
		if k == "HOME" {
			return home
		}
		return ""
	}
	keep := filepath.Join(home, ".config", "tewyn", "skills", "keep", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(keep), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keep, []byte("stay"), 0o644); err != nil {
		t.Fatal(err)
	}
	tree := &failSecondDownload{paths: []string{"brainstorming/SKILL.md", "writing-plans/SKILL.md"}}
	var out, errb bytes.Buffer
	code := Main([]string{"--add-skills", "https://github.com/obra/superpowers/tree/main/skills"}, t.TempDir(), &out, &errb, getenv, tree)
	if code != 1 || errb.String() != "error: download failed\n" {
		t.Fatalf("code %d stderr %q out %q", code, errb.String(), out.String())
	}
	if _, err := os.Stat(home + "/.config/tewyn/library.json"); !os.IsNotExist(err) {
		t.Fatalf("library.json: %v", err)
	}
	for _, name := range []string{"brainstorming", "writing-plans"} {
		dir := filepath.Join(home, ".config", "tewyn", "skills", name)
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	got, err := os.ReadFile(keep)
	if err != nil || string(got) != "stay" {
		t.Fatalf("keep %q %v", got, err)
	}
}

func TestAddSkillsPathChanged(t *testing.T) {
	home := t.TempDir()
	getenv := func(k string) string {
		if k == "HOME" {
			return home
		}
		return ""
	}
	var out, errb bytes.Buffer
	code := Main([]string{"--add-skills", "https://github.com/obra/superpowers/tree/main/skills"}, t.TempDir(), &out, &errb, getenv, errTree{})
	if code != 1 || errb.String() != "error: no skills at that path\n" || out.Len() != 0 {
		t.Fatalf("code %d out %q err %q", code, out.String(), errb.String())
	}
	if _, err := os.Stat(home + "/.config/tewyn/library.json"); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

type failDownload struct{}

func (failDownload) List(context.Context, string, string, string) (string, []string, error) {
	return strings.Repeat("b", 40), []string{"SKILL.md"}, nil
}

func (failDownload) Download(context.Context, string, string, string, string) error {
	return errors.New("download failed")
}

func TestVersionAndHelp(t *testing.T) {
	help := "" +
		"tewyn 1.0.0\n" +
		"usage: tewyn [--add-skills <url>|-u|-v|-h]\n" +
		"\n" +
		"  tewyn                         open the project TUI\n" +
		"  tewyn --add-skills <url>      add a GitHub skill tree to the library\n" +
		"  tewyn -u                      refresh cached skills and copy them into the project\n" +
		"  tewyn -v, --version           print the version\n" +
		"  tewyn -h, --help              print this help\n"
	usage := "error: usage: tewyn [--add-skills <url>|-u|-v|-h]\n"
	cases := []struct {
		args []string
		code int
		out  string
		err  string
	}{
		{[]string{"-v"}, 0, "tewyn 1.0.0\n", ""},
		{[]string{"--version"}, 0, "tewyn 1.0.0\n", ""},
		{[]string{"-h"}, 0, help, ""},
		{[]string{"--help"}, 0, help, ""},
		{[]string{"-v", "-h"}, 2, "", usage},
		{[]string{"-h", "extra"}, 2, "", usage},
		{[]string{"-V"}, 2, "", usage},
		{[]string{"version"}, 2, "", usage},
		{[]string{"--v"}, 2, "", usage},
	}
	for _, tc := range cases {
		var out, errb bytes.Buffer
		code := Main(tc.args, t.TempDir(), &out, &errb, func(string) string { return "" }, nil)
		if code != tc.code || out.String() != tc.out || errb.String() != tc.err {
			t.Fatalf("%q: code %d out %q err %q", tc.args, code, out.String(), errb.String())
		}
	}
}

func TestUpdateCommandExitCodes(t *testing.T) {
	home := t.TempDir()
	getenv := func(k string) string {
		if k == "HOME" {
			return home
		}
		return ""
	}
	proj := t.TempDir()
	var out, errb bytes.Buffer
	code := Main([]string{"--update-skills"}, proj, &out, &errb, getenv, fakeTree{})
	if code != 2 || errb.String() != "error: usage: tewyn [--add-skills <url>|-u|-v|-h]\n" || out.Len() != 0 {
		t.Fatalf("usage: code %d out %q err %q", code, out.String(), errb.String())
	}
	out.Reset()
	errb.Reset()
	code = Main([]string{"-u"}, proj, &out, &errb, getenv, fakeTree{})
	if code != 1 || errb.String() != "error: no project config\n" || out.Len() != 0 {
		t.Fatalf("missing: code %d out %q err %q", code, out.String(), errb.String())
	}
	if err := os.Mkdir(filepath.Join(proj, ".tewyn"), 0o755); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errb.Reset()
	code = Main([]string{"-u"}, proj, &out, &errb, getenv, fakeTree{})
	if code != 0 || out.String() != "no skills\n" || errb.Len() != 0 {
		t.Fatalf("empty: code %d out %q err %q", code, out.String(), errb.String())
	}
	write := filepath.Join(proj, ".tewyn", "skills", "writing-plans", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(write), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(write, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := library.Save(filepath.Join(home, ".config", "tewyn", "library.json"), library.Library{Groups: []library.Group{{
		Name: "superpowers",
		Skills: []library.Skill{{
			Name:   "writing-plans",
			URL:    "https://github.com/obra/superpowers/tree/main/skills/writing-plans",
			Commit: strings.Repeat("a", 40),
		}},
	}}}); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errb.Reset()
	code = Main([]string{"-u"}, proj, &out, &errb, getenv, failDownload{})
	if code != 0 || !strings.Contains(out.String(), "warnings\n") || !strings.Contains(out.String(), "writing-plans  download failed\n") {
		t.Fatalf("fail: code %d out %q err %q", code, out.String(), errb.String())
	}
	body, err := os.ReadFile(write)
	if err != nil || string(body) != "keep" {
		t.Fatalf("project %q %v", body, err)
	}
}

func TestMainRejectsDuplicateSkillURL(t *testing.T) {
	home := t.TempDir()
	proj := t.TempDir()
	getenv := func(k string) string {
		if k == "HOME" {
			return home
		}
		return ""
	}
	lib := library.Library{Groups: []library.Group{
		{Name: "a", Skills: []library.Skill{{Name: "one", URL: "https://example/one"}}},
		{Name: "b", Skills: []library.Skill{{Name: "two", URL: "https://example/one"}}},
	}}
	if err := library.Save(home+"/.config/tewyn/library.json", lib); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	code := Main(nil, proj, &out, &errb, getenv, fakeTree{})
	if code != 1 || errb.String() != "error: skill url https://example/one is in more than one group\n" {
		t.Fatalf("code %d stderr %q", code, errb.String())
	}
	if _, err := os.Stat(filepath.Join(proj, ".tewyn")); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}
