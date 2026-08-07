package solve

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Nano-AI/leetui/internal/config"
	"github.com/Nano-AI/leetui/internal/vcs"
	"github.com/Nano-AI/leetui/internal/workspace"
)

// Committing an accepted solution.
//
// Here rather than in the TUI for the same reason Prepare is: the CLI's `leetui submit`
// and the app's `s` key must produce identical commits, and neither frontend should own
// the rule for which files go in.

// Solved describes an accepted submission, for the commit message and the file list.
type Solved struct {
	ID    int
	Slug  string
	Title string

	// Lang is the display name, e.g. "Go".
	Lang string

	// Filename is the solution file inside the problem folder, e.g. "solution.go".
	Filename string

	// Runtime and Percentile come from the judge. Both optional; an absent figure is
	// left out of the subject rather than guessed at.
	Runtime    string
	Percentile float64
}

// Commit records an accepted solution in the workspace repository.
//
// Returns ErrNotRepo when the workspace has never been init'd, which is the normal state
// for a new user — the caller should treat it as "nothing to do", not as a failure.
// ErrNothingToCommit means the files already match HEAD, which is what a second accepted
// submission of an unedited solution produces.
//
// Only ever called on an Accepted verdict (D-011). A wrong answer is not a milestone.
func Commit(ctx context.Context, root string, git config.Git, s Solved) (vcs.CommitResult, error) {
	ws, err := workspace.New(root)
	if err != nil {
		return vcs.CommitResult{}, err
	}
	repo, err := vcs.Open(ctx, ws.Root)
	if err != nil {
		return vcs.CommitResult{}, err
	}

	return repo.Commit(ctx, vcs.CommitRequest{
		Paths:  commitPaths(ws, git, s),
		Branch: git.Branch,
		Message: vcs.Message{
			ID: s.ID, Title: s.Title, Lang: s.Lang,
			Runtime: s.Runtime, Percentile: s.Percentile,
		}.String(),
	})
}

// commitPaths lists what belongs in a solution commit.
//
// The solution, the statement, and the test cases: enough that the folder reads on
// GitHub without leetui. Notes are opt-out via git.commit_notes — they are the user's own
// prose and some people would rather not publish a half-finished thought.
//
// Missing files are dropped rather than passed through: `git add` fails the whole
// invocation on a pathspec that matches nothing, so one absent notes.md would take the
// commit down with it. A problem prepared without metaData has no testcases.txt, and that
// is a perfectly ordinary state to be committing from.
func commitPaths(ws workspace.Workspace, git config.Git, s Solved) []string {
	dir := ws.Dir(s.ID, s.Slug)
	want := []string{workspace.ReadmeFile, workspace.TestcasesFile}
	if s.Filename != "" {
		want = append(want, s.Filename)
	}
	if git.CommitNotes {
		want = append(want, workspace.NotesFile)
	}

	var paths []string
	for _, name := range want {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			paths = append(paths, path)
		}
	}
	return paths
}
