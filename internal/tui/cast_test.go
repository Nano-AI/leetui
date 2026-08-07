package tui

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/tui/components"
)

// Recording a demo without a terminal.
//
// asciinema's v2 format is a JSON header followed by one JSON array per write:
// [elapsed, "o", output]. Nothing about it requires a TTY — a recorder is just a program
// that timestamps what a process wrote. So the frames come from the real View(), driven
// through the real Update(), and the result plays back identically to a hand-recorded
// session.
//
// Which also means it cannot drift. A screenshot goes stale the first time a column
// changes; this is generated from the code that draws the column.

const (
	castW = 100
	castH = 30
)

// castFrame is one screen and how long it stays up.
type castFrame struct {
	view  string
	holdS float64
}

func TestRecordCast(t *testing.T) {
	if os.Getenv("LEETUI_CAST") == "" {
		t.Skip("set LEETUI_CAST=1 to record")
	}
	// Motion off: the flip is driven by tea.Tick, which the test harness does not run
	// on wall time. The frames below stage the flip explicitly instead, so the cast
	// shows the real sequence rather than a stalled one.
	components.ReduceMotion = true
	t.Cleanup(func() { components.ReduceMotion = false })

	m := boot(t, true, castW, castH)
	frames := []castFrame{{m.View(), 2.0}}

	// Search, one keystroke at a time — the whole point of the local FTS index is that
	// it keeps up, and a demo that jumps straight to the result does not show that.
	m = drive(t, m, key("/"))
	for _, r := range "cache" {
		m = drive(t, m, key(string(r)))
		frames = append(frames, castFrame{m.View(), 0.18})
	}
	frames[len(frames)-1].holdS = 1.4

	m = drive(t, m, key("enter"))
	frames = append(frames, castFrame{m.View(), 1.2})

	// Open it, reveal the tags, then the repository view.
	m = drive(t, m, key("enter"))
	frames = append(frames, castFrame{m.View(), 2.2})

	m = drive(t, m, key("z"))
	frames = append(frames, castFrame{m.View(), 1.8})

	m = drive(t, m, key("esc"))
	m = drive(t, m, key("V"))
	frames = append(frames, castFrame{m.View(), 2.0})

	m = drive(t, m, key("esc"))
	m = drive(t, m, key("?"))
	frames = append(frames, castFrame{m.View(), 2.5})

	if err := writeCast("/tmp/leetui.cast", frames); err != nil {
		t.Fatalf("write cast: %v", err)
	}
	t.Logf("wrote /tmp/leetui.cast — %d frames", len(frames))
}

// writeCast emits an asciinema v2 file.
func writeCast(path string, frames []castFrame) error {
	var b strings.Builder

	header := map[string]any{
		"version": 2,
		"width":   castW,
		"height":  castH,
		// Fixed, not time.Now(): a recording that differs byte-for-byte on every run
		// shows up as a diff in every commit that touches anything near it.
		"timestamp": 1767225600,
		"title":     "leetui",
		"env":       map[string]string{"TERM": "xterm-256color", "SHELL": "/bin/zsh"},
	}
	head, err := json.Marshal(header)
	if err != nil {
		return err
	}
	b.Write(head)
	b.WriteByte('\n')

	at := 0.0
	for _, f := range frames {
		// Home the cursor and clear, then paint. Writing each frame whole rather than
		// diffing keeps the file readable and costs nothing at this size.
		out := "\x1b[H\x1b[2J" + f.view

		line, err := json.Marshal([]any{at, "o", out})
		if err != nil {
			return err
		}
		b.Write(line)
		b.WriteByte('\n')
		at += f.holdS
	}

	// A trailing frame so the last screen is not cut off the instant it appears.
	last, err := json.Marshal([]any{at, "o", ""})
	if err != nil {
		return err
	}
	b.Write(last)
	b.WriteByte('\n')

	return os.WriteFile(path, []byte(b.String()), 0o644)
}
