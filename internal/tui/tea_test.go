package tui

import (
	"context"
	"strings"
	"testing"
	"tewyn/internal/library"
	"tewyn/internal/project"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestQuitWaitsForSkills(t *testing.T) {
	release := make(chan struct{})
	u := &ui{onSkills: func(ctx context.Context, _ project.Selection, _ func(string)) error {
		<-release
		return ctx.Err()
	}}
	key := func(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }
	_, start := u.Update(key('a'))
	got := make(chan tea.Msg, 1)
	go func() { got <- start() }()
	_, cmd := u.Update(key('q'))
	if _, ok := msgOf(cmd).(tea.QuitMsg); ok {
		t.Fatal("quit while in flight")
	}
	close(release)
	var msg tea.Msg
	select {
	case msg = <-got:
	case <-time.After(time.Second):
		t.Fatal("callback did not return")
	}
	if am, ok := msg.(appliedMsg); !ok || am.err != context.Canceled {
		t.Fatal(msg)
	}
	_, cmd = u.Update(msg)
	if _, ok := msgOf(cmd).(tea.QuitMsg); !ok {
		t.Fatal("expected quit")
	}
}

func TestWindowSizeClipsTheList(t *testing.T) {
	names := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
	groups := make([]library.Group, len(names))
	for i, name := range names {
		groups[i] = library.Group{Name: name, Skills: []library.Skill{{
			Name: name, URL: "https://github.com/example/skills/tree/main/" + name,
		}}}
	}
	u := &ui{m: New(library.Library{Groups: groups}, nil, nil)}
	u.Update(tea.WindowSizeMsg{Width: 80, Height: 5})
	view := u.View()
	if n := len(strings.Split(view, "\n")); n > 5 {
		t.Fatalf("view is %d lines; a full window drops the header", n)
	}
	if strings.Contains(view, "epsilon") || !strings.HasPrefix(view, "available") {
		t.Fatal(view)
	}
}

func msgOf(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}
