package project_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"tewyn/internal/library"
	"tewyn/internal/project"
)

func TestApplySwapSameName(t *testing.T) {
	dir := t.TempDir()
	cache := t.TempDir()
	newURL := "https://github.com/other/skills/tree/main/skills/brainstorming"
	commit := strings.Repeat("b", 40)
	skillDir := filepath.Join(dir, ".tewyn", "skills", "brainstorming")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeCache(t, cache, "brainstorming")
	res, err := project.Apply(context.Background(), dir, cache, project.Selection{
		Delete: []string{"brainstorming"},
		Add:    []library.Skill{{Name: "brainstorming", URL: newURL, Commit: commit}},
	})
	if err != nil || len(res.Added) != 1 {
		t.Fatal(res, err)
	}
	body, err := os.ReadFile(filepath.Clean(filepath.Join(skillDir, "SKILL.md")))
	if err != nil || string(body) != "new" {
		t.Fatalf("swap left %q %v", body, err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".tewyn", "config.json")); !os.IsNotExist(err) {
		t.Fatal("config.json created", err)
	}
}

func TestApplyNameClashKeepsUniqueAdd(t *testing.T) {
	dir := t.TempDir()
	cache := t.TempDir()
	commit := strings.Repeat("b", 40)
	if err := os.MkdirAll(filepath.Join(dir, ".tewyn", "skills", "brainstorming"), 0o750); err != nil {
		t.Fatal(err)
	}
	plans := filepath.Join(dir, ".tewyn", "skills", "writing-plans")
	if err := os.MkdirAll(plans, 0o750); err != nil {
		t.Fatal(err)
	}
	writeCache(t, cache, "ponytail")
	_, err := project.Apply(context.Background(), dir, cache, project.Selection{
		Delete: []string{"writing-plans"},
		Add: []library.Skill{
			{Name: "ponytail", URL: "https://github.com/DietrichGebert/ponytail/tree/main/skills/ponytail", Commit: commit},
			{Name: "brainstorming", URL: "https://github.com/other/skills/tree/main/review/brainstorming", Commit: strings.Repeat("c", 40)},
		},
	})
	clash, ok := err.(project.NameClashError)
	if !ok || clash.Name != "brainstorming" || clash.Error() != "two skills named \"brainstorming\" cannot both exist" {
		t.Fatal(err)
	}
	names, err := project.SkillNames(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "brainstorming" || names[1] != "ponytail" {
		t.Fatal(names)
	}
	if _, err := os.Stat(filepath.Join(dir, ".gitignore")); !os.IsNotExist(err) {
		t.Fatal("gitignore was written")
	}
	if _, err := os.Stat(plans); !os.IsNotExist(err) {
		t.Fatal("writing-plans remains", err)
	}
	body, err := os.ReadFile(filepath.Clean(filepath.Join(dir, ".tewyn", "skills", "ponytail", "SKILL.md")))
	if err != nil || string(body) != "new" {
		t.Fatalf("ponytail %q %v", body, err)
	}
}

func TestApplyMissingConfigCreates(t *testing.T) {
	dir := t.TempDir()
	cache := t.TempDir()
	commit := strings.Repeat("c", 40)
	writeCache(t, cache, "ponytail")
	res, err := project.Apply(context.Background(), dir, cache, project.Selection{
		Add: []library.Skill{{Name: "ponytail", URL: "https://github.com/DietrichGebert/ponytail/tree/main/skills/ponytail", Commit: commit}},
	})
	if err != nil || len(res.Added) != 1 {
		t.Fatal(res, err)
	}
	names, err := project.SkillNames(dir)
	if err != nil || len(names) != 1 || names[0] != "ponytail" {
		t.Fatal(names, err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".tewyn", "config.json")); !os.IsNotExist(err) {
		t.Fatal("config.json created", err)
	}
	body, err := os.ReadFile(filepath.Clean(filepath.Join(dir, ".tewyn", "skills", "ponytail", "SKILL.md")))
	if err != nil || string(body) != "new" {
		t.Fatalf("ponytail %q %v", body, err)
	}
}

func TestApplyNotifiesEachAddedSkill(t *testing.T) {
	dir := t.TempDir()
	cache := t.TempDir()
	writeCache(t, cache, "ponytail")
	writeCache(t, cache, "brainstorming")
	var got []string
	_, err := project.ApplyNotify(context.Background(), dir, cache, project.Selection{
		Add: []library.Skill{
			{Name: "ponytail", URL: "https://github.com/DietrichGebert/ponytail/tree/main/skills/ponytail", Commit: strings.Repeat("c", 40)},
			{Name: "brainstorming", URL: "https://github.com/obra/superpowers/tree/main/skills/brainstorming", Commit: strings.Repeat("d", 40)},
		},
	}, func(name string) {
		body, readErr := os.ReadFile(filepath.Clean(filepath.Join(dir, ".tewyn", "skills", name, "SKILL.md")))
		if readErr != nil || string(body) != "new" {
			t.Fatalf("notified %s before copy %q %v", name, body, readErr)
		}
		got = append(got, name)
	})
	if err != nil || len(got) != 2 || got[0] != "ponytail" || got[1] != "brainstorming" {
		t.Fatal(err, got)
	}
	names, err := project.SkillNames(dir)
	if err != nil || len(names) != 2 || names[0] != "brainstorming" || names[1] != "ponytail" {
		t.Fatal(names, err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".tewyn", "config.json")); !os.IsNotExist(err) {
		t.Fatal("config.json created", err)
	}
}

func TestApplyDotDotDoesNotEscape(t *testing.T) {
	dir := t.TempDir()
	cache := t.TempDir()
	keep := filepath.Join(dir, ".tewyn", "skills", "keep")
	if err := os.MkdirAll(keep, 0o750); err != nil {
		t.Fatal(err)
	}
	_, err := project.Apply(context.Background(), dir, cache, project.Selection{
		Delete: []string{".."},
		Add:    []library.Skill{{Name: "..", URL: "https://github.com/other/skills/tree/main/skills/brainstorming", Commit: strings.Repeat("b", 40)}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "SKILL.md")); !os.IsNotExist(statErr) {
		t.Fatal("wrote next to .tewyn", statErr)
	}
	if _, statErr := os.Stat(keep); statErr != nil {
		t.Fatal("escaped delete", statErr)
	}
}

func TestApplyEmptySelectionLeavesConfig(t *testing.T) {
	dir := t.TempDir()
	cache := t.TempDir()
	if _, err := project.Apply(context.Background(), dir, cache, project.Selection{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".tewyn")); !os.IsNotExist(err) {
		t.Fatal("tewyn dir created", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".tewyn"), 0o750); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ".tewyn", "config.json")
	raw := []byte("{}\n")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := project.Apply(context.Background(), dir, cache, project.Selection{}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Clean(path))
	if err != nil || string(got) != string(raw) {
		t.Fatalf("config changed %q %v", got, err)
	}
}

func TestApplyCreatesSkillsParent(t *testing.T) {
	dir := t.TempDir()
	cache := t.TempDir()
	commit := strings.Repeat("d", 40)
	writeCache(t, cache, "ponytail")
	res, err := project.Apply(context.Background(), dir, cache, project.Selection{
		Add: []library.Skill{{Name: "ponytail", URL: "https://github.com/DietrichGebert/ponytail/tree/main/skills/ponytail", Commit: commit}},
	})
	if err != nil || len(res.Added) != 1 {
		t.Fatal(res, err)
	}
	names, err := project.SkillNames(dir)
	if err != nil || len(names) != 1 || names[0] != "ponytail" {
		t.Fatal(names, err)
	}
	body, err := os.ReadFile(filepath.Clean(filepath.Join(dir, ".tewyn", "skills", "ponytail", "SKILL.md")))
	if err != nil || string(body) != "new" {
		t.Fatalf("ponytail %q %v", body, err)
	}
}

func TestApplyMissingCache(t *testing.T) {
	dir := t.TempDir()
	cache := t.TempDir()
	_, err := project.Apply(context.Background(), dir, cache, project.Selection{
		Add: []library.Skill{{Name: "ponytail", URL: "https://github.com/DietrichGebert/ponytail/tree/main/skills/ponytail", Commit: strings.Repeat("c", 40)}},
	})
	if err == nil || err.Error() != "skill ponytail is not in the cache" {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".tewyn")); !os.IsNotExist(statErr) {
		t.Fatal("tewyn dir created", statErr)
	}
}

