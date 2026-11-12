// Package tui is the key model for the skill and agent screens.
package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"tewyn/internal/agents"
	"tewyn/internal/library"
	"tewyn/internal/project"
)

// Key is one keyboard command.
type Key int

const (
	KeyUp Key = iota
	KeyDown
	KeyLeft
	KeyRight
	KeySpace
	KeyG
	KeyA
	KeyTab
	KeyQ
)

// Model is the skill panels and the agent screen.
type Model struct {
	left, right []row
	cur, acur   int
	onRight     bool
	screen      string
	want        map[string]bool
	busy, quit  bool
	done, total int
	height      int
	notice      string
	agentNote   string
	projectDir  string
	getenv      func(string) string
}

type row struct {
	head bool
	grp  int
	name string
	add  library.Skill
	del  string
	on   bool
}

// New builds a model. Left rows are each visible group's heading and skills.
// Right rows are project skill names under the library group with that name, then unmatched names.
func New(lib library.Library, names []string, linked map[string]bool) Model {
	vis := lib.Visible()
	want := map[string]bool{}
	for name, on := range linked {
		want[name] = on
	}
	m := Model{screen: "skills", want: want}
	loc := map[string]int{}
	for gi, g := range vis.Groups {
		m.left = append(m.left, row{head: true, grp: gi, name: g.Name})
		for _, s := range g.Skills {
			m.left = append(m.left, row{grp: gi, name: s.Name, add: s})
			if _, ok := loc[s.Name]; !ok {
				loc[s.Name] = gi
			}
		}
	}
	buckets := make([][]string, len(vis.Groups))
	var rest []string
	for _, name := range names {
		if gi, ok := loc[name]; ok {
			buckets[gi] = append(buckets[gi], name)
			continue
		}
		rest = append(rest, name)
	}
	for gi, skills := range buckets {
		if len(skills) == 0 {
			continue
		}
		m.right = append(m.right, row{head: true, grp: gi, name: vis.Groups[gi].Name})
		for _, name := range skills {
			m.right = append(m.right, row{grp: gi, name: name, del: name})
		}
	}
	for i, name := range rest {
		m.right = append(m.right, row{grp: -1 - i, name: name, del: name})
	}
	return m
}

// Handle applies one key. KeyA sets Busy and does no IO.
func (m *Model) Handle(k Key) {
	switch k {
	case KeyQ:
		m.quit = true
		return
	case KeyA:
		if m.screen == "agents" || m.busy {
			return
		}
		m.busy = true
		m.done = 0
		m.total = 0
		for _, r := range m.left {
			if !r.head && r.on {
				m.total++
			}
		}
		return
	case KeyTab:
		if m.screen == "skills" {
			m.screen = "agents"
			return
		}
		m.screen = "skills"
		return
	}
	if m.screen != "skills" {
		links := agents.Links()
		switch k {
		case KeyUp:
			if m.acur > 0 {
				m.acur--
			}
		case KeyDown:
			if m.acur+1 < len(links) {
				m.acur++
			}
		case KeySpace:
			if m.acur < len(links) {
				name := links[m.acur].Name
				m.want[name] = !m.want[name]
			}
		}
		return
	}
	rows := m.left
	if m.onRight {
		rows = m.right
	}
	switch k {
	case KeyLeft:
		m.onRight, m.cur = false, 0
	case KeyRight:
		m.onRight, m.cur = true, 0
	case KeyUp:
		if m.cur > 0 {
			m.cur--
		}
	case KeyDown:
		if m.cur+1 < len(rows) {
			m.cur++
		}
	case KeySpace:
		if m.cur < len(rows) && !rows[m.cur].head {
			rows[m.cur].on = !rows[m.cur].on
		}
	case KeyG:
		if m.cur < len(rows) {
			flip(rows, rows[m.cur].grp)
		}
	}
}

func flip(rows []row, grp int) {
	on := false
	for i := range rows {
		if !rows[i].head && rows[i].grp == grp && !rows[i].on {
			on = true
			break
		}
	}
	for i := range rows {
		if !rows[i].head && rows[i].grp == grp {
			rows[i].on = on
		}
	}
}

// Quit reports whether KeyQ was pressed.
func (m Model) Quit() bool { return m.quit }

// Busy reports whether KeyA was pressed.
func (m Model) Busy() bool { return m.busy }

// Screen returns "skills" or "agents".
func (m Model) Screen() string { return m.screen }

// Selection returns checked library skills to add and checked project skills to delete.
func (m Model) Selection() project.Selection {
	var sel project.Selection
	for _, r := range m.left {
		if !r.head && r.on {
			sel.Add = append(sel.Add, r.add)
		}
	}
	for _, r := range m.right {
		if !r.head && r.on {
			sel.Delete = append(sel.Delete, r.del)
		}
	}
	return sel
}

