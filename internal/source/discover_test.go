package source_test

import (
	"errors"
	"testing"
	"tewyn/internal/source"
)

func TestDiscoverSuperpowersChildren(t *testing.T) {
	raw := "https://github.com/obra/superpowers/tree/main/skills"
	paths := []string{
		"brainstorming/SKILL.md",
		"brainstorming/scripts/run.sh",
		"writing-plans/SKILL.md",
		"README.md",
		"notes/readme.md",
		"nested/deeper/SKILL.md",
	}
	g, err := source.Discover(raw, paths)
	if err != nil {
		t.Fatal(err)
	}
	if g.Name != "superpowers" || len(g.Skills) != 2 {
		t.Fatalf("%+v", g)
	}
	if g.Skills[0].Name != "brainstorming" || g.Skills[0].URL != "https://github.com/obra/superpowers/tree/main/skills/brainstorming" {
		t.Fatalf("%+v", g.Skills[0])
	}
	if g.Skills[1].Name != "writing-plans" {
		t.Fatal(g.Skills[1])
	}
}

func TestDiscoverPonytailSkill(t *testing.T) {
	raw := "https://github.com/DietrichGebert/ponytail/tree/main/skills/ponytail"
	g, err := source.Discover(raw, []string{"SKILL.md", "scripts/run.sh"})
	if err != nil || len(g.Skills) != 1 || g.Name != "ponytail" || g.Skills[0].Name != "ponytail" || g.Skills[0].URL != raw {
		t.Fatalf("%+v %v", g, err)
	}
}

func TestFolderWithSkillMDIgnoresChildren(t *testing.T) {
	raw := "https://github.com/DietrichGebert/ponytail/tree/main/skills/ponytail"
	g, err := source.Discover(raw, []string{"SKILL.md", "child/SKILL.md"})
	if err != nil || len(g.Skills) != 1 || g.Skills[0].URL != raw {
		t.Fatalf("%+v %v", g, err)
	}
}

func TestDiscoverNoSkills(t *testing.T) {
	_, err := source.Discover("https://github.com/obra/superpowers", nil)
	if !errors.Is(err, source.ErrNoSkills) {
		t.Fatal(err)
	}
	_, err = source.Discover("https://github.com/obra/superpowers/tree/main/skills", []string{"README.md"})
	if !errors.Is(err, source.ErrNoSkills) {
		t.Fatal(err)
	}
}
