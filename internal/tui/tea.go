package tui

import (
	"context"
	"tewyn/internal/project"

	tea "github.com/charmbracelet/bubbletea"
)

// Start runs the Bubble Tea loop, mapping keys onto Key and calling Model.Handle.
func Start(m Model, onSkills func(context.Context, project.Selection, func(string)) error, onAgents func(map[string]bool) error) error {
	u := &ui{m: m, onSkills: onSkills, onAgents: onAgents}
	p := tea.NewProgram(u, tea.WithAltScreen())
	u.send = p.Send
	_, err := p.Run()
	return err
}

type ui struct {
	m          Model
	onSkills   func(context.Context, project.Selection, func(string)) error
	onAgents   func(map[string]bool) error
	cancel     context.CancelFunc
	skillsBusy bool
	agentsBusy bool
	send       func(tea.Msg)
}

func (u *ui) Init() tea.Cmd { return nil }

func (u *ui) View() string { return u.m.View() }

func (u *ui) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		u.m.SetHeight(msg.Height)
		return u, nil
	case tea.KeyMsg:
		k, ok := teaKey(msg)
		if !ok {
			return u, nil
		}
		u.m.Handle(k)
		if u.m.Quit() {
			if u.skillsBusy {
				u.stop()
			}
			if u.skillsBusy || u.agentsBusy {
				return u, nil
			}
			return u, tea.Quit
		}
		if k != KeyA {
			return u, nil
		}
		if u.m.Screen() == "agents" {
			if u.agentsBusy {
				return u, nil
			}
			u.agentsBusy = true
			want := u.m.AgentWant()
			return u, func() tea.Msg {
				var err error
				if u.onAgents != nil {
					err = u.onAgents(want)
				}
				return appliedMsg{kind: "agents", err: err}
			}
		}
		if u.skillsBusy {
			return u, nil
		}
		u.skillsBusy = true
		sel := u.m.Selection()
		ctx, cancel := context.WithCancel(context.Background())
		u.cancel = cancel
		return u, func() tea.Msg {
			var err error
			if u.onSkills != nil {
				err = u.onSkills(ctx, sel, func(name string) {
					if u.send != nil {
						u.send(progressMsg{name})
					}
				})
			}
			return appliedMsg{kind: "skills", err: err}
		}
	case progressMsg:
		u.m.NoteDownloaded(msg.name)
		return u, nil
	case appliedMsg:
		if msg.kind == "agents" {
			u.agentsBusy = false
			u.m = u.m.appliedAgents(msg.err)
		} else {
			u.skillsBusy = false
			u.stop()
			u.m = u.m.applied(msg.err)
		}
		if u.m.Quit() && !u.skillsBusy && !u.agentsBusy {
			return u, tea.Quit
		}
		return u, nil
	}
	return u, nil
}

func (u *ui) stop() {
	if u.cancel != nil {
		u.cancel()
		u.cancel = nil
	}
}

type progressMsg struct{ name string }

type appliedMsg struct {
	kind string
	err  error
}

func teaKey(msg tea.KeyMsg) (Key, bool) {
	switch msg.String() {
	case "up":
		return KeyUp, true
	case "down":
		return KeyDown, true
	case "left":
		return KeyLeft, true
	case "right":
		return KeyRight, true
	case " ":
		return KeySpace, true
	case "g":
		return KeyG, true
	case "a":
		return KeyA, true
	case "tab":
		return KeyTab, true
	case "q", "ctrl+c":
		return KeyQ, true
	default:
		return 0, false
	}
}