// NoteDownloaded records one finished skill and shows it under current.
func (m *Model) NoteDownloaded(name string) {
	m.done++
	for _, r := range m.right {
		if !r.head && r.name == name {
			return
		}
	}
	grp := -1
	gname := ""
	for _, r := range m.left {
		if r.head || r.name != name {
			continue
		}
		grp = r.grp
		break
	}
	for _, r := range m.left {
		if r.head && r.grp == grp {
			gname = r.name
			break
		}
	}
	skill := row{grp: grp, name: name, del: name}
	if grp < 0 || gname == "" {
		m.right = append(m.right, skill)
		return
	}
	at := -1
	for i, r := range m.right {
		if r.head && r.grp == grp {
			at = i
			break
		}
	}
	if at < 0 {
		m.right = append(m.right, row{head: true, grp: grp, name: gname}, skill)
		return
	}
	insert := at + 1
	for insert < len(m.right) && !m.right[insert].head && m.right[insert].grp == grp {
		insert++
	}
	m.right = append(m.right, row{})
	copy(m.right[insert+1:], m.right[insert:])
	m.right[insert] = skill
}

// SetHeight is the terminal height in rows. Zero shows the whole list.
func (m *Model) SetHeight(n int) { m.height = n }

// AgentWant returns the agent checks passed to New.
func (m Model) AgentWant() map[string]bool { return m.want }

// Notice returns the last error from applying skills or agents.
func (m Model) Notice() string { return m.notice }

const (
	skillsKeys = "↑↓ move  ←→ panel  space select  g group  a add and remove  tab agents  q quit"
	agentKeys  = "↑↓ move  space select  a link  tab skills  q quit"
	greenCheck = "[\x1b[32m✓\x1b[0m]"
	redCross   = "[\x1b[31m✕\x1b[0m]"
)

// View shows the skills panels or one checkbox row per agent.
// The title line and the key line stay on screen; the rows between them scroll.
func (m Model) View() string {
	if m.screen == "agents" {
		return m.viewAgents()
	}
	return m.viewSkills()
}

func (m Model) viewSkills() string {
	left := rowLines(m.left, !m.onRight, m.cur, true)
	right := rowLines(m.right, m.onRight, m.cur, false)
	w := visLen("available")
	for _, line := range left {
		if v := visLen(line); v > w {
			w = v
		}
	}
	var footer []string
	if p := strings.TrimSuffix(m.progress(), "\n"); p != "" {
		footer = append(footer, p)
	}
	if m.notice != "" {
		footer = append(footer, m.notice)
	}
	footer = append(footer, skillsKeys)
	return fit(m.height, zipLine("available", "current", w), zipBody(left, right, w), footer, m.cur)
}

func (m Model) viewAgents() string {
	links := agents.Links()
	body := make([]string, len(links))
	for i, l := range links {
		gutter := "  "
		if i == m.acur {
			gutter = "> "
		}
		mark := "[ ] "
		if m.want[l.Name] {
			mark = "[x] "
		}
		body[i] = gutter + mark + l.Name + " " + l.Rel
	}
	var footer []string
	if p := strings.TrimSuffix(m.progress(), "\n"); p != "" {
		footer = append(footer, p)
	}
	if m.agentNote != "" {
		footer = append(footer, m.agentNote)
	}
	footer = append(footer, agentKeys)
	return fit(m.height, "symlinks", body, footer, m.acur)
}

// fit keeps header and footer on screen and shows the body rows around cur.
func fit(height int, header string, body, footer []string, cur int) string {
	if height > 0 && height < 1+len(footer) {
		keep := height - 1
		if keep < 0 {
			keep = 0
		}
		footer = footer[len(footer)-keep:]
	}
	room := len(body)
	if height > 0 {
		room = height - 1 - len(footer)
		if room < 0 {
			room = 0
		}
		if room > len(body) {
			room = len(body)
		}
	}
	if room < len(body) {
		off := cur - room + 1
		if off < 0 {
			off = 0
		}
		if max := len(body) - room; off > max {
			off = max
		}
		body = body[off : off+room]
	}
	lines := make([]string, 0, 1+len(body)+len(footer))
	lines = append(lines, header)
	lines = append(lines, body...)
	lines = append(lines, footer...)
	// No trailing newline. A full window plus a blank line is one row too
	// tall, and the renderer drops the top line, which is the header.
	return strings.Join(lines, "\n")
}

func (m Model) progress() string {
	if m.total == 0 {
		return ""
	}
	return fmt.Sprintf("%d / %d downloaded\n", m.done, m.total)
}

func rowLines(rows []row, focus bool, cur int, add bool) []string {
	lines := make([]string, 0, len(rows))
	for i, r := range rows {
		gutter := "  "
		if focus && i == cur {
			gutter = "> "
		}
		if r.head {
			lines = append(lines, gutter+r.name)
			continue
		}
		mark := "[ ]"
		if r.on {
			if add {
				mark = greenCheck
			} else {
				mark = redCross
			}
		}
		lines = append(lines, gutter+mark+" "+r.name)
	}
	return lines
}

func zipLine(left, right string, w int) string {
	if right == "" {
		return left
	}
	pad := w - visLen(left)
	if pad < 0 {
		pad = 0
	}
	return left + strings.Repeat(" ", pad+2) + right
}

func zipBody(left, right []string, w int) []string {
	n := len(left)
	if len(right) > n {
		n = len(right)
	}
	out := make([]string, n)
	for i := 0; i < n; i++ {
		l, r := "", ""
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		out[i] = zipLine(l, r, w)
	}
	return out
}

func visLen(s string) int {
	n := 0
	for i := 0; i < len(s); {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				i = j + 1
				continue
			}
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		i += size
		n++
	}
	return n
}
