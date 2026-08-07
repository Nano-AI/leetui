package tui

import (
	"context"
	"errors"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/solve"
	"github.com/Nano-AI/leetui/internal/vcs"
)

// The git pane's state and the commands that fill it.
//
// Two things happen here and they have opposite temperaments. Committing is automatic on
// an accepted verdict, because the moment work is provably correct is worth capturing and
// asking would interrupt the one part of this app that feels good. Pushing is never
// automatic (D-011): it is visible to other people and cannot be undone.

// gitLogLines is how much history the pane shows.
//
// Enough to recognise the last session's work, few enough that the pane stays a glance.
const gitLogLines = 8

// gitPane is everything the repository view draws.
type gitPane struct {
	loading bool

	// err is the reason there is nothing to show. Held rather than raised as a status
	// line because "not a git repository" is the normal state for a new workspace and
	// wants explaining in place, with the fix beside it.
	err error

	status vcs.Status
	log    []vcs.Commit

	// remote and url describe where a push would go. The URL is the part that matters:
	// "origin" tells nobody which account is about to receive their solutions.
	remote string
	url    string

	// confirming is the armed state between pressing p and it actually pushing.
	confirming bool
	pushing    bool
}

type (
	// gitLoadedMsg carries a repository read.
	gitLoadedMsg struct {
		pane gitPane
	}
	// gitCommittedMsg reports an auto-commit's outcome.
	gitCommittedMsg struct {
		res vcs.CommitResult
		err error
	}
	// gitPushedMsg reports a push's outcome.
	gitPushedMsg struct {
		ahead int
		err   error
	}
)

// openGit switches to the repository view and starts the read.
func (m Model) openGit() (tea.Model, tea.Cmd) {
	m.mode = modeGit
	m.git = gitPane{loading: true}
	return m, m.loadGit()
}

// loadGit reads the workspace repository.
func (m Model) loadGit() tea.Cmd {
	root := m.cfg.Workspace
	want := m.cfg.Git.Remote

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		repo, err := vcs.Open(ctx, root)
		if err != nil {
			return gitLoadedMsg{pane: gitPane{err: err}}
		}

		p := gitPane{}
		if p.status, err = repo.Status(ctx); err != nil {
			return gitLoadedMsg{pane: gitPane{err: err}}
		}
		p.log, _ = repo.Log(ctx, gitLogLines)
		p.remote, p.url = pushTarget(ctx, repo, want, p.status)
		return gitLoadedMsg{pane: p}
	}
}

// pushTarget works out where a push would go, and says so in the user's terms.
//
// Preference order: the configured remote, then the upstream's remote, then the only
// remote if there is exactly one. Never a guess among several — a workspace with both a
// personal and a work remote must not have leetui choose.
func pushTarget(ctx context.Context, repo vcs.Repo, want string, st vcs.Status) (name, url string) {
	if want != "" {
		return want, repo.RemoteURL(ctx, want)
	}
	if name, _, ok := strings.Cut(st.Upstream, "/"); ok && name != "" {
		return name, repo.RemoteURL(ctx, name)
	}
	if names, err := repo.Remotes(ctx); err == nil && len(names) == 1 {
		return names[0], repo.RemoteURL(ctx, names[0])
	}
	return "", ""
}

// pushGit publishes the current branch.
//
// Only ever reached from the confirmation inside the git pane.
func (m Model) pushGit() tea.Cmd {
	root := m.cfg.Workspace
	remote := m.git.remote
	ahead := m.git.status.Ahead
	hasUpstream := m.git.status.Upstream != ""

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		repo, err := vcs.Open(ctx, root)
		if err != nil {
			return gitPushedMsg{err: err}
		}
		if hasUpstream {
			return gitPushedMsg{ahead: ahead, err: repo.Push(ctx)}
		}
		return gitPushedMsg{ahead: ahead, err: repo.PushTo(ctx, remote)}
	}
}

// commitAccepted commits a solution the judge has just accepted.
//
// Returns nil when the feature is off or there is nothing identifying the problem, so
// the caller can batch it unconditionally.
func (m Model) commitAccepted(j leetcode.Judgement) tea.Cmd {
	if !m.cfg.Git.CommitOnAccepted || m.detail == nil {
		return nil
	}
	root, git := m.cfg.Workspace, m.cfg.Git
	s := solve.Solved{
		ID: m.detail.NumericID, Slug: m.detail.Slug, Title: m.detail.Title,
		Lang: m.lang.Display, Filename: m.lang.Filename(),
		Runtime: j.Runtime, Percentile: j.RuntimePercentile,
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		res, err := solve.Commit(ctx, root, git, s)
		return gitCommittedMsg{res: res, err: err}
	}
}

// handleCommitted reports an auto-commit, and stays quiet about the two outcomes that
// are not news.
//
// A workspace that is not a repository is the normal state for someone who has never
// wanted one, and re-submitting an unchanged solution reaching "nothing to commit" is
// routine. Neither is a thing the user did wrong, and a red status line after an
// Accepted verdict would land as if it were.
func (m Model) handleCommitted(msg gitCommittedMsg) (tea.Model, tea.Cmd) {
	switch {
	case errors.Is(msg.err, vcs.ErrNotRepo), errors.Is(msg.err, vcs.ErrNoGit),
		errors.Is(msg.err, vcs.ErrNothingToCommit):
		return m, nil
	case msg.err != nil:
		return m, status("Not committed: "+msg.err.Error(), true)
	}
	return m, status(commitNote(msg.res), false)
}
