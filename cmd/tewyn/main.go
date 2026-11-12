// Command tewyn manages a local skill library.
package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"tewyn/internal/agents"
	"tewyn/internal/library"
	"tewyn/internal/project"
	"tewyn/internal/source"
	"tewyn/internal/tui"
	"tewyn/internal/update"
)

const version = "1.0.0"

var helpText = "tewyn " + version + "\n" +
	"usage: tewyn [--add-skills <url>|-u|-v|-h]\n" +
	"\n" +
	"  tewyn                         open the project TUI\n" +
	"  tewyn --add-skills <url>      add a GitHub skill tree to the library\n" +
	"  tewyn -u                      refresh cached skills and copy them into the project\n" +
	"  tewyn -v, --version           print the version\n" +
	"  tewyn -h, --help              print this help\n"

// Main runs tewyn. No arguments opens the TUI. --add-skills writes the library. -u refreshes the cache and copies it into the project.
func Main(args []string, projectDir string, stdout, stderr io.Writer, getenv func(string) string, tree source.Tree) int {
	if len(args) == 1 && (args[0] == "-v" || args[0] == "--version") {
		return emit(stdout, 0, "tewyn "+version+"\n")
	}
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		return emit(stdout, 0, helpText)
	}
	if len(args) == 1 && args[0] == "-u" {
		if tree == nil {
			tree = source.New()
		}
		path, err := library.Path(getenv)
		if err != nil {
			return bail(stderr, err)
		}
		if err = update.Run(context.Background(), projectDir, path, tree, stdout); err != nil {
			return bail(stderr, err)
		}
		return 0
	}
	if len(args) == 1 && args[0] == "--add-skills" {
		return emit(stderr, 1, "error: no skills at that path\n")
	}
	if len(args) == 0 {
		m, err := tui.LoadModel(projectDir, getenv)
		if err != nil {
			return bail(stderr, err)
		}
		err = tui.Start(m, func(ctx context.Context, sel project.Selection, progress func(string)) error {
			path, err := library.Path(getenv)
			if err != nil {
				return err
			}
			_, err = project.ApplyNotify(ctx, projectDir, library.SkillsDir(path), sel, progress)
			return err
		}, func(want map[string]bool) error {
			return agents.Apply(projectDir, want)
		})
		if err != nil {
			return bail(stderr, err)
		}
		return 0
	}
	if len(args) != 2 || args[0] != "--add-skills" {
		return emit(stderr, 2, "error: usage: tewyn [--add-skills <url>|-u|-v|-h]\n")
	}
	parsed, err := source.Parse(args[1])
	if err != nil {
		return bail(stderr, err)
	}
	if tree == nil {
		tree = source.New()
	}
	ctx := context.Background()
	_, paths, err := tree.List(ctx, parsed.RepoURL, parsed.Ref, parsed.Dir)
	if errors.Is(err, source.ErrPathChanged) {
		err = source.ErrNoSkills
	}
	if err != nil {
		return bail(stderr, err)
	}
	group, err := source.Discover(args[1], paths)
	if err != nil {
		return bail(stderr, err)
	}
	path, err := library.Path(getenv)
	if err != nil {
		return bail(stderr, err)
	}
	lib, err := library.Load(path)
	if err != nil {
		return bail(stderr, err)
	}
	probe := lib
	if err = probe.Add(group); err != nil {
		return bail(stderr, err)
	}
	if err = os.MkdirAll(library.SkillsDir(path), 0o750); err != nil {
		return bail(stderr, err)
	}
	var created []string
	for i := range group.Skills {
		skill := &group.Skills[i]
		parsedSkill, err := source.Parse(skill.URL)
		if err != nil {
			removeDirs(created)
			return bail(stderr, err)
		}
		commit, _, err := tree.List(ctx, parsedSkill.RepoURL, parsedSkill.Ref, parsedSkill.Dir)
		if err != nil {
			removeDirs(created)
			return bail(stderr, err)
		}
		if _, err = io.WriteString(stdout, "downloading "+skill.Name+"\n"); err != nil {
			removeDirs(created)
			return 1
		}
		dest := filepath.Join(library.SkillsDir(path), skill.Name)
		if err = tree.Download(ctx, parsedSkill.RepoURL, commit, parsedSkill.Dir, dest); err != nil {
			removeDirs(append(created, dest))
			return bail(stderr, err)
		}
		created = append(created, dest)
		skill.Commit = commit
	}
	if err = lib.Add(group); err != nil {
		return bail(stderr, err)
	}
	if err = library.Save(path, lib); err != nil {
		return bail(stderr, err)
	}
	msg := "added group " + group.Name + "\n"
	for _, skill := range group.Skills {
		msg += "  " + skill.Name + "\n"
	}
	if home := getenv("HOME"); home != "" && path == home+"/.config/tewyn/library.json" {
		path = "~/.config/tewyn/library.json"
	}
	return emit(stdout, 0, msg+"saved to "+path+"\n")
}

func removeDirs(dirs []string) {
	for _, dir := range dirs {
		_ = os.RemoveAll(dir)
	}
}

func bail(w io.Writer, err error) int {
	return emit(w, 1, "error: "+err.Error()+"\n")
}

func emit(w io.Writer, code int, msg string) int {
	if _, err := io.WriteString(w, msg); err != nil {
		return 1
	}
	return code
}

func main() {
	dir, err := os.Getwd()
	if err != nil {
		os.Exit(bail(os.Stderr, err))
	}
	os.Exit(Main(os.Args[1:], dir, os.Stdout, os.Stderr, os.Getenv, nil))
}
