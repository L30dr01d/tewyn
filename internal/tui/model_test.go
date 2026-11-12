package tui_test

import (
	"strings"
	"testing"
	"tewyn/internal/library"
	"tewyn/internal/tui"
)

func TestGroupToggleAndPanels(t *testing.T) {
	lib := library.Library{Groups: []library.Group{
		{Name: "superpowers", Skills: []library.Skill{
			{Name: "brainstorming", URL: "https://github.com/obra/superpowers/tree/main/skills/brainstorming"},
			{Name: "writing-plans", URL: "https://github.com/obra/superpowers/tree/main/skills/writing-plans"},
		}},
		{Name: "empty"},
		{Name: "ponytail", Skills: []library.Skill{
			{Name: "ponytail", URL: "https://github.com/DietrichGebert/ponytail/tree/main/skills/ponytail"},
		}},
	}}
	m := tui.New(lib, []string{"writing-plans", "orphan"}, map[string]bool{"Claude": true})
	if strings.Contains(m.View(), "empty") {
		t.Fatal("empty group shown")
	}
	if !strings.Contains(m.View(), "writing-plans") || !strings.Contains(m.View(), "orphan") {
		t.Fatal(m.View())
	}
	m.Handle(tui.KeyG) // cursor starts on the first group heading, left panel
	sel := m.Selection()
	if len(sel.Add) != 2 || sel.Add[0].Name != "brainstorming" {
		t.Fatalf("%+v", sel)
	}
	m.Handle(tui.KeyG)
	if len(m.Selection().Add) != 0 {
		t.Fatal("second g should clear the group")
	}
	m.Handle(tui.KeyRight)
	m.Handle(tui.KeyDown) // onto writing-plans
	m.Handle(tui.KeySpace)
	if len(m.Selection().Delete) != 1 || m.Selection().Delete[0] != "writing-plans" {
		t.Fatal(m.Selection())
	}
	m.Handle(tui.KeyTab)
	if m.Screen() != "agents" || !m.AgentWant()["Claude"] {
		t.Fatal(m.Screen(), m.AgentWant())
	}
	m.Handle(tui.KeyG)
	if !m.AgentWant()["Claude"] {
		t.Fatal("g changed agents")
	}
}

func TestOrphanToggleDoesNotSelectTheOther(t *testing.T) {
	lib := library.Library{Groups: []library.Group{{Name: "review", Skills: []library.Skill{
		{Name: "brainstorming", URL: "https://github.com/other/skills/tree/main/review/brainstorming"},
	}}}}
	m := tui.New(lib, []string{"orphan-a", "orphan-b"}, nil)
	m.Handle(tui.KeyRight)
	if strings.Contains(m.View(), "[x] orphan-a") || len(m.Selection().Delete) != 0 {
		t.Fatal("unchecked row in Delete", m.Selection())
	}
	m.Handle(tui.KeyG)
	if len(m.Selection().Delete) != 1 || m.Selection().Delete[0] != "orphan-a" {
		t.Fatal(m.Selection().Delete)
	}
	view := m.View()
	if !strings.Contains(view, "[\x1b[31m✕\x1b[0m] orphan-a") || strings.Contains(view, "[\x1b[31m✕\x1b[0m] orphan-b") {
		t.Fatal(view)
	}
	if !strings.Contains(view, "[ ] orphan-b") {
		t.Fatal(view)
	}
}

func TestSkillsView(t *testing.T) {
	lib := library.Library{Groups: []library.Group{{Name: "review", Skills: []library.Skill{
		{Name: "brainstorming", URL: "https://github.com/other/skills/tree/main/review/brainstorming"},
	}}}}
	m := tui.New(lib, []string{"brainstorming"}, nil)
	view := m.View()
	first, _, _ := strings.Cut(view, "\n")
	if !strings.HasPrefix(first, "available") || !strings.Contains(first, "current") || strings.Contains(first, "\n") {
		t.Fatal(view)
	}
	if strings.Contains(view, "saved") || strings.Contains(view, "this repo") || !strings.Contains(view, "[ ] brainstorming") {
		t.Fatal(view)
	}
	if !strings.HasSuffix(view, "↑↓ move  ←→ panel  space select  g group  a add and remove  tab agents  q quit") {
		t.Fatal(view)
	}
	m.Handle(tui.KeyDown)
	m.Handle(tui.KeySpace)
	if !strings.Contains(m.View(), "[\x1b[32m✓\x1b[0m] brainstorming") {
		t.Fatal(m.View())
	}
	m.Handle(tui.KeyRight)
	m.Handle(tui.KeyDown)
	m.Handle(tui.KeySpace)
	if !strings.Contains(m.View(), "[\x1b[31m✕\x1b[0m] brainstorming") {
		t.Fatal(m.View())
	}
	m.Handle(tui.KeyA)
	if !strings.Contains(m.View(), "0 / 1 downloaded\n") {
		t.Fatal(m.View())
	}
}

