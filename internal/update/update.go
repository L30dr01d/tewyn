// Package update refreshes cached skills and copies them into the project.
package update

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"

	"tewyn/internal/library"
	"tewyn/internal/project"
	"tewyn/internal/source"
)

// Run reads skill directory names under .tewyn/skills, refreshes those names
// in the library cache, and copies the cache folders into the project.
// A missing .tewyn directory returns an error whose text is "no project config".
// One skill's failure is a warning and the loop continues.
func Run(ctx context.Context, projectDir, libraryPath string, tree source.Tree, stdout io.Writer) error {
	tewynDir := filepath.Join(projectDir, ".tewyn")
	if _, err := os.Stat(tewynDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errors.New("no project config")
		}
		return err
	}
	entries, err := os.ReadDir(filepath.Join(tewynDir, "skills"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return writeLine(stdout, "no skills\n")
		}
		return err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	slices.Sort(names)
	if len(names) == 0 {
		return writeLine(stdout, "no skills\n")
	}
	lib, err := library.Load(libraryPath)
	if err != nil {
		return err
	}
	var warnings []struct{ name, reason string }
	for _, name := range names {
		reason, err := refresh(ctx, &lib, libraryPath, projectDir, name, tree, stdout)
		if err != nil {
			return err
		}
		if reason != "" {
			warnings = append(warnings, struct{ name, reason string }{name, reason})
		}
	}
	if len(warnings) == 0 {
		return nil
	}
	if err = writeLine(stdout, "warnings\n"); err != nil {
		return err
	}
	for _, w := range warnings {
		if err = writeLine(stdout, "  "+w.name+"  "+w.reason+"\n"); err != nil {
			return err
		}
	}
	return nil
}

// refresh returns a warning reason. A non-nil error is a failure to write progress.
func refresh(ctx context.Context, lib *library.Library, libraryPath, projectDir, name string, tree source.Tree, stdout io.Writer) (string, error) {
	skill, ok := lib.Find(name)
	if !ok {
		return "not in library", nil
	}
	parsed, err := source.Parse(skill.URL)
	if err != nil {
		return "invalid url", nil
	}
	commit, _, err := tree.List(ctx, parsed.RepoURL, parsed.Ref, parsed.Dir)
	if err != nil {
		return err.Error(), nil
	}
	cache := filepath.Join(library.SkillsDir(libraryPath), name)
	_, statErr := os.Stat(cache)
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return statErr.Error(), nil
	}
	if statErr != nil || commit != skill.Commit {
		if err = writeLine(stdout, "downloading "+name+"\n"); err != nil {
			return "", err
		}
		if err = stageCache(ctx, tree, parsed, cache, commit); err != nil {
			return err.Error(), nil
		}
		setCommit(lib, name, commit)
		if err = library.Save(libraryPath, *lib); err != nil {
			return err.Error(), nil
		}
	}
	proj := filepath.Join(projectDir, ".tewyn", "skills", name)
	if err = project.CopyDir(cache, proj); err != nil {
		return err.Error(), nil
	}
	return "", writeLine(stdout, "copying "+name+"\n")
}

// stageCache downloads into a temp dir beside cache and replaces cache only after Download succeeds.
func stageCache(ctx context.Context, tree source.Tree, parsed source.Parsed, cache, commit string) error {
	if err := os.MkdirAll(filepath.Dir(cache), 0o750); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(filepath.Dir(cache), ".update-*")
	if err != nil {
		return err
	}
	if err = tree.Download(ctx, parsed.RepoURL, commit, parsed.Dir, tmp); err != nil {
		_ = os.RemoveAll(tmp)
		return err
	}
	if err = project.CopyDir(tmp, cache); err != nil {
		_ = os.RemoveAll(tmp)
		return err
	}
	return os.RemoveAll(tmp)
}

func setCommit(lib *library.Library, name, commit string) {
	for i := range lib.Groups {
		for j := range lib.Groups[i].Skills {
			if lib.Groups[i].Skills[j].Name == name {
				lib.Groups[i].Skills[j].Commit = commit
				return
			}
		}
	}
}

func writeLine(w io.Writer, s string) error {
	_, err := io.WriteString(w, s)
	return err
}
