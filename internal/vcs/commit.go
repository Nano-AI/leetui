package vcs

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// CommitRequest is one guarded commit.
type CommitRequest struct {
	// Paths to stage. Absolute or relative to the repository root; anything outside the
	// repository is refused rather than silently skipped.
	//
	// Deliberately explicit — this package never runs `git add -A`. The workspace is the
	// user's own repository and may hold work leetui knows nothing about.
	Paths []string

	// Message is the full commit message, subject line first.
	Message string

	// Branch is the branch the caller expects to be on. Empty accepts whatever is
	// checked out; set, a mismatch aborts with ErrWrongBranch (D-011).
	Branch string
}

// CommitResult describes what landed.
type CommitResult struct {
	Short string
	Files int
}

// Commit stages the requested paths and commits them, refusing anything unexpected.
//
// The guard runs before anything is staged, so a refusal leaves the index exactly as it
// was. That matters: the user may have had their own work staged in the next pane, and
// leetui aborting is not a reason to disturb it.
func (r Repo) Commit(ctx context.Context, req CommitRequest) (CommitResult, error) {
	if strings.TrimSpace(req.Message) == "" {
		return CommitResult{}, fmt.Errorf("commit: empty message")
	}
	rel, err := r.relative(req.Paths)
	if err != nil {
		return CommitResult{}, err
	}
	if len(rel) == 0 {
		return CommitResult{}, ErrNothingToCommit
	}

	st, err := r.Status(ctx)
	if err != nil {
		return CommitResult{}, err
	}
	if st.Detached {
		return CommitResult{}, ErrDetached
	}
	if req.Branch != "" && st.Branch != req.Branch {
		return CommitResult{}, fmt.Errorf("%w: on %q, expected %q",
			ErrWrongBranch, st.Branch, req.Branch)
	}

	if _, err := run(ctx, r.Root, append([]string{"add", "--"}, rel...)...); err != nil {
		return CommitResult{}, err
	}

	staged, err := r.stagedPaths(ctx, rel)
	if err != nil {
		return CommitResult{}, err
	}
	if len(staged) == 0 {
		return CommitResult{}, ErrNothingToCommit
	}

	// Only the requested paths are committed, even if the user has other things staged.
	// `commit -- <paths>` bypasses the index for everything else, which is the whole
	// point: auto-commit must never sweep up work it was not asked about.
	args := append([]string{"commit", "-m", req.Message, "--"}, staged...)
	if _, err := run(ctx, r.Root, args...); err != nil {
		return CommitResult{}, err
	}

	short, err := run(ctx, r.Root, "rev-parse", "--short", "HEAD")
	if err != nil {
		return CommitResult{Files: len(staged)}, nil
	}
	return CommitResult{Short: strings.TrimSpace(short), Files: len(staged)}, nil
}

// stagedPaths narrows the request to what actually differs from HEAD.
//
// Re-submitting an unchanged solution is the common case — accepted twice, same file —
// and `git commit` on nothing at all exits non-zero with a message about the working
// tree being clean. Asking first turns that into ErrNothingToCommit, which the caller
// can report as "already committed" instead of as a failure.
func (r Repo) stagedPaths(ctx context.Context, rel []string) ([]string, error) {
	args := append([]string{"diff", "--cached", "--name-only", "-z", "--"}, rel...)
	out, err := run(ctx, r.Root, args...)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, p := range strings.Split(strings.TrimSuffix(out, "\x00"), "\x00") {
		if p != "" {
			paths = append(paths, p)
		}
	}
	return paths, nil
}

// relative converts paths to repository-relative form and rejects any that escape it.
//
// A path outside the repository is a bug in the caller, not a user error, but it would
// present as git committing something from elsewhere in the filesystem — worth refusing
// loudly.
//
// Both sides are resolved through symlinks first. On macOS the two spellings of the same
// directory (/var/… and /private/var/…) are routine, and comparing them textually
// reports every path in the repository as outside it.
func (r Repo) relative(paths []string) ([]string, error) {
	root := resolve(r.Root)
	var out []string
	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}
		// A relative path is relative to the REPOSITORY, not to the process's working
		// directory. leetui is a TUI: its cwd is wherever the user happened to launch
		// it from, which has nothing to do with where the workspace lives.
		abs := p
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(root, abs)
		}
		rel, err := filepath.Rel(root, resolve(abs))
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("%s is outside the repository", p)
		}
		out = append(out, filepath.ToSlash(rel))
	}
	return out, nil
}

// resolve expands symlinks, falling back to the input.
//
// A file that does not exist yet cannot be resolved, and that is a normal case here —
// the caller may be staging a deletion. Resolving the parent instead keeps the common
// /var vs /private/var mismatch handled without failing on a missing leaf.
func resolve(path string) string {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		return real
	}
	dir, base := filepath.Split(path)
	if real, err := filepath.EvalSymlinks(filepath.Clean(dir)); err == nil {
		return filepath.Join(real, base)
	}
	return path
}