func TestAgentsScreen(t *testing.T) {
	lib := library.Library{Groups: []library.Group{{Name: "review", Skills: []library.Skill{
		{Name: "brainstorming", URL: "https://github.com/other/skills/tree/main/review/brainstorming"},
	}}}}
	linked := map[string]bool{"Claude": true}
	m := tui.New(lib, nil, linked)
	linked["Claude"] = false
	if !m.AgentWant()["Claude"] {
		t.Fatal("shared map")
	}
	m.Handle(tui.KeyDown)
	m.Handle(tui.KeyTab)
	if m.View() != "symlinks\n> [x] Claude .claude/skills\n  [ ] Codex / Goose .agents/skills\n  [ ] Crush .crush/skills\n  [ ] Cursor .cursor/skills\n  [ ] Grok Build .grok/skills\n  [ ] OpenCode .opencode/skills\n  [ ] Pi .pi/skills\n↑↓ move  space select  a link  tab skills  q quit" {
		t.Fatal(m.View())
	}
	m.Handle(tui.KeyDown)
	m.Handle(tui.KeySpace)
	m.Handle(tui.KeyG)
	m.Handle(tui.KeyLeft)
	m.Handle(tui.KeyRight)
	if m.View() != "symlinks\n  [x] Claude .claude/skills\n> [x] Codex / Goose .agents/skills\n  [ ] Crush .crush/skills\n  [ ] Cursor .cursor/skills\n  [ ] Grok Build .grok/skills\n  [ ] OpenCode .opencode/skills\n  [ ] Pi .pi/skills\n↑↓ move  space select  a link  tab skills  q quit" {
		t.Fatal(m.View())
	}
	m.Handle(tui.KeyTab)
	m.Handle(tui.KeySpace)
	if len(m.Selection().Add) != 1 || m.Selection().Add[0].Name != "brainstorming" {
		t.Fatal(m.Selection())
	}
	m.Handle(tui.KeyTab)
	if !strings.HasPrefix(m.View(), "symlinks\n  [x] Claude .claude/skills\n> [x] Codex / Goose .agents/skills\n") {
		t.Fatal(m.View())
	}
	bare := tui.New(lib, nil, nil)
	bare.Handle(tui.KeyTab)
	bare.Handle(tui.KeySpace)
	bare.Handle(tui.KeyUp)
	bare.Handle(tui.KeyDown)
	bare.Handle(tui.KeyDown)
	bare.Handle(tui.KeyDown)
	bare.Handle(tui.KeySpace)
	if !bare.AgentWant()["Claude"] || !bare.AgentWant()["Cursor"] || bare.AgentWant()["Codex / Goose"] {
		t.Fatal(bare.AgentWant())
	}
}

func TestShortWindowKeepsHeaderAndKeys(t *testing.T) {
	m := tallSkills()
	m.SetHeight(5)
	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) > 5 {
		t.Fatalf("view is %d lines; a full window drops the header", len(lines))
	}
	if len(lines) != 5 || !strings.HasPrefix(lines[0], "available") || !strings.Contains(lines[0], "current") {
		t.Fatal(view)
	}
	if lines[4] != "↑↓ move  ←→ panel  space select  g group  a add and remove  tab agents  q quit" {
		t.Fatal(view)
	}
	if !strings.Contains(view, "alpha") || strings.Contains(view, "epsilon") {
		t.Fatal(view)
	}
	for i := 0; i < 8; i++ {
		m.Handle(tui.KeyDown)
	}
	view = m.View()
	if !strings.HasPrefix(view, "available") || !strings.HasSuffix(view, "↑↓ move  ←→ panel  space select  g group  a add and remove  tab agents  q quit") {
		t.Fatal(view)
	}
	if strings.Contains(view, "alpha") || !strings.Contains(view, "> epsilon") {
		t.Fatal(view)
	}
}

