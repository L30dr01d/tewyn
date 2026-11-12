package library_test

import (
	"testing"
	"tewyn/internal/library"
)

func TestPathUsesXDGThenHome(t *testing.T) {
	p, err := library.Path(func(k string) string {
		if k == "XDG_CONFIG_HOME" {
			return "/cfg"
		}
		return ""
	})
	if err != nil || p != "/cfg/tewyn/library.json" {
		t.Fatalf("xdg: %q %v", p, err)
	}
	p, err = library.Path(func(k string) string {
		if k == "HOME" {
			return "/home/me"
		}
		return ""
	})
	if err != nil || p != "/home/me/.config/tewyn/library.json" {
		t.Fatalf("home: %q %v", p, err)
	}
}

func TestLoadSaveAndRules(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/library.json"
	lib, err := library.Load(path)
	if err != nil || len(lib.Groups) != 0 {
		t.Fatalf("missing file: %+v %v", lib, err)
	}
	lib.Groups = []library.Group{{
		Name: "superpowers",
		Skills: []library.Skill{{
			Name:   "brainstorming",
			URL:    "https://github.com/obra/superpowers/tree/main/skills/brainstorming",
			Commit: "abc",
		}},
	}, {Name: "empty"}}
	if err := library.Save(path, lib); err != nil {
		t.Fatal(err)
	}
	got, err := library.Load(path)
	if err != nil || got.Visible().Groups[0].Skills[0].Name != "brainstorming" || got.Visible().Groups[0].Skills[0].Commit != "abc" {
		t.Fatal(got, err)
	}
	if len(got.Visible().Groups) != 1 {
		t.Fatalf("empty group visible: %+v", got.Visible())
	}
	dup := got
	dup.Groups = append(dup.Groups, got.Groups[0])
	if err := dup.Check(); err == nil || err.Error() != "group superpowers appears twice" {
		t.Fatal(err)
	}
	urlDup := library.Library{Groups: []library.Group{
		{Name: "a", Skills: []library.Skill{{Name: "one", URL: "https://example/one"}}},
		{Name: "b", Skills: []library.Skill{{Name: "two", URL: "https://example/one"}}},
	}}
	if err := urlDup.Check(); err == nil || err.Error() != "skill url https://example/one is in more than one group" {
		t.Fatal(err)
	}
	sameName := library.Library{Groups: []library.Group{
		{Name: "a", Skills: []library.Skill{{Name: "brainstorming", URL: "https://example/a"}}},
		{Name: "b", Skills: []library.Skill{{Name: "brainstorming", URL: "https://example/b"}}},
	}}
	if err := sameName.Check(); err == nil || err.Error() != "skill brainstorming appears twice" {
		t.Fatal(err)
	}
	if err := lib.Add(library.Group{Name: "superpowers"}); err == nil || err.Error() != "group superpowers already exists" {
		t.Fatal(err)
	}
	if len(lib.Groups) != 2 {
		t.Fatal("failed Add changed the library")
	}
	fresh := library.Library{}
	if err := fresh.Add(library.Group{Name: "nope"}); err == nil || err.Error() != "group nope is empty" {
		t.Fatal(err)
	}
	if len(fresh.Groups) != 0 {
		t.Fatal("empty group was stored")
	}
}

func TestSkillsDirAndFind(t *testing.T) {
	if library.SkillsDir("/cfg/tewyn/library.json") != "/cfg/tewyn/skills" {
		t.Fatal(library.SkillsDir("/cfg/tewyn/library.json"))
	}
	lib := library.Library{Groups: []library.Group{{
		Name:   "superpowers",
		Skills: []library.Skill{{Name: "brainstorming", URL: "https://example/b", Commit: "abc"}},
	}}}
	got, ok := lib.Find("brainstorming")
	if !ok || got.Commit != "abc" {
		t.Fatal(got, ok)
	}
	if _, ok := lib.Find("missing"); ok {
		t.Fatal("found missing")
	}
}
