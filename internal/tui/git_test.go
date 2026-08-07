package tui

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/vcs"
)

// gitWorkspace boots a model whose workspace is a real repository.
//
// A real one rather than a fake: the pane's whole job is reporting what git says, and a
// stub would only ever confirm that the stub agrees with itself.
func gitWorkspace(t *testing.T, commit bool) Model {
	t.Helper()
	if !vcs.Available() {
		t.Skip("git not on PATH")
	}

	m := boot(t, true, 120, 32)
	root := m.cfg.Workspace

	repo, err := vcs.Init(context.Background(), root)
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
			t.Fatalf("git config: %v: %s", err, out)
		}
	}

	dir := filepath.Join(root, "0001-two-sum")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "solution.py"), []byte("print(1)\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if commit {
		if _, err := repo.Commit(context.Background(), vcs.CommitRequest{
			Paths: []string{filepath.Join(dir, "solution.py")}, Message: "solve(0001): two sum",
		}); err != nil {
			t.Fatalf("commit: %v", err)
		}
	}
	return m
}

func TestGitPaneNamesTheBranchAndTheUncommittedFile(t *testing.T) {
	m := drive(t, gitWorkspace(t, false), key("v"))

	if m.mode != modeGit {
		t.Fatalf("v did not open the repository view (mode %v)", m.mode)
	}
	out := stripANSI(m.View())

	// The folder, not the file: git collapses a wholly untracked directory into one
	// entry, and for a workspace of problem folders that is the readable form — a user
	// with fifty unsaved problems wants fifty lines, not two hundred.
	if !strings.Contains(out, "0001-two-sum/") {
		t.Errorf("the uncommitted problem folder is not listed:\n%s", out)
	}
	// "new", not git's "??" — the whole reason this pane exists is that porcelain codes
	// mean nothing to someone using a LeetCode client.
	if !strings.Contains(out, "new") {
		t.Errorf("the change is not described in words:\n%s", out)
	}
	if !strings.Contains(out, "no upstream") {
		t.Errorf("a branch with no upstream should say so:\n%s", out)
	}
}

func TestGitPaneExplainsAWorkspaceWithNoRepository(t *testing.T) {
	m := drive(t, boot(t, true, 120, 32), key("v"))
	out := stripANSI(m.View())

	// Not a fault, and not a dead end: the exact command is on screen. leetui does not
	// offer to run `git init` itself — a stray .git in someone's home directory is a
	// real annoyance.
	if !strings.Contains(out, "not a git repository yet") {
		t.Errorf("no explanation for a plain workspace:\n%s", out)
	}
	if !strings.Contains(out, "git init") {
		t.Errorf("the fix is not shown:\n%s", out)
	}
}

// TestPushRefusalSaysWhy covers the failure mode a silent keybinding creates: p doing
// nothing is indistinguishable from p being broken.
func TestPushRefusalSaysWhy(t *testing.T) {
	m := drive(t, gitWorkspace(t, true), key("v"))
	m = drive(t, m, key("p"))

	if m.git.confirming {
		t.Fatal("push was armed with no remote configured")
	}
	if !strings.Contains(m.status, "remote") {
		t.Errorf("status = %q, want an explanation mentioning the remote", m.status)
	}
}

// TestPushAsksFirst is the guard on the one outward-facing action in the app.
func TestPushAsksFirst(t *testing.T) {
	m := gitWorkspace(t, true)

	// Give the branch somewhere to go: a second repository on disk, so nothing in this
	// test can reach the network.
	remote := t.TempDir()
	gitRun(t, remote, "init", "--bare")
	gitRun(t, m.cfg.Workspace, "remote", "add", "origin", remote)

	m = drive(t, m, key("v"))
	if !m.canPush() {
		t.Fatalf("expected a pushable state; status %+v remote %q", m.git.status, m.git.remote)
	}

	m = drive(t, m, key("p"))
	if !m.git.confirming {
		t.Fatal("p pushed without asking")
	}
	// The confirmation must name the destination — the URL, not "origin".
	if out := stripANSI(m.View()); !strings.Contains(out, remote) {
		t.Errorf("the confirmation does not name where this is going:\n%s", out)
	}

	// Anything that is not an explicit yes cancels.
	if cancelled := drive(t, m, key("n")); cancelled.git.confirming {
		t.Error("a key other than y left the push armed")
	}
}

func TestPushConfirmedReachesTheRemote(t *testing.T) {
	m := gitWorkspace(t, true)
	remote := t.TempDir()
	gitRun(t, remote, "init", "--bare")
	gitRun(t, m.cfg.Workspace, "remote", "add", "origin", remote)

	m = drive(t, m, key("v"))
	m = drive(t, m, key("p"))
	m = drive(t, m, key("y"))

	// Ask the bare repository directly rather than trusting the status line.
	out := gitRun(t, remote, "log", "--oneline")
	if !strings.Contains(out, "two sum") {
		t.Errorf("the commit did not reach the remote:\n%s", out)
	}
	if m.git.confirming {
		t.Error("the confirmation is still armed after pushing")
	}
}

func gitRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

var _ tea.Msg = gitLoadedMsg{}