func TestShortWindowKeepsDownloadCount(t *testing.T) {
	lib := library.Library{Groups: []library.Group{{Name: "review", Skills: []library.Skill{
		{Name: "brainstorming", URL: "https://github.com/other/skills/tree/main/review/brainstorming"},
	}}}}
	m := tui.New(lib, nil, nil)
	m.Handle(tui.KeyG)
	m.Handle(tui.KeyA)
	m.SetHeight(4)
	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) > 4 {
		t.Fatalf("view is %d lines; a full window drops the header", len(lines))
	}
	if len(lines) != 4 || !strings.HasPrefix(lines[0], "available") || !strings.Contains(lines[0], "current") {
		t.Fatal(view)
	}
	if !strings.Contains(view, "0 / 1 downloaded\n") || !strings.HasSuffix(view, "↑↓ move  ←→ panel  space select  g group  a add and remove  tab agents  q quit") {
		t.Fatal(view)
	}
}

func TestShortAgentsWindowKeepsTitle(t *testing.T) {
	m := tui.New(library.Library{}, nil, nil)
	m.Handle(tui.KeyTab)
	m.SetHeight(4)
	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) > 4 {
		t.Fatalf("view is %d lines; a full window drops the header", len(lines))
	}
	if len(lines) != 4 || lines[0] != "symlinks" {
		t.Fatal(view)
	}
	if !strings.Contains(view, "Claude") || strings.Contains(view, "Pi") {
		t.Fatal(view)
	}
	if lines[3] != "↑↓ move  space select  a link  tab skills  q quit" {
		t.Fatal(view)
	}
	for i := 0; i < 6; i++ {
		m.Handle(tui.KeyDown)
	}
	view = m.View()
	if !strings.HasPrefix(view, "symlinks\n") || strings.Contains(view, "Claude") || !strings.Contains(view, "> [ ] Pi .pi/skills") {
		t.Fatal(view)
	}
}

func tallSkills() tui.Model {
	names := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
	groups := make([]library.Group, len(names))
	for i, name := range names {
		groups[i] = library.Group{Name: name, Skills: []library.Skill{{
			Name: name, URL: "https://github.com/example/skills/tree/main/" + name,
		}}}
	}
	return tui.New(library.Library{Groups: groups}, nil, nil)
}

func TestApplyStaysResponsive(t *testing.T) {
	lib := library.Library{Groups: []library.Group{{Name: "review", Skills: []library.Skill{
		{Name: "brainstorming", URL: "https://github.com/other/skills/tree/main/review/brainstorming"},
	}}}}
	m := tui.New(lib, []string{"brainstorming"}, nil)
	m.Handle(tui.KeyG)
	m.Handle(tui.KeyA)
	if !m.Busy() || len(m.Selection().Add) != 1 {
		t.Fatal(m.Busy(), m.Selection())
	}
	m.Handle(tui.KeyTab)
	if m.Screen() != "agents" {
		t.Fatal("tab blocked while busy")
	}
	if !strings.Contains(m.View(), "0 / 1 downloaded\n") {
		t.Fatal(m.View())
	}
	if !strings.HasSuffix(m.View(), "↑↓ move  space select  a link  tab skills  q quit") {
		t.Fatal(m.View())
	}
}

func TestDownloadedSkillAppearsOnTheRight(t *testing.T) {
	lib := library.Library{Groups: []library.Group{{Name: "superpowers", Skills: []library.Skill{
		{Name: "brainstorming", URL: "https://github.com/obra/superpowers/tree/main/skills/brainstorming"},
		{Name: "writing-plans", URL: "https://github.com/obra/superpowers/tree/main/skills/writing-plans"},
	}}}}
	m := tui.New(lib, nil, nil)
	m.Handle(tui.KeyG)
	m.Handle(tui.KeyA)
	m.NoteDownloaded("brainstorming")
	view := m.View()
	if !strings.Contains(view, "1 / 2 downloaded\n") || !strings.Contains(view, "[ ] brainstorming") {
		t.Fatal(view)
	}
	if strings.Count(view, "superpowers") != 2 {
		t.Fatal(view)
	}
	if !strings.Contains(view, "[\x1b[32m✓\x1b[0m] brainstorming") {
		t.Fatal(view)
	}
	m.NoteDownloaded("writing-plans")
	view = m.View()
	if !strings.Contains(view, "2 / 2 downloaded\n") || !strings.Contains(view, "[ ] writing-plans") {
		t.Fatal(view)
	}
}
