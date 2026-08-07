package vcs

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// These tests drive a real git binary against real repositories in t.TempDir(). Mocking
// git would test the mock: the whole reason D-011 shells out is that git's actual
// behaviour — hooks, config, default branch names — is what has to be inherited.

// testRepo returns an initialised repository with an identity of its own.
//
// The identity is set locally rather than relied upon from the machine, so the suite
// passes on a fresh checkout and on CI. Signing is disabled for the same reason: a
// developer with commit.gpgsign=true globally would otherwise get a prompt.
func testRepo(t *testing.T) Repo {
	t.Helper()
	if !Available() {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	repo, err := Init(context.Background(), dir)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	for _, kv := range [][2]string{
		{"user.email", "test@example.com"},
		{"user.name", "leetui test"},
		{"commit.gpgsign", "false"},
	} {
		cmd := exec.Command("git", "config", kv[0], kv[1])
		cmd.Dir = repo.Root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git config %s: %v: %s", kv[0], err, out)
		}
	}
	return repo
}

// write creates a file inside the repository and returns its path.
func write(t *testing.T, repo Repo, name, body string) string {
	t.Helper()
	path := filepath.Join(repo.Root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func TestOpenRejectsAPlainDirectory(t *testing.T) {
	if !Available() {
		t.Skip("git not on PATH")
	}
	// A directory the user has never run `git init` in is the state every new
	// workspace starts in. It must be distinguishable from a broken one.
	if _, err := Open(context.Background(), t.TempDir()); !errors.Is(err, ErrNotRepo) {
		t.Fatalf("Open on a plain directory: got %v, want ErrNotRepo", err)
	}
}

func TestOpenFindsTheRootFromASubdirectory(t *testing.T) {
	repo := testRepo(t)
	sub := filepath.Join(repo.Root, "0001-two-sum")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Opening from a problem folder must reach the workspace repository — `leetui run`
	// is routinely invoked from inside one.
	got, err := Open(context.Background(), sub)
	if err != nil {
		t.Fatalf("Open from subdirectory: %v", err)
	}
	if resolve(got.Root) != resolve(repo.Root) {
		t.Errorf("root = %q, want %q", got.Root, repo.Root)
	}
}

func TestStatusSeesUntrackedAndCommittedFiles(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	st, err := repo.Status(ctx)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !st.Clean() {
		t.Fatalf("a fresh repository reported %d changes", st.Dirty())
	}
	if st.Branch == "" {
		t.Error("no branch name on a fresh repository")
	}

	path := write(t, repo, "0001-two-sum/solution.py", "print(1)\n")

	st, err = repo.Status(ctx)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if st.Dirty() != 1 {
		t.Fatalf("after writing one file: %d changes, want 1", st.Dirty())
	}
	// Untracked, not modified. The pane says "new" for these, and a fresh workspace is
	// entirely untracked — calling that "modified" would be a lie.
	if c := st.Changes[0]; !c.Untracked {
		t.Errorf("change %+v: want Untracked", c)
	}

	if _, err := repo.Commit(ctx, CommitRequest{
		Paths: []string{path}, Message: "solve(0001): two sum",
	}); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if st, _ := repo.Status(ctx); !st.Clean() {
		t.Errorf("after committing, %d changes remain", st.Dirty())
	}
}

func TestUnpushedWithoutAnUpstream(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()
	write(t, repo, "a.txt", "x")
	if _, err := repo.Commit(ctx, CommitRequest{
		Paths: []string{"a.txt"}, Message: "add a",
	}); err != nil {
		t.Fatalf("commit: %v", err)
	}

	st, err := repo.Status(ctx)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	// Commits no remote tracks are the ones most in danger of dying with the machine,
	// so the pane must flag them even though ahead/behind is unavailable.
	if !st.Unpushed() {
		t.Error("a committed branch with no upstream should report unpushed work")
	}
	if err := repo.Push(ctx); !errors.Is(err, ErrNoUpstream) {
		t.Errorf("Push with no upstream: got %v, want ErrNoUpstream", err)
	}
}

func TestLogReadsBackTheSubject(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	if got, err := repo.Log(ctx, 5); err != nil || len(got) != 0 {
		t.Fatalf("log on an empty repository: %v, %v — want no commits, no error", got, err)
	}

	write(t, repo, "a.txt", "x")
	if _, err := repo.Commit(ctx, CommitRequest{
		Paths: []string{"a.txt"}, Message: "solve(0001): two sum — go, 3ms",
	}); err != nil {
		t.Fatalf("commit: %v", err)
	}

	log, err := repo.Log(ctx, 5)
	if err != nil {
		t.Fatalf("log: %v", err)
	}
	if len(log) != 1 {
		t.Fatalf("got %d commits, want 1", len(log))
	}
	if log[0].Subject != "solve(0001): two sum — go, 3ms" {
		t.Errorf("subject = %q", log[0].Subject)
	}
	if log[0].Short == "" || log[0].When == "" {
		t.Errorf("incomplete log entry %+v", log[0])
	}
}
