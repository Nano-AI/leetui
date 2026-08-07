package vcs

import (
	"context"
	"fmt"
)

// Status is the repository state the git pane draws.
type Status struct {
	// Branch is the checked-out branch, or "" when Detached.
	Branch   string
	Detached bool

	// Upstream is the tracking branch, e.g. "origin/main", or "" when there is none.
	Upstream string

	// Ahead and Behind count commits relative to Upstream. Both zero with no upstream.
	Ahead  int
	Behind int

	// Changes is every path git considers modified, staged, or untracked.
	Changes []Change
}

// Change is one path git reports as not matching HEAD.
type Change struct {
	Path string

	// Staged and Unstaged can both be true: a file edited after being added.
	Staged   bool
	Unstaged bool

	// Untracked means git has never seen this path. Kept distinct from Unstaged
	// because the pane says "new" rather than "modified", and because a fresh
	// workspace is entirely untracked — reporting that as modification is a lie.
	Untracked bool
}

// Clean reports whether the working tree matches HEAD.
func (s Status) Clean() bool { return len(s.Changes) == 0 }

// Dirty counts paths that differ from HEAD, for the pane's summary line.
func (s Status) Dirty() int { return len(s.Changes) }

// Unpushed reports whether there is work the remote does not have.
//
// True with no upstream as well: commits that no remote is tracking are exactly the
// ones most in danger of being lost with the machine.
func (s Status) Unpushed() bool { return s.Ahead > 0 || (s.Upstream == "" && s.Branch != "") }

// Status reads the repository state.
//
// Uses porcelain v2, which is the documented-stable machine format — v1's `XY path`
// cannot express ahead/behind, and `git status` without --porcelain is explicitly not a
// format to parse.
func (r Repo) Status(ctx context.Context) (Status, error) {
	out, err := run(ctx, r.Root, "status", "--porcelain=v2", "--branch", "-z")
	if err != nil {
		return Status{}, err
	}
	return parsePorcelain(out), nil
}

// HasCommits reports whether HEAD points at anything.
//
// A repository that has been init'd but never committed has no HEAD, and several git
// commands fail confusingly against it. Worth asking before showing a log.
func (r Repo) HasCommits(ctx context.Context) bool {
	_, code, _ := runCode(ctx, r.Root, "rev-parse", "--verify", "HEAD")
	return code == 0
}

// Log returns the last n commit subjects, newest first.
func (r Repo) Log(ctx context.Context, n int) ([]Commit, error) {
	if n <= 0 {
		return nil, nil
	}
	if !r.HasCommits(ctx) {
		return nil, nil
	}
	out, err := run(ctx, r.Root,
		"log", fmt.Sprintf("-%d", n), "--format=%h%x00%s%x00%cr", "-z")
	if err != nil {
		return nil, err
	}
	return parseLog(out), nil
}

// Commit is one entry in the log.
type Commit struct {
	Short   string
	Subject string

	// When is git's relative form ("3 minutes ago"). Relative rather than a timestamp
	// because the pane's question is "did that just commit?", not "when exactly".
	When string
}
