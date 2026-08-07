package solve

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/config"
	"github.com/Nano-AI/leetui/internal/vcs"
	"github.com/Nano-AI/leetui/internal/workspace"
)

// solvedWorkspace builds a repository holding one prepared problem folder.
func solvedWorkspace(t *testing.T) string {
	t.Helper()
	if !vcs.Available() {
		t.Skip("git not on PATH")
	}
	root := t.TempDir()
	if _, err := vcs.Init(context.Background(), root); err != nil {
		t.Fatalf("init: %v", err)
	}
	for _, kv := range [][2]string{
		{"user.email", "test@example.com"},
		{"user.name", "leetui test"},
		{"commit.gpgsign", "false"},
	} {
		cmd := exec.Command("git", "config", kv[0], kv[1])
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git config: %v: %s", err, out)
		}
	}

	dir := filepath.Join(root, workspace.FolderName(1, "two-sum"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for name, body := range map[string]string{
		workspace.ReadmeFile:    "# 1. Two Sum\n",
		workspace.NotesFile:     "# notes\n",
		workspace.TestcasesFile: "[2,7]\n9\noutput:\n[0,1]\n",
		"solution.py":           "class Solution: ...\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return root
}

func solved() Solved {
	return Solved{
		ID: 1, Slug: "two-sum", Title: "Two Sum",
		Lang: "Python3", Filename: "solution.py",
		Runtime: "58 ms", Percentile: 91.23,
	}
}

func TestCommitWritesTheConventionalSubject(t *testing.T) {
	root := solvedWorkspace(t)
	git := config.Git{CommitOnAccepted: true, CommitNotes: true}

	res, err := Commit(context.Background(), root, git, solved())
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if res.Files != 4 {
		t.Errorf("committed %d files, want 4 (solution, README, testcases, notes)", res.Files)
	}

	subject := gitOut(t, root, "log", "-1", "--format=%s")
	want := "solve(0001): two sum — python3, 58ms, beats 91.2%"
	if strings.TrimSpace(subject) != want {
		t.Errorf("subject =\n  %q\nwant\n  %q", strings.TrimSpace(subject), want)
	}
}

// TestCommitNotesIsRespected covers the opt-out: notes.md is the user's own prose, and
// some people would rather not publish a half-finished thought.
func TestCommitNotesIsRespected(t *testing.T) {
	root := solvedWorkspace(t)
	git := config.Git{CommitOnAccepted: true, CommitNotes: false}

	if _, err := Commit(context.Background(), root, git, solved()); err != nil {
		t.Fatalf("commit: %v", err)
	}

	files := gitOut(t, root, "show", "--name-only", "--format=", "HEAD")
	if strings.Contains(files, workspace.NotesFile) {
		t.Errorf("notes.md was committed with commit_notes off:\n%s", files)
	}
	if !strings.Contains(files, "solution.py") {
		t.Errorf("the solution is missing from the commit:\n%s", files)
	}
}

// TestCommitSurvivesAMissingFile is the regression this nearly shipped with: `git add`
// fails the entire invocation on a pathspec that matches nothing, so one absent notes.md
// would have taken the commit down with it.
func TestCommitSurvivesAMissingFile(t *testing.T) {
	root := solvedWorkspace(t)
	dir := filepath.Join(root, workspace.FolderName(1, "two-sum"))
	if err := os.Remove(filepath.Join(dir, workspace.TestcasesFile)); err != nil {
		t.Fatalf("remove: %v", err)
	}

	res, err := Commit(context.Background(), root,
		config.Git{CommitOnAccepted: true, CommitNotes: true}, solved())
	if err != nil {
		t.Fatalf("commit with no testcases.txt: %v", err)
	}
	if res.Files != 3 {
		t.Errorf("committed %d files, want 3", res.Files)
	}
}

func TestCommitOnAPlainWorkspaceIsRecognisable(t *testing.T) {
	// The caller has to be able to tell "you have no repository" — which is normal and
	// silent — from a real failure.
	_, err := Commit(context.Background(), t.TempDir(), config.Git{}, solved())
	if !errors.Is(err, vcs.ErrNotRepo) {
		t.Fatalf("got %v, want ErrNotRepo", err)
	}
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
	}
	return string(out)
}
