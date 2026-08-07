package vcs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommitReturnsTheShortHashAndFileCount(t *testing.T) {
	repo := testRepo(t)
	a := write(t, repo, "0001-two-sum/solution.go", "package main\n")
	b := write(t, repo, "0001-two-sum/README.md", "# 1. Two Sum\n")

	res, err := repo.Commit(context.Background(), CommitRequest{
		Paths:   []string{a, b},
		Message: "solve(0001): two sum — go",
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if res.Files != 2 {
		t.Errorf("Files = %d, want 2", res.Files)
	}
	if res.Short == "" {
		t.Error("no short hash returned")
	}
}

func TestCommittingTheSameFileTwiceIsNotAFailure(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()
	path := write(t, repo, "solution.py", "print(1)\n")
	req := CommitRequest{Paths: []string{path}, Message: "solve(0001): two sum"}

	if _, err := repo.Commit(ctx, req); err != nil {
		t.Fatalf("first commit: %v", err)
	}

	// The second accepted verdict on an unedited solution reaches this every time. It
	// has to read as "already committed", not as an error the user must act on.
	_, err := repo.Commit(ctx, req)
	if !errors.Is(err, ErrNothingToCommit) {
		t.Fatalf("second commit: got %v, want ErrNothingToCommit", err)
	}
}

func TestCommitRefusesTheWrongBranch(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()
	path := write(t, repo, "solution.py", "print(1)\n")

	_, err := repo.Commit(ctx, CommitRequest{
		Paths:   []string{path},
		Message: "solve(0001): two sum",
		Branch:  "definitely-not-the-branch",
	})
	if !errors.Is(err, ErrWrongBranch) {
		t.Fatalf("got %v, want ErrWrongBranch", err)
	}

	// The guard runs before anything is staged. A refusal must leave the index exactly
	// as it was — the user may have had their own work staged in another pane.
	st, _ := repo.Status(ctx)
	for _, c := range st.Changes {
		if c.Staged {
			t.Errorf("%s was staged despite the refusal", c.Path)
		}
	}
}

func TestCommitTouchesOnlyTheRequestedPaths(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	mine := write(t, repo, "0001-two-sum/solution.py", "print(1)\n")
	theirs := write(t, repo, "scratch.txt", "unrelated work\n")

	// Stage the unrelated file the way a user would in another pane.
	if _, err := run(ctx, repo.Root, "add", "--", "scratch.txt"); err != nil {
		t.Fatalf("stage: %v", err)
	}

	if _, err := repo.Commit(ctx, CommitRequest{
		Paths: []string{mine}, Message: "solve(0001): two sum",
	}); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Auto-commit must never sweep up work it was not asked about.
	out, err := run(ctx, repo.Root, "show", "--name-only", "--format=", "HEAD")
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if strings.Contains(out, "scratch.txt") {
		t.Errorf("the commit swept up an unrelated staged file:\n%s", out)
	}
	if !strings.Contains(out, "solution.py") {
		t.Errorf("the requested file is missing from the commit:\n%s", out)
	}
	_ = theirs
}

func TestCommitRefusesPathsOutsideTheRepository(t *testing.T) {
	repo := testRepo(t)
	outside := filepath.Join(t.TempDir(), "elsewhere.txt")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := repo.Commit(context.Background(), CommitRequest{
		Paths: []string{outside}, Message: "solve(0001): two sum",
	})
	if err == nil || !strings.Contains(err.Error(), "outside the repository") {
		t.Fatalf("got %v, want a refusal naming the repository boundary", err)
	}
}

func TestCommitRejectsAnEmptyMessage(t *testing.T) {
	repo := testRepo(t)
	path := write(t, repo, "solution.py", "print(1)\n")
	_, err := repo.Commit(context.Background(), CommitRequest{
		Paths: []string{path}, Message: "   ",
	})
	if err == nil {
		t.Fatal("an empty commit message was accepted")
	}
}

// TestRelativeSurvivesASymlinkedRoot covers the macOS case directly: t.TempDir() hands
// back /var/…, git reports /private/var/…, and comparing them textually declares every
// path in the repository to be outside it.
func TestRelativeSurvivesASymlinkedRoot(t *testing.T) {
	repo := testRepo(t)
	path := write(t, repo, "0001-two-sum/solution.py", "print(1)\n")

	// Deliberately the unresolved spelling, which is what the workspace package hands
	// over when the user's config points through a symlink.
	rel, err := repo.relative([]string{path})
	if err != nil {
		t.Fatalf("relative: %v", err)
	}
	if len(rel) != 1 || rel[0] != "0001-two-sum/solution.py" {
		t.Fatalf("relative = %v, want [0001-two-sum/solution.py]", rel)
	}
}
