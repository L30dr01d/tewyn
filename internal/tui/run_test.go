package tui_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"tewyn/internal/library"
	"tewyn/internal/project"
	"tewyn/internal/tui"
	"time"
)

func TestQuitCancelsDownload(t *testing.T) {
	lib := library.Library{Groups: []library.Group{{Name: "ponytail", Skills: []library.Skill{{
		Name: "ponytail", URL: "https://github.com/DietrichGebert/ponytail/tree/main/skills/ponytail",
	}}}}}
	m := tui.New(lib, nil, nil)
	m.Handle(tui.KeyDown)
	m.Handle(tui.KeySpace)
	keys := make(chan tui.Key, 2)
	keys <- tui.KeyA
	keys <- tui.KeyQ
	close(keys)
	canceled := make(chan struct{})
	done := make(chan struct{})
	go func() {
		tui.Run(context.Background(), m, keys, func(ctx context.Context, _ project.Selection, _ func(string)) error {
			<-ctx.Done()
			close(canceled)
			return ctx.Err()
		}, func(map[string]bool) error { return nil })
		close(done)
	}()
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("download context was not canceled")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return")
	}
}

func TestOpenDoesNotCreateProjectDir(t *testing.T) {
	proj := t.TempDir()
	home := t.TempDir()
	m, err := tui.LoadModel(proj, func(k string) string {
		if k == "HOME" {
			return home
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	keys := make(chan tui.Key, 1)
	keys <- tui.KeyQ
	close(keys)
	tui.Run(context.Background(), m, keys, nil, nil)
	if _, err := os.Stat(filepath.Join(proj, ".tewyn")); !os.IsNotExist(err) {
		t.Fatal(err)
	}
}

func TestRunShowsApplyError(t *testing.T) {
	lib := library.Library{Groups: []library.Group{{Name: "review", Skills: []library.Skill{{
		Name: "brainstorming", URL: "https://github.com/other/skills/tree/main/review/brainstorming",
	}}}}}
	m := tui.New(lib, []string{"brainstorming"}, nil)
	m.Handle(tui.KeyG)
	keys := make(chan tui.Key, 2)
	keys <- tui.KeyA
	keys <- tui.KeyQ
	close(keys)
	got := tui.Run(context.Background(), m, keys, func(context.Context, project.Selection, func(string)) error {
		return project.NameClashError{Name: "brainstorming"}
	}, nil)
	if got.Notice() != "two skills named \"brainstorming\" cannot both exist" {
		t.Fatal(got.Notice())
	}
}

func TestSuccessfulApplyClearsNotice(t *testing.T) {
	proj := t.TempDir()
	home := t.TempDir()
	m, err := tui.LoadModel(proj, func(k string) string {
		if k == "HOME" {
			return home
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	run := func(fn func(context.Context, project.Selection, func(string)) error) tui.Model {
		keys := make(chan tui.Key, 1)
		keys <- tui.KeyA
		close(keys)
		return tui.Run(context.Background(), m, keys, fn, nil)
	}
	m = run(func(context.Context, project.Selection, func(string)) error {
		return project.NameClashError{Name: "brainstorming"}
	})
	if m.Notice() != "two skills named \"brainstorming\" cannot both exist" {
		t.Fatal(m.Notice())
	}
	m = run(func(context.Context, project.Selection, func(string)) error { return nil })
	if m.Notice() != "" {
		t.Fatal(m.Notice())
	}
}

func TestViewDownloadingBeforeNotice(t *testing.T) {
	m := tui.New(library.Library{}, nil, nil)
	keys := make(chan tui.Key, 1)
	keys <- tui.KeyA
	close(keys)
	got := tui.Run(context.Background(), m, keys, func(context.Context, project.Selection, func(string)) error {
		return errors.New("boom")
	}, nil)
	view := got.View()
	if strings.Contains(view, "downloading") || !strings.Contains(view, "boom\n") {
		t.Fatal(view)
	}
	if !strings.HasSuffix(view, "↑↓ move  ←→ panel  space select  g group  a add and remove  tab agents  q quit") {
		t.Fatal(view)
	}
	if strings.LastIndex(view, "boom\n") > strings.LastIndex(view, "↑↓ move") {
		t.Fatal(view)
	}
}

func TestRunShowsEachFinishedSkill(t *testing.T) {
	lib := library.Library{Groups: []library.Group{{Name: "superpowers", Skills: []library.Skill{
		{Name: "brainstorming", URL: "https://github.com/obra/superpowers/tree/main/skills/brainstorming"},
		{Name: "writing-plans", URL: "https://github.com/obra/superpowers/tree/main/skills/writing-plans"},
	}}}}
	m := tui.New(lib, nil, nil)
	m.Handle(tui.KeyG)
	keys := make(chan tui.Key)
	finished := make(chan struct{})
	done := make(chan tui.Model, 1)
	go func() {
		done <- tui.Run(context.Background(), m, keys, func(_ context.Context, _ project.Selection, progress func(string)) error {
			progress("brainstorming")
			progress("writing-plans")
			close(finished)
			return nil
		}, nil)
	}()
	keys <- tui.KeyA
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("download did not report skills")
	}
	keys <- tui.KeyQ
	close(keys)
	var got tui.Model
	select {
	case got = <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return")
	}
	view := got.View()
	if !strings.Contains(view, "2 / 2 downloaded\n") || !strings.Contains(view, "[ ] brainstorming") || !strings.Contains(view, "[ ] writing-plans") {
		t.Fatal(view)
	}
}

func TestAgentLinkDuringDownload(t *testing.T) {
	lib := library.Library{Groups: []library.Group{{Name: "superpowers", Skills: []library.Skill{
		{Name: "brainstorming", URL: "https://github.com/obra/superpowers/tree/main/skills/brainstorming"},
	}}}}
	m := tui.New(lib, nil, nil)
	m.Handle(tui.KeyDown)
	m.Handle(tui.KeySpace)
	linked := make(chan struct{})
	keys := make(chan tui.Key)
	done := make(chan tui.Model, 1)
	go func() {
		done <- tui.Run(context.Background(), m, keys, func(ctx context.Context, _ project.Selection, _ func(string)) error {
			<-ctx.Done()
			return ctx.Err()
		}, func(want map[string]bool) error {
			if !want["Claude"] {
				t.Errorf("Claude unchecked")
			}
			close(linked)
			return nil
		})
	}()
	keys <- tui.KeyA
	keys <- tui.KeyTab
	keys <- tui.KeySpace
	keys <- tui.KeyA
	select {
	case <-linked:
	case <-time.After(time.Second):
		t.Fatal("agent link waited for the download")
	}
	keys <- tui.KeyQ
	close(keys)
	var got tui.Model
	select {
	case got = <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return")
	}
	if !strings.Contains(got.View(), "linked Claude\n") {
		t.Fatal(got.View())
	}
}
