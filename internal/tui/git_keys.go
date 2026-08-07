package tui

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/vcs"
)

// Keys in the repository view.
//
// Push is the only outward-facing action in the whole app, and it is guarded twice: it is
// unreachable from the board, and inside here it takes a confirmation that names the
// destination URL. Anyone pressing the second key has read where their solutions are
// about to go.

func (m Model) handleGitKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if m.git.confirming {
		// Anything other than an explicit yes cancels. A confirmation that accepts
		// "enter" would be answered by a stray keypress.
		m.git.confirming = false
		if key != "y" && key != "Y" {
			return m, status("Push cancelled.", false)
		}
		m.git.pushing = true
		return m, tea.Batch(m.pushGit(), status("Pushing to "+m.gitDestination()+"…", false))
	}

	switch key {
	case "esc", "q", "ctrl+c":
		m.mode = modeBoard
		return m, nil

	case "R":
		m.git = gitPane{loading: true}
		return m, m.loadGit()

	case "p":
		if m.git.pushing {
			return m, nil
		}
		if !m.canPush() {
			return m, status(pushRefusal(m.git), false)
		}
		m.git.confirming = true
		return m, nil
	}
	return m, nil
}

// pushRefusal explains why p did nothing.
//
// A key that silently does nothing reads as a broken key. Each of these is a different
// situation with a different fix, so each says which one it is.
func pushRefusal(g gitPane) string {
	switch {
	case g.err != nil:
		return "No repository to push from."
	case g.status.Detached:
		return "HEAD is detached — check out a branch first."
	case g.status.Upstream == "" && len(g.log) == 0:
		return "Nothing committed yet."
	case g.status.Upstream == "":
		return "No remote configured. Add one, or set git.remote in your config."
	default:
		return "Nothing to push — the remote already has every commit."
	}
}

// handlePushed reports the outcome and refreshes, so ahead/behind is never stale.
func (m Model) handlePushed(msg gitPushedMsg) (tea.Model, tea.Cmd) {
	m.git.pushing = false
	if msg.err != nil {
		return m, status(pushError(msg.err), true)
	}
	m.git.loading = true
	return m, tea.Batch(m.loadGit(), status(pushNote(msg.ahead), false))
}

// pushError translates git's failure into something actionable.
//
// The credential case is the one worth naming: leetui runs git with terminal prompts
// disabled, because a hidden prompt on the alternate screen hangs the app with no way to
// answer it. The user's own shell can still push, and that is the fix.
func pushError(err error) string {
	switch {
	case errors.Is(err, vcs.ErrNoUpstream):
		return "This branch has no upstream. Set git.remote in your config to publish it."
	case containsAny(err.Error(), "could not read Username", "Authentication failed",
		"terminal prompts disabled", "Permission denied"):
		return "git could not authenticate. Push from your own shell — leetui never prompts for credentials."
	default:
		return "Push failed: " + err.Error()
	}
}

func pushNote(ahead int) string {
	if ahead <= 0 {
		return "Pushed."
	}
	return "Pushed " + plural(ahead, "commit") + "."
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
