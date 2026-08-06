package solve

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Finding which problem a command is about.
//
// The CLI has to work from inside an editor, where the natural thing to type is the file
// you are looking at rather than a slug you would have to remember:
//
//	:!leetui run %                  the buffer's path
//	leetui run                      from inside the problem folder
//	leetui run two-sum              the slug, from anywhere
//
// All three resolve to the same problem. Supporting only the last would make the nvim
// keymap a lookup exercise instead of one line.

// ErrNoProblem means nothing identified a problem.
var ErrNoProblem = errors.New("no problem given, and the current directory is not a problem folder")

// folderName matches a problem folder: a zero-padded id, a dash, then the slug.
var folderName = regexp.MustCompile(`^(\d{1,5})-(.+)$`)

// Locate resolves an argument to a problem slug.
//
// arg may be a slug ("two-sum"), a folder name ("0001-two-sum"), a path to either the
// folder or a file inside it, or empty to mean the working directory.
func Locate(arg string) (string, error) {
	if arg == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("read the working directory: %w", err)
		}
		if slug := slugFromPath(cwd); slug != "" {
			return slug, nil
		}
		return "", ErrNoProblem
	}

	// A path that exists on disk is treated as one, so a slug that happens to match a
	// local directory name cannot be misread.
	if _, err := os.Stat(arg); err == nil {
		abs, err := filepath.Abs(arg)
		if err != nil {
			return "", fmt.Errorf("resolve %s: %w", arg, err)
		}
		if slug := slugFromPath(abs); slug != "" {
			return slug, nil
		}
		return "", fmt.Errorf("%s is not inside a problem folder: %w", arg, ErrNoProblem)
	}

	// Not a path. A bare folder name carries its id; strip it.
	if m := folderName.FindStringSubmatch(arg); m != nil {
		return m[2], nil
	}
	return strings.TrimSpace(arg), nil
}

// slugFromPath walks up from a file or directory to the problem folder that contains it,
// returning its slug. Returns "" when there is none.
//
// Walking up rather than checking one level is what makes a path to a nested file work —
// and Go problem folders in particular can hold generated files a user might be editing.
func slugFromPath(path string) string {
	if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
		path = filepath.Dir(path)
	}
	for {
		if m := folderName.FindStringSubmatch(filepath.Base(path)); m != nil {
			return m[2]
		}
		parent := filepath.Dir(path)
		if parent == path {
			return "" // reached the filesystem root
		}
		path = parent
	}
}
