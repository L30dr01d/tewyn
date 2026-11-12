package project

import (
	"errors"
	"os"
	"path/filepath"
)

// CopyDir copies src to a temp directory beside dst, then replaces dst.
// On a copy error, dst is unchanged and the temp directory is gone.
// The result is a real directory.
func CopyDir(src, dst string) error {
	tmp, err := os.MkdirTemp(filepath.Dir(dst), ".copy-*")
	if err != nil {
		return err
	}
	if err := os.CopyFS(tmp, os.DirFS(src)); err != nil {
		_ = os.RemoveAll(tmp)
		return err
	}
	return replaceDir(tmp, dst)
}

// replaceDir moves tmp onto dst. dst is renamed aside first so a failed
// rename does not delete the previous folder.
func replaceDir(tmp, dst string) error {
	aside := tmp + ".old"
	moved := true
	if err := os.Rename(dst, aside); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			_ = os.RemoveAll(tmp)
			return err
		}
		moved = false
	}
	if err := os.Rename(tmp, dst); err != nil {
		if moved {
			_ = os.Rename(aside, dst)
		}
		_ = os.RemoveAll(tmp)
		return err
	}
	if moved {
		return os.RemoveAll(aside)
	}
	return nil
}
