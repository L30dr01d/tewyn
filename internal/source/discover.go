// Package source discovers skills from a GitHub tree URL.
package source

import (
	"errors"
	"slices"
	"strings"
	"tewyn/internal/library"
)

// ErrNoSkills means that path has no skills.
var ErrNoSkills = errors.New("no skills at that path")

// Parsed is a GitHub tree URL split into repo, ref, and directory.
type Parsed struct {
	RepoURL, RepoName, Ref, Dir string
}

// Parse accepts only https://github.com/<owner>/<repo>/tree/<ref>/<path...>.
// Ref is one segment. Anything else returns ErrNoSkills.
func Parse(raw string) (Parsed, error) {
	rest, ok := strings.CutPrefix(raw, "https://github.com/")
	parts := strings.Split(rest, "/")
	if !ok || len(parts) < 5 || parts[2] != "tree" || strings.ContainsAny(rest, "?#") {
		return Parsed{}, ErrNoSkills
	}
	for _, s := range parts {
		if s == "" {
			return Parsed{}, ErrNoSkills
		}
	}
	repo := strings.TrimSuffix(parts[1], ".git")
	if repo == "" {
		return Parsed{}, ErrNoSkills
	}
	return Parsed{
		RepoURL:  "https://github.com/" + parts[0] + "/" + repo,
		RepoName: repo,
		Ref:      parts[3],
		Dir:      strings.Join(parts[4:], "/"),
	}, nil
}

// Discover returns the skills under a GitHub tree URL.
// A root SKILL.md is the skill and hides children. Otherwise each immediate child with SKILL.md is a skill.
func Discover(raw string, paths []string) (library.Group, error) {
	p, err := Parse(raw)
	if err != nil {
		return library.Group{}, err
	}
	if slices.Contains(paths, "SKILL.md") {
		name := p.Dir[strings.LastIndex(p.Dir, "/")+1:]
		return library.Group{Name: p.RepoName, Skills: []library.Skill{{Name: name, URL: raw}}}, nil
	}
	var skills []library.Skill
	for _, path := range paths {
		name, rest, ok := strings.Cut(path, "/")
		if !ok || rest != "SKILL.md" {
			continue
		}
		skills = append(skills, library.Skill{
			Name: name,
			URL:  p.RepoURL + "/tree/" + p.Ref + "/" + p.Dir + "/" + name,
		})
	}
	if len(skills) == 0 {
		return library.Group{}, ErrNoSkills
	}
	slices.SortFunc(skills, func(a, b library.Skill) int {
		return strings.Compare(a.Name, b.Name)
	})
	return library.Group{Name: p.RepoName, Skills: skills}, nil
}
