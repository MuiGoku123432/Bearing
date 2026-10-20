package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	bearing "github.com/MuiGoku123432/Bearing"
)

func readInstallFile(root *os.Root, path string) ([]byte, fs.FileMode, error) {
	parts := strings.Split(filepath.Clean(path), string(filepath.Separator))
	for i := range parts {
		info, err := root.Lstat(filepath.Join(parts[:i+1]...))
		if errors.Is(err, fs.ErrNotExist) {
			return nil, 0600, nil
		}
		if err != nil {
			return nil, 0, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, 0, fmt.Errorf("%s is a symlink; installation needs ordinary files and directories", path)
		}
		if i < len(parts)-1 && !info.IsDir() {
			return nil, 0, fmt.Errorf("%s has a non-directory parent", path)
		}
		if i == len(parts)-1 {
			if !info.Mode().IsRegular() || info.Size() > 1<<20 {
				return nil, 0, fmt.Errorf("%s must be a regular file of at most 1 MiB", path)
			}
			data, err := root.ReadFile(path)
			return data, info.Mode().Perm(), err
		}
	}
	return nil, 0, fmt.Errorf("missing file path")
}

func writeInstallFile(root *os.Root, path string, data []byte, mode fs.FileMode) error {
	if err := root.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	// Stage beside the destination so replacement stays on the same filesystem.
	temporary := filepath.Join(filepath.Dir(path), ".bearing-install-"+bearing.NewID())
	file, err := root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer root.Remove(temporary)
	if _, err = file.Write(data); err == nil {
		err = file.Chmod(mode)
	}
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return root.Rename(temporary, path)
}