func TestApplyDownloadFailRemovesEmptyDir(t *testing.T) {
	dir := t.TempDir()
	cache := t.TempDir()
	_, err := project.Apply(context.Background(), dir, cache, project.Selection{
		Add: []library.Skill{{Name: "ponytail", URL: "https://github.com/DietrichGebert/ponytail/tree/main/skills/ponytail", Commit: strings.Repeat("e", 40)}},
	})
	if err == nil || err.Error() != "skill ponytail is not in the cache" {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".tewyn")); !os.IsNotExist(statErr) {
		t.Fatal("empty tewyn remains", statErr)
	}
	keep := filepath.Join(dir, ".tewyn", "skills", "keep")
	if err = os.MkdirAll(keep, 0o750); err != nil {
		t.Fatal(err)
	}
	if _, err = project.Apply(context.Background(), dir, cache, project.Selection{
		Add: []library.Skill{{Name: "ponytail", URL: "https://github.com/DietrichGebert/ponytail/tree/main/skills/ponytail", Commit: strings.Repeat("e", 40)}},
	}); err == nil {
		t.Fatal("expected error")
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".tewyn", "config.json")); !os.IsNotExist(statErr) {
		t.Fatal("config.json created", statErr)
	}
	if _, statErr := os.Stat(keep); statErr != nil {
		t.Fatal(statErr)
	}
}

func TestApplyCancelledBeforeCopy(t *testing.T) {
	dir := t.TempDir()
	cache := t.TempDir()
	writeCache(t, cache, "ponytail")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := project.Apply(ctx, dir, cache, project.Selection{
		Add: []library.Skill{{Name: "ponytail", URL: "https://github.com/DietrichGebert/ponytail/tree/main/skills/ponytail", Commit: strings.Repeat("c", 40)}},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, ".tewyn", "skills", "ponytail")); !os.IsNotExist(statErr) {
		t.Fatal(statErr)
	}
}

func TestSkillNamesMissingDoesNotCreate(t *testing.T) {
	dir := t.TempDir()
	names, err := project.SkillNames(dir)
	if err != nil || len(names) != 0 {
		t.Fatal(names, err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".tewyn")); !os.IsNotExist(err) {
		t.Fatal("tewyn dir created")
	}
}

func writeCache(t *testing.T, cache, name string) {
	t.Helper()
	dir := filepath.Join(cache, name)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
}
