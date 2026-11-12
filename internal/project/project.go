// Package project pins and removes skills in a project directory.
package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"tewyn/internal/library"
)

// SkillNames lists the skill directories under .tewyn/skills.
// A missing .tewyn or skills directory is an empty list and creates nothing.
func SkillNames(projectDir string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(projectDir, ".tewyn", "skills"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	slices.Sort(names)
	return names, nil
}

// Selection is the skills to remove and pin.
type Selection struct {
	Delete []string
	Add    []library.Skill
}

// Result is what Apply removed, pinned, and skipped.
type Result struct {
	Deleted, Added, Skipped []string
}

// NameClashError means two skills would share one name.
type NameClashError struct{ Name string }

// Error returns two skills named "<Name>" cannot both exist.
func (e NameClashError) Error() string {
	return fmt.Sprintf("two skills named %q cannot both exist", e.Name)
}

func badName(name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("bad skill name %q", name)
	}
	return nil
}

// Apply deletes every name in Delete, then copies unique adds from the cache.
func Apply(ctx context.Context, projectDir, skillsDir string, sel Selection) (Result, error) {
	return ApplyNotify(ctx, projectDir, skillsDir, sel, nil)
}

// ApplyNotify is Apply, and it calls added with each skill name after that skill is copied.
func ApplyNotify(ctx context.Context, projectDir, skillsDir string, sel Selection, added func(string)) (Result, error) {
	for _, name := range sel.Delete {
		if err := badName(name); err != nil {
			return Result{}, err
		}
	}
	for _, s := range sel.Add {
		if err := badName(s.Name); err != nil {
			return Result{}, err
		}
	}
	names, err := SkillNames(projectDir)
	if err != nil {
		return Result{}, err
	}
	present := map[string]bool{}
	for _, name := range names {
		present[name] = true
	}
	var res Result
	for _, name := range sel.Delete {
		path := filepath.Join(projectDir, ".tewyn", "skills", name)
		_, statErr := os.Stat(path)
		if err = os.RemoveAll(path); err != nil {
			return res, err
		}
		res.Deleted = append(res.Deleted, name)
		if statErr == nil {
			delete(present, name)
		}
	}
	n := map[string]int{}
	for name := range present {
		n[name]++
	}
	for _, s := range sel.Add {
		n[s.Name]++
	}
	var clash string
	for _, s := range sel.Add {
		if n[s.Name] > 1 {
			res.Skipped = append(res.Skipped, s.Name)
			if clash == "" {
				clash = s.Name
			}
			continue
		}
		if err = ctx.Err(); err != nil {
			return res, err
		}
		src := filepath.Join(skillsDir, s.Name)
		if _, err = os.Stat(src); err != nil {
			if os.IsNotExist(err) {
				err = fmt.Errorf("skill %s is not in the cache", s.Name)
			}
			return res, err
		}
		dest := filepath.Join(projectDir, ".tewyn", "skills", s.Name)
		if err = os.MkdirAll(filepath.Dir(dest), 0o750); err != nil {
			return res, err
		}
		if err = CopyDir(src, dest); err != nil {
			return res, err
		}
		res.Added = append(res.Added, s.Name)
		if added != nil {
			added(s.Name)
		}
	}
	if clash != "" {
		return res, NameClashError{Name: clash}
	}
	return res, nil
}
