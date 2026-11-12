package source

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// New returns a Tree that reads a repository with git.
func New() Tree { return gitTree{} }

type gitTree struct{}

func (gitTree) List(ctx context.Context, repoURL, ref, dir string) (string, []string, error) {
	checkout, commit, err := fetch(ctx, repoURL, ref)
	if checkout != "" {
		defer func() { _ = os.RemoveAll(checkout) }()
	}
	if err != nil {
		return "", nil, err
	}
	out, err := git(ctx, checkout, "ls-tree", "-r", "-z", "--name-only", commit+":"+dir)
	if err != nil {
		if ctx.Err() != nil {
			return "", nil, ctx.Err()
		}
		if missingPath(err) {
			return "", nil, ErrPathChanged
		}
		return "", nil, err
	}
	var paths []string
	for _, p := range bytes.Split(out, []byte{0}) {
		if len(p) > 0 {
			paths = append(paths, string(p))
		}
	}
	return commit, paths, nil
}

func (gitTree) Download(ctx context.Context, repoURL, commit, dir, dest string) error {
	checkout, _, err := fetch(ctx, repoURL, commit)
	if checkout != "" {
		defer func() { _ = os.RemoveAll(checkout) }()
	}
	if err != nil {
		return err
	}
	raw, err := git(ctx, checkout, "archive", "--format=tar", commit+":"+dir)
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(filepath.Dir(dest), ".tewyn-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	if err = untar(raw, tmp); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	return replace(tmp, dest)
}

func missingPath(err error) bool {
	var exit *exec.ExitError
	return errors.As(err, &exit) && bytes.Contains(exit.Stderr, []byte("Not a valid object name"))
}

func fetch(ctx context.Context, repoURL, ref string) (string, string, error) {
	dir, err := os.MkdirTemp("", "tewyn-")
	if err != nil {
		return "", "", err
	}
	if _, err = git(ctx, dir, "init", "-q"); err != nil {
		return dir, "", err
	}
	if _, err = git(ctx, dir, "fetch", "--depth", "1", "-q", repoURL, ref); err != nil {
		return dir, "", err
	}
	raw, err := git(ctx, dir, "rev-parse", "--verify", "FETCH_HEAD^{commit}")
	if err != nil {
		return dir, "", err
	}
	return dir, strings.TrimSpace(string(raw)), nil
}

// afterGit runs after each git command so tests can cancel between them.
var afterGit func([]string)

func git(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git")
	cmd.Args = append(cmd.Args, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
	out, err := cmd.Output()
	if afterGit != nil {
		afterGit(args)
	}
	var exit *exec.ExitError
	if err == nil || errors.Is(err, context.Canceled) || !errors.As(err, &exit) {
		return out, err
	}
	if msg := strings.TrimSpace(string(exit.Stderr)); msg != "" {
		return out, &gitError{exit, msg}
	}
	return out, err
}

type gitError struct {
	err *exec.ExitError
	msg string
}

func (e *gitError) Error() string { return e.msg }
func (e *gitError) Unwrap() error { return e.err }

func untar(data []byte, dest string) error {
	root, err := os.OpenRoot(dest)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	tr := tar.NewReader(bytes.NewReader(data))
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		name := strings.TrimSuffix(hdr.Name, "/")
		switch hdr.Typeflag {
		case tar.TypeDir:
			if name == "" || name == "." {
				continue
			}
			if err = root.MkdirAll(name, 0o750); err != nil {
				return err
			}
		case tar.TypeReg:
			if dir := filepath.Dir(name); dir != "." {
				if err = root.MkdirAll(dir, 0o750); err != nil {
					return err
				}
			}
			// ponytail: mode 0644; use hdr.Mode when scripts must stay executable
			f, err := root.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			_, err = io.CopyN(f, tr, hdr.Size)
			cerr := f.Close()
			if err != nil {
				return err
			}
			if cerr != nil {
				return cerr
			}
		}
	}
}

func replace(tmp, dest string) error {
	old := dest + ".old"
	if _, err := os.Lstat(dest); errors.Is(err, os.ErrNotExist) {
		if err = os.Rename(old, dest); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	} else if err != nil {
		return err
	}
	if err := os.RemoveAll(old); err != nil {
		return err
	}
	if err := os.Rename(dest, old); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		return errors.Join(err, os.Rename(old, dest))
	}
	return os.RemoveAll(old)
}
