package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// The debug log.
//
// leetui talks to an API nobody documents. When it misbehaves the user gets one error
// string and there is nothing to look at afterwards — which is exactly how a decode bug
// that failed EVERY submission took a screenshot to find, and one line of trace to
// explain once it was reproduced.
//
// So: a file, off by default, that records what was sent and what came back.
//
// THE INVARIANT IS REDACTION. D-002 says a session cookie never reaches a log, and the
// whole point of a debug log is that it gets pasted into bug reports. `Client.Debugf`
// receives traces with credentials already reduced to `auth.Redact` form; nothing here
// may widen that, and nothing here writes a header wholesale.

// DebugEnv turns the log on without a flag, for the subcommands.
//
// The TUI has `--debug`, but `leetui run` inside an editor has no convenient place to
// put one — the editor owns the command line. An environment variable is the seam that
// works from a shell profile, a Makefile, or a single prefixed invocation.
const DebugEnv = "LEETUI_DEBUG"

// Logger writes timestamped lines to a file, or discards them.
type Logger struct {
	mu  sync.Mutex
	w   io.WriteCloser
	off bool
}

// Discard is a Logger that writes nothing. Returned when debugging is off, so callers
// never branch on nil.
func Discard() *Logger { return &Logger{off: true} }

// OpenLog starts a debug log, returning where it went.
//
// Appends rather than truncates: the interesting run is often the one BEFORE the one you
// are looking at, and losing it to a restart is how you end up reproducing a bug twice.
func OpenLog() (*Logger, string, error) {
	dir, err := DataDir()
	if err != nil {
		return Discard(), "", err
	}
	path := filepath.Join(dir, "debug.log")

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return Discard(), path, fmt.Errorf("open debug log: %w", err)
	}

	l := &Logger{w: f}
	l.Printf("--- leetui started ---")
	return l, path, nil
}

// DebugRequested reports whether the environment asks for a log.
func DebugRequested() bool { return os.Getenv(DebugEnv) != "" }

// Printf writes one timestamped line.
//
// Safe from any goroutine: traces arrive from tea.Cmd goroutines and the sync worker at
// the same time, and interleaved half-lines would make the log worse than none.
func (l *Logger) Printf(format string, args ...any) {
	if l == nil || l.off {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.w == nil {
		return
	}
	fmt.Fprintf(l.w, "%s %s\n",
		time.Now().Format("15:04:05.000"), fmt.Sprintf(format, args...))
}

// Close flushes and closes the file.
func (l *Logger) Close() error {
	if l == nil || l.off || l.w == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	err := l.w.Close()
	l.w = nil
	return err
}
