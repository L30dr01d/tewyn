// Package library stores the skill library on disk.
package library

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Skill is one skill in the library.
type Skill struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Commit string `json:"commit"`
}

// Group is a named set of skills.
type Group struct {
	Name   string  `json:"name"`
	Skills []Skill `json:"skills"`
}

// Library is the set of skill groups.
type Library struct {
	Groups []Group `json:"groups"`
}

// Path returns the library.json path from XDG_CONFIG_HOME, then HOME.
func Path(getenv func(string) string) (string, error) {
	if d := getenv("XDG_CONFIG_HOME"); d != "" {
		return d + "/tewyn/library.json", nil
	}
	if d := getenv("HOME"); d != "" {
		return d + "/.config/tewyn/library.json", nil
	}
	return "", fmt.Errorf("no config home")
}

// Load reads a library. A missing file is an empty library.
func Load(path string) (Library, error) {
	b, err := fs.ReadFile(os.DirFS(filepath.Dir(path)), filepath.Base(path))
	if os.IsNotExist(err) {
		return Library{}, nil
	}
	if err != nil {
		return Library{}, err
	}
	var lib Library
	err = json.Unmarshal(b, &lib)
	return lib, err
}

// Save writes the library as two-space JSON and creates parent directories.
func Save(path string, lib Library) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	b, err := json.MarshalIndent(lib, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o600)
}

// Visible drops groups that contain no skills.
func (l Library) Visible() Library {
	var groups []Group
	for _, g := range l.Groups {
		if len(g.Skills) > 0 {
			groups = append(groups, g)
		}
	}
	return Library{Groups: groups}
}

// SkillsDir is the cache directory beside library.json.
func SkillsDir(libraryPath string) string {
	return filepath.Join(filepath.Dir(libraryPath), "skills")
}

// Find returns the first skill with that name.
func (l Library) Find(name string) (Skill, bool) {
	for _, g := range l.Groups {
		for _, s := range g.Skills {
			if s.Name == name {
				return s, true
			}
		}
	}
	return Skill{}, false
}

// Check reports a repeated group name, a skill URL shared by groups, or a repeated skill name.
func (l Library) Check() error {
	names := map[string]bool{}
	urls := map[string]bool{}
	skillNames := map[string]bool{}
	for _, g := range l.Groups {
		if names[g.Name] {
			return fmt.Errorf("group %s appears twice", g.Name)
		}
		names[g.Name] = true
		for _, s := range g.Skills {
			if urls[s.URL] {
				return fmt.Errorf("skill url %s is in more than one group", s.URL)
			}
			urls[s.URL] = true
			if skillNames[s.Name] {
				return fmt.Errorf("skill %s appears twice", s.Name)
			}
			skillNames[s.Name] = true
		}
	}
	return nil
}

// Add appends a group when the library would still pass Check.
func (l *Library) Add(g Group) error {
	for _, e := range l.Groups {
		if e.Name == g.Name {
			return fmt.Errorf("group %s already exists", g.Name)
		}
	}
	if len(g.Skills) == 0 {
		return fmt.Errorf("group %s is empty", g.Name)
	}
	next := Library{Groups: append(append([]Group{}, l.Groups...), g)}
	if err := next.Check(); err != nil {
		return err
	}
	*l = next
	return nil
}
