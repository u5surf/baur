package fs

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
)

// DirsExist runs DirExists for multiple paths.
func DirsExist(paths []string) error {
	for _, path := range paths {
		err := DirExists(path)
		if err != nil {
			return err
		}
	}

	return nil
}

// DirExists returns nil if the path is a directory.
func DirExists(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("'%s' does not exist", path)
		}
		return err
	}

	if fi.IsDir() {
		return nil
	}

	return fmt.Errorf("'%s' is not a directory", path)
}

// FindFileInParentDirs finds a directory that contains filename. The function
// starts searching in startPath and then checks recursively each parent
// directory for the file. It returns the absolute path to the first found
// directory contains the file.
// If it reaches the root directory without finding the file it returns
// os.ErrNotExist
func FindFileInParentDirs(startPath, filename string) (string, error) {
	searchDir := startPath

	for {
		p := path.Join(searchDir, filename)

		_, err := os.Stat(p)
		if err == nil {
			abs, err := filepath.Abs(p)
			if err != nil {
				return "", errors.Wrapf(err,
					"could not get absolute path of %v", p)
			}

			return abs, nil
		}

		if !os.IsNotExist(err) {
			return "", err
		}

		// TODO: how to detect OS independent if reached the root dir
		if searchDir == "/" {
			return "", os.ErrNotExist
		}

		searchDir = path.Join(searchDir, "..")
	}
}

// FindFilesInSubDir returns all directories that contain filename that are in
// searchDir. The function descends up to maxdepth levels of directories below
// searchDir
func FindFilesInSubDir(searchDir, filename string, maxdepth int) ([]string, error) {
	var result []string
	glob := ""

	for i := 0; i <= maxdepth; i++ {
		globPath := path.Join(searchDir, glob, filename)

		matches, err := filepath.Glob(globPath)
		if err != nil {
			return nil, err
		}

		for _, m := range matches {
			abs, err := filepath.Abs(m)
			if err != nil {
				return nil, errors.Wrapf(err, "could not get absolute path of %s", m)
			}

			result = append(result, abs)
		}

		glob += "*/"
	}

	return result, nil
}
