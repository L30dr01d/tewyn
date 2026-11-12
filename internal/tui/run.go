package tui

import (
	"context"
	"strings"
	"tewyn/internal/agents"
	"tewyn/internal/library"
	"tewyn/internal/project"
)

// LoadModel reads the library and the skill directories in the project.
func LoadModel(projectDir string, getenv func(string) string) (Model, error) {
	path, err := library.Path(getenv)
	if err != nil {
		return Model{}, err
	}
	lib, err := library.Load(path)
	if err != nil {
		return Model{}, err
	}
	if err = lib.Check(); err != nil {
		return Model{}, err
	}
	names, err := project.SkillNames(projectDir)
	if err != nil {
		return Model{}, err
	}
	linked, err := agents.State(projectDir)
	if err != nil {
		return Model{}, err
	}
	m := New(lib, names, linked)
	m.projectDir, m.getenv = projectDir, getenv
	return m, nil
}

// Run handles keys until Quit. A skill download and an agent link can run at the same time.
func Run(ctx context.Context, m Model, keys <-chan Key, onSkills func(context.Context, project.Selection, func(string)) error, onAgents func(map[string]bool) error) Model {
	for k := range keys {
		m.Handle(k)
		if m.Quit() {
			return m
		}
		if k != KeyA {
			continue
		}
		if m.Screen() == "agents" {
			if onAgents != nil {
				m = m.appliedAgents(onAgents(m.AgentWant()))
			}
			continue
		}
		if onSkills == nil {
			continue
		}
		var stop bool
		m, stop = m.skills(ctx, keys, onSkills, onAgents)
		if stop {
			return m
		}
	}
	return m
}

func (m Model) skills(ctx context.Context, keys <-chan Key, onSkills func(context.Context, project.Selection, func(string)) error, onAgents func(map[string]bool) error) (Model, bool) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	sel := m.Selection()
	errc := make(chan error, 1)
	prog := make(chan string, 256)
	go func() {
		errc <- onSkills(ctx, sel, func(name string) {
			select {
			case prog <- name:
			case <-ctx.Done():
			}
		})
	}()
	for {
		select {
		case name := <-prog:
			m.NoteDownloaded(name)
		case k, ok := <-keys:
			if !ok {
				cancel()
				m = m.applied(<-errc)
				m = drainProgress(m, prog)
				return m, true
			}
			m.Handle(k)
			if m.Quit() {
				cancel()
				m = m.applied(<-errc)
				m = drainProgress(m, prog)
				return m, true
			}
			if k == KeyA && m.Screen() == "agents" && onAgents != nil {
				m = m.appliedAgents(onAgents(m.AgentWant()))
			}
		case err := <-errc:
			m = m.applied(err)
			return drainProgress(m, prog), false
		}
	}
}

func drainProgress(m Model, prog <-chan string) Model {
	for {
		select {
		case name := <-prog:
			m.NoteDownloaded(name)
		default:
			return m
		}
	}
}

func (m Model) applied(err error) Model {
	m.busy = false
	m.notice = ""
	if err != nil {
		m.notice = err.Error()
	}
	if m.projectDir == "" {
		return m
	}
	n, nerr := LoadModel(m.projectDir, m.getenv)
	if nerr != nil {
		return m
	}
	n.screen, n.notice, n.agentNote, n.quit = m.screen, m.notice, m.agentNote, m.quit
	n.done, n.total = m.done, m.total
	n.onRight, n.cur, n.acur = m.onRight, m.cur, m.acur
	return n
}

func (m Model) appliedAgents(err error) Model {
	if err != nil {
		m.agentNote = err.Error()
		return m
	}
	var names []string
	for _, l := range agents.Links() {
		if m.want[l.Name] {
			names = append(names, l.Name)
		}
	}
	if len(names) == 0 {
		m.agentNote = "unlinked"
		return m
	}
	m.agentNote = "linked " + strings.Join(names, ", ")
	return m
}
