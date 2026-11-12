// Package agents symlinks project skills into agent directories.
package agents

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
)

const linkTarget = "../.tewyn/skills"

// Link is an agent name and its skill path relative to the project.
type Link struct {
	Name string
	Rel  string
}

// Links returns agent skill paths sorted by name.
func Links() []Link {
	links := []Link{
		{Name: "Claude", Rel: ".claude/skills"},
		{Name: "Codex / Goose", Rel: ".agents/skills"},
		{Name: "Crush", Rel: ".crush/skills"},
		{Name: "Cursor", Rel: ".cursor/skills"},
		{Name: "Grok Build", Rel: ".grok/skills"},
		{Name: "OpenCode", Rel: ".opencode/skills"},
		{Name: "Pi", Rel: ".pi/skills"},
	}
	sort.Slice(links, func(i, j int) bool { return links[i].Name < links[j].Name })
	return links
}

// LinkError means Rel is not a symlink to .tewyn/skills.
type LinkError struct {
	Agent string
	Rel   string
}

// Error returns "<Agent>: <Rel> is not a symlink to .tewyn/skills".
func (e LinkError) Error() string {
	return e.Agent + ": " + e.Rel + " is not a symlink to .tewyn/skills"
}

// State reports true only for a symlink whose target is ../.tewyn/skills.
func State(projectDir string) (map[string]bool, error) {
	out := make(map[string]bool, len(Links()))
	for _, l := range Links() {
		path := filepath.Join(projectDir, l.Rel)
		got, err := os.Readlink(path)
		if err != nil && !os.IsNotExist(err) {
			if _, statErr := os.Lstat(path); statErr != nil {
				return nil, err
			}
		}
		out[l.Name] = got == linkTarget
	}
	return out, nil
}

// Apply creates or removes agent skill symlinks. A missing want key is unchecked.
func Apply(projectDir string, want map[string]bool) error {
	var err error
	for _, l := range Links() {
		err = errors.Join(err, applyLink(projectDir, l, want[l.Name]))
	}
	return err
}

func applyLink(projectDir string, l Link, want bool) error {
	path := filepath.Join(projectDir, l.Rel)
	got, err := os.Readlink(path)
	if err == nil && got == linkTarget {
		if want {
			return nil
		}
		return os.Remove(path)
	}
	if os.IsNotExist(err) {
		if !want {
			return nil
		}
		if err = os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			return err
		}
		return os.Symlink(linkTarget, path)
	}
	if _, statErr := os.Lstat(path); err != nil && statErr != nil {
		return err
	}
	return LinkError{Agent: l.Name, Rel: l.Rel}
}
