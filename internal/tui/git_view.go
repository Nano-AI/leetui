package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Nano-AI/leetui/internal/tui/theme"
	"github.com/Nano-AI/leetui/internal/vcs"
)

// The repository view.
//
// One screen answering three questions in the order they get asked: what branch am I on,
// what have I not committed, and what has the remote not got. The last of those is the
// reason this screen exists — solutions that live on one laptop are solutions one disk
// failure from gone.

// viewGit draws the repository.
func (m Model) viewGit() string {
	head := "\n " + lipgloss.NewStyle().Foreground(theme.Amber).Bold(true).
		Render(theme.Display("repository")) + "\n"

	if m.git.loading {
		return head + "\n " + theme.Meta.Render(
			"Reading "+fitPath(m.cfg.Workspace, maxInt(m.width-11, 12))+"…") + "\n"
	}
	if m.git.err != nil {
		return head + m.viewGitProblem() + gitFooter(false, false)
	}

	var b strings.Builder
	b.WriteString(head)
	b.WriteString(gitSection("branch", m.width))
	b.WriteString("   " + m.gitBranchLine() + "\n")

	b.WriteString(gitSection("working tree", m.width))
	b.WriteString(m.gitChanges())

	if len(m.git.log) > 0 {
		b.WriteString(gitSection("recent", m.width))
		for _, c := range m.git.log {
			b.WriteString(fmt.Sprintf("   %s  %s  %s\n",
				theme.Label.Width(9).Render(c.Short),
				theme.Body.Render(truncate(c.Subject, maxInt(m.width-34, 20))),
				theme.Meta.Render(c.When)))
		}
	}

	if m.git.confirming {
		b.WriteString("\n " + theme.Label.Render("Push to "+m.gitDestination()+"? ") +
			theme.Body.Render("y to publish, any other key to cancel") + "\n")
	}
	return b.String() + gitFooter(m.canPush(), m.git.confirming)
}

// viewGitProblem explains why there is nothing to show, with the fix beside it.
//
// A workspace without a repository is not a fault — it is what someone who has never run
// `git init` has, and most people have never wanted leetui to be the one to run it. So
// this states the situation and the exact command, and leetui does not offer to do it.
func (m Model) viewGitProblem() string {
	switch {
	case errors.Is(m.git.err, vcs.ErrNoGit):
		return "\n " + theme.Body.Render("git is not installed, so leetui cannot version your solutions.") + "\n"
	case errors.Is(m.git.err, vcs.ErrNotRepo):
		// The path is shown so it can be retyped, so it is shortened from the LEFT —
		// the tail identifies the directory, and a workspace root is what can be spared.
		return "\n " + theme.Body.Render("Your workspace is not a git repository yet.") +
			"\n\n " + theme.Meta.Render("cd "+fitPath(m.cfg.Workspace, maxInt(m.width-5, 12))) +
			"\n " + theme.Meta.Render("git init") +
			"\n\n " + theme.Meta.Render("Accepted solutions commit themselves once it is.") + "\n"
	default:
		return "\n " + theme.Label.Render(truncate(m.git.err.Error(), maxInt(m.width-2, 10))) + "\n"
	}
}

// gitBranchLine is the branch and its relationship to the remote.
func (m Model) gitBranchLine() string {
	st := m.git.status
	if st.Detached {
		return theme.Label.Render("detached HEAD") +
			theme.Meta.Render("  ·  commits here are easy to lose")
	}

	line := theme.Label.Render(st.Branch)
	switch {
	case st.Upstream == "":
		line += theme.Meta.Render("  ·  no upstream — nothing is tracking this branch")
	case st.Ahead > 0 && st.Behind > 0:
		line += theme.Label.Render(fmt.Sprintf("  ·  %s ahead, %s behind %s",
			plural(st.Ahead, "commit"), plural(st.Behind, "commit"), st.Upstream))
	case st.Ahead > 0:
		line += theme.Label.Render(fmt.Sprintf("  ·  %s not on %s",
			plural(st.Ahead, "commit"), st.Upstream))
	case st.Behind > 0:
		line += theme.Meta.Render(fmt.Sprintf("  ·  %s behind %s",
			plural(st.Behind, "commit"), st.Upstream))
	default:
		line += theme.Meta.Render("  ·  in sync with " + st.Upstream)
	}
	return line
}

// gitChanges lists uncommitted paths, capped.
func (m Model) gitChanges() string {
	st := m.git.status
	if st.Clean() {
		return "   " + theme.Meta.Render("clean — everything is committed") + "\n"
	}

	const maxRows = 8
	var b strings.Builder
	for i, c := range st.Changes {
		if i == maxRows {
			// Never silently truncate: a count the user cannot see is a count they
			// will assume is zero.
			b.WriteString("   " + theme.Meta.Render(
				fmt.Sprintf("… and %d more", len(st.Changes)-maxRows)) + "\n")
			break
		}
		b.WriteString(fmt.Sprintf("   %s  %s\n",
			theme.Label.Width(9).Render(changeWord(c)),
			theme.Body.Render(truncate(c.Path, maxInt(m.width-18, 20)))))
	}
	return b.String()
}

// changeWord names a change in words rather than git's XY codes.
//
// "M." and "??" are precise and mean nothing to someone who has not read the porcelain
// documentation, which is everyone using a LeetCode client.
func changeWord(c vcs.Change) string {
	switch {
	case c.Untracked:
		return "new"
	case c.Staged && c.Unstaged:
		return "part-staged"
	case c.Staged:
		return "staged"
	default:
		return "modified"
	}
}

// canPush reports whether there is anything to publish and somewhere to publish it.
func (m Model) canPush() bool {
	st := m.git.status
	if m.git.err != nil || st.Detached {
		return false
	}
	if st.Upstream == "" {
		// A first push needs a remote, and leetui refuses to pick one.
		return m.git.remote != "" && len(m.git.log) > 0
	}
	return st.Ahead > 0
}

// gitDestination is what the confirmation names — the URL when there is one.
func (m Model) gitDestination() string {
	if m.git.url != "" {
		return m.git.url
	}
	if m.git.status.Upstream != "" {
		return m.git.status.Upstream
	}
	return m.git.remote
}

func gitSection(name string, width int) string {
	return "\n " + theme.Utility.Render(theme.UtilityText(name)) + " " +
		theme.Rule.Render(strings.Repeat(theme.Chars().DashRule, maxInt(minInt(width, 72)-len(name)-3, 1))) + "\n"
}

func gitFooter(canPush, confirming bool) string {
	var keys string
	switch {
	case confirming:
		keys = theme.Label.Render("y") + theme.Body.Render(" push   ") +
			theme.Label.Render("esc") + theme.Body.Render(" cancel")
	case canPush:
		keys = theme.Label.Render("p") + theme.Body.Render(" push   ") +
			theme.Label.Render("R") + theme.Body.Render(" refresh   ") +
			theme.Label.Render("esc") + theme.Body.Render(" back")
	default:
		keys = theme.Label.Render("R") + theme.Body.Render(" refresh   ") +
			theme.Label.Render("esc") + theme.Body.Render(" back")
	}
	return "\n " + keys + "\n"
}

// commitNote phrases an auto-commit for the status line.
func commitNote(res vcs.CommitResult) string {
	return fmt.Sprintf("Committed %s as %s. Press v to push.",
		plural(res.Files, "file"), res.Short)
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
