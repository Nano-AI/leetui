package tui

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/config"
	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/syncer"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// newTestModel builds a model backed by a real store seeded with sample problems, so
// the render path exercises the same code as production.
func newTestModel(t *testing.T, seeded bool) Model {
	t.Helper()
	lipgloss.SetColorProfile(termenv.TrueColor)

	// Point config at a temp directory for the whole test.
	//
	// Not optional: anything reaching Config.Save — the editor picker, remembering a
	// language — writes to the developer's REAL config.toml otherwise. That happened,
	// and it persisted a t.TempDir() workspace into a live install.
	t.Setenv(config.DirEnv, t.TempDir())

	st, err := store.OpenPath(filepath.Join(t.TempDir(), "tui.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	if seeded {
		if err := st.UpsertSummaries(context.Background(), seedProblems()); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	cfg := config.Default()

	// And point the workspace somewhere disposable. The default is ~/leetcode, which on
	// a developer's machine is their REAL workspace and quite possibly a real git
	// repository — the repository view would read it, and its contents would decide
	// what the test sees.
	cfg.Workspace = t.TempDir()

	// Offline transport: the detail loader would otherwise fetch statements from
	// leetcode.com on every cursor move. Unit tests must not touch the network.
	cl := leetcode.New(leetcode.WithHTTPClient(&http.Client{Transport: offlineTransport{}}))
	return New(cfg, st, cl, syncer.New(cl, st, 100))
}

type offlineTransport struct{}

func (offlineTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("network disabled in tests")
}

func seedProblems() []leetcode.ProblemSummary {
	return []leetcode.ProblemSummary{
		{FrontendID: "1", Title: "Two Sum", Slug: "two-sum", Difficulty: leetcode.Easy,
			AcRate: 57.9, Status: leetcode.StatusAccepted,
			Tags: []leetcode.Tag{{Name: "Array", Slug: "array"}}},
		{FrontendID: "42", Title: "Trapping Rain Water", Slug: "trapping-rain-water",
			Difficulty: leetcode.Hard, AcRate: 61.2,
			Tags: []leetcode.Tag{{Name: "Two Pointers", Slug: "two-pointers"}}},
		{FrontendID: "146", Title: "LRU Cache", Slug: "lru-cache", Difficulty: leetcode.Medium,
			AcRate: 42.7, Status: leetcode.StatusAttempted,
			Tags: []leetcode.Tag{{Name: "Design", Slug: "design"}}},
		{FrontendID: "1650", Title: "Lowest Common Ancestor of a Binary Tree III",
			Slug: "lowest-common-ancestor-of-a-binary-tree-iii", Difficulty: leetcode.Medium,
			AcRate: 80.1, PaidOnly: true,
			Tags: []leetcode.Tag{{Name: "Tree", Slug: "tree"}}},
	}
}

// cmdDeadline is how long a command gets to produce a message before the harness moves
// on. Timer-based commands (the one-second clock tick, the status-line expiry) never
// resolve inside it, which is exactly the intent — a test must not wait on wall time.
var cmdDeadline = 120 * time.Millisecond

// collect runs a command and gathers the messages it produces, recursing into batches.
func collect(cmd tea.Cmd, out *[]tea.Msg) {
	if cmd == nil {
		return
	}
	done := make(chan tea.Msg, 1)
	// Deliberately unsupervised: a timer command blocks for its full duration, and the
	// goroutine is reaped when the test binary exits.
	go func() { done <- cmd() }()

	select {
	case msg := <-done:
		switch m := msg.(type) {
		case nil:
		case tea.BatchMsg:
			for _, c := range m {
				collect(c, out)
			}
		default:
			*out = append(*out, msg)
		}
	case <-time.After(cmdDeadline):
		// Timer-based command; nothing to deliver.
	}
}

// drive sends messages to a model and settles the resulting commands, so store-backed
// reads land before the frame is rendered.
//
// It runs a bounded number of rounds: a message can produce a command that produces
// another message (rows -> detail), and without a bound a self-rescheduling command
// would spin forever.
func drive(t *testing.T, m Model, msgs ...tea.Msg) Model {
	t.Helper()

	var model tea.Model = m
	pending := append([]tea.Msg{}, msgs...)

	for round := 0; round < 6 && len(pending) > 0; round++ {
		var produced []tea.Msg
		for _, msg := range pending {
			var cmd tea.Cmd
			model, cmd = model.Update(msg)
			collect(cmd, &produced)
		}
		pending = produced
	}
	return model.(Model)
}

// boot builds a model, runs Init (which is what loads the board from the store), and
// settles the resulting commands — the same sequence Bubbletea performs at startup.
func boot(t *testing.T, seeded bool, w, h int) Model {
	t.Helper()
	m := newTestModel(t, seeded)

	var initMsgs []tea.Msg
	collect(m.Init(), &initMsgs)

	msgs := append([]tea.Msg{tea.WindowSizeMsg{Width: w, Height: h}}, initMsgs...)
	return drive(t, m, msgs...)
}

func key(s string) tea.KeyMsg {
	switch s {
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEscape}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}
