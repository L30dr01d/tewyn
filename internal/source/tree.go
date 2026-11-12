package source

import (
	"context"
	"errors"
)

// ErrPathChanged means dir is absent from the repository tree.
var ErrPathChanged = errors.New("path changed")

// Tree lists and downloads one directory of a git repository.
// List paths are relative to dir. When dir is absent, List returns ErrPathChanged.
type Tree interface {
	// List returns the commit and paths relative to dir.
	List(ctx context.Context, repoURL, ref, dir string) (commit string, paths []string, err error)
	// Download copies the directory at commit into dest.
	Download(ctx context.Context, repoURL, commit, dir, dest string) error
}
