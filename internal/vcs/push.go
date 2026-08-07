package vcs

import (
	"context"
	"fmt"
	"strings"
)

// Pushing.
//
// Nothing in this file is ever called by a background task, a sync, or an accepted
// verdict. Push happens because someone pressed a key (D-011). Publishing to a remote is
// visible to other people and cannot be taken back, so the app does not get to decide
// when it happens.

// Push sends the current branch to its upstream.
//
// Returns ErrNoUpstream when the branch has never been pushed — the caller should offer
// PushTo with a named remote rather than picking one, since guessing "origin" would set
// a tracking branch the user never asked for.
func (r Repo) Push(ctx context.Context) error {
	st, err := r.Status(ctx)
	if err != nil {
		return err
	}
	if st.Detached {
		return ErrDetached
	}
	if st.Upstream == "" {
		return ErrNoUpstream
	}
	_, err = run(ctx, r.Root, "push")
	return err
}

// PushTo sends the current branch to remote and sets it as the upstream.
//
// The first push of a branch. Separate from Push because it does two things — publishes,
// and changes the repository's configuration — and the caller should be naming both when
// it asks the user.
func (r Repo) PushTo(ctx context.Context, remote string) error {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return fmt.Errorf("push: no remote named")
	}
	st, err := r.Status(ctx)
	if err != nil {
		return err
	}
	if st.Detached {
		return ErrDetached
	}
	if st.Branch == "" {
		return ErrDetached
	}
	_, err = run(ctx, r.Root, "push", "--set-upstream", remote, st.Branch)
	return err
}

// Remotes lists the configured remote names.
func (r Repo) Remotes(ctx context.Context) ([]string, error) {
	out, err := run(ctx, r.Root, "remote")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, line := range strings.Split(out, "\n") {
		if n := strings.TrimSpace(line); n != "" {
			names = append(names, n)
		}
	}
	return names, nil
}

// RemoteURL returns the fetch URL for a remote, or "" if there is no such remote.
//
// Used to show the user where a push would go before they agree to it. A URL is the only
// part of this that is unambiguous — "origin" tells them nothing about which account or
// which host is about to receive their solutions.
func (r Repo) RemoteURL(ctx context.Context, remote string) string {
	out, _, err := runCode(ctx, r.Root, "remote", "get-url", remote)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}
