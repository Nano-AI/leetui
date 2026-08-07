// Package vcs is leetui's git integration: read the workspace's status, commit a
// solution the judge has accepted, and push when the user asks for it.
//
// It shells out to the `git` binary rather than linking a library (D-011). The user's
// credentials, commit signing, hooks, `includeIf` blocks, and pager config all live in
// their git setup; a reimplementation inherits none of it and would be strictly worse at
// the one job that matters — producing commits indistinguishable from the ones they make
// by hand.
//
// Two rules shape everything here:
//
//   - **Committing is guarded, pushing is explicit.** An accepted verdict may auto-commit
//     because the moment work is provably correct is worth capturing. Nothing in this
//     package ever pushes on its own — Push is only ever called from a keypress.
//   - **Refuse rather than guess.** Wrong branch, no repository, nothing staged: each is
//     a distinct error the caller can name to the user. Writing a commit somewhere
//     unexpected is worse than writing none.
package vcs

import (
	"context"
	"errors"
	"os/exec"
	"strings"
)

// Errors callers branch on. Everything else arrives as a wrapped message from git
// itself, which is usually more informative than anything this package would invent.
var (
	// ErrNoGit means the binary is missing. Every operation checks, because a TUI that
	// reports "exec: git: executable file not found in $PATH" has failed twice.
	ErrNoGit = errors.New("git is not installed")

	// ErrNotRepo means the workspace has never had `git init` run in it. This is the
	// expected state for a new user, not a fault — the caller should offer, not scold.
	ErrNotRepo = errors.New("not a git repository")

	// ErrWrongBranch means git.branch is configured and HEAD is somewhere else. The
	// commit is refused: a solution landing on a feature branch by accident is a mess to
	// untangle later.
	ErrWrongBranch = errors.New("checked out branch is not the configured one")

	// ErrDetached means HEAD is not on a branch at all — mid-rebase, or a bare checkout.
	// Committing here creates work that is trivially lost.
	ErrDetached = errors.New("HEAD is detached")

	// ErrNothingToCommit means the files were already committed. Not a failure: the
	// second accepted submission of an unchanged solution reaches this every time.
	ErrNothingToCommit = errors.New("nothing to commit")

	// ErrNoUpstream means the branch has no remote tracking branch, so push has no
	// default destination.
	ErrNoUpstream = errors.New("branch has no upstream")
)

// Repo is a git repository, identified by its top level.
type Repo struct {
	// Root is the absolute path git reported for the working tree — the directory
	// holding .git, which is not necessarily the directory that was opened.
	Root string
}

// Open finds the repository containing dir.
//
// dir is typically the workspace root, but a problem folder works too: git walks up. A
// missing directory and an un-initialised one both return ErrNotRepo, since from the
// caller's side they mean the same thing — there is nothing to commit into yet.
func Open(ctx context.Context, dir string) (Repo, error) {
	if !Available() {
		return Repo{}, ErrNoGit
	}
	out, err := run(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return Repo{}, ErrNotRepo
	}
	root := strings.TrimSpace(out)
	if root == "" {
		return Repo{}, ErrNotRepo
	}
	return Repo{Root: root}, nil
}

// Available reports whether the git binary is on PATH.
//
// Separate from Open so a caller can tell "you have no git" from "you have no repo" and
// say something useful about each.
func Available() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// Init runs `git init` in dir and returns the repository.
//
// Called only from an explicit user action. leetui never initialises a repository
// behind the user's back: a stray .git in a directory they did not intend to version is
// a real annoyance, and the workspace default is inside their home directory.
func Init(ctx context.Context, dir string) (Repo, error) {
	if !Available() {
		return Repo{}, ErrNoGit
	}
	if _, err := run(ctx, dir, "init"); err != nil {
		return Repo{}, err
	}
	return Open(ctx, dir)
}
