package vcs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Running git.
//
// Every invocation goes through here so the environment is set the same way each time,
// and so a failure carries git's own stderr rather than "exit status 1".

// gitEnv is the environment every git call runs with.
//
// GIT_TERMINAL_PROMPT=0 is the important one. leetui draws on the alternate screen; if
// git decides to ask for a username on stdin the prompt is invisible and the app hangs
// with no way to answer. Failing with "could not read Username" is recoverable — the
// user can push from their own shell — and a hung TUI is not.
//
// GIT_OPTIONAL_LOCKS=0 keeps a status refresh from taking the index lock, which would
// collide with an editor running gitsigns or fugitive in the next pane.
func gitEnv() []string {
	return append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_OPTIONAL_LOCKS=0",
	)
}

// run executes git in dir and returns stdout.
//
// A non-zero exit becomes an error carrying stderr, trimmed. git's messages are written
// for humans and are almost always better than a paraphrase.
func run(ctx context.Context, dir string, args ...string) (string, error) {
	stdout, _, err := runCode(ctx, dir, args...)
	return stdout, err
}

// runCode is run, plus the exit code, for the commands where a non-zero exit is an
// answer rather than a failure — `diff --quiet` reports "there are changes" that way.
func runCode(ctx context.Context, dir string, args ...string) (string, int, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = gitEnv()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		return stdout.String(), 0, nil
	}

	var exit *exec.ExitError
	code := -1
	if errors.As(err, &exit) {
		code = exit.ExitCode()
	}

	msg := strings.TrimSpace(stderr.String())
	if msg == "" {
		msg = err.Error()
	}
	return stdout.String(), code, fmt.Errorf("git %s: %s", args[0], firstLine(msg))
}

// firstLine keeps an error to one line.
//
// git is fond of multi-paragraph advice ("hint: Updates were rejected because…"), which
// is good in a terminal and wrong in a single-line status bar. The detail is still
// available by running the command by hand, which the hint itself tells you to do.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}
