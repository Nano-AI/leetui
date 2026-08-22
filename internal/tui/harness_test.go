package tui

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"sync"
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
//
// It is a filter, not a budget, and it cannot be both. Widening it past the 700ms watch
// tick would make drive wait on the very commands it exists to skip, so anything slower
// than this is unreachable by widening. A command that shells out — a local run spawns
// python, a push spawns git — is real work with no ceiling the harness can name, and
// belongs to driveAwaiting instead.
var cmdDeadline = 120 * time.Millisecond

// driveRounds bounds how many times a round of messages may produce another round.
const driveRounds = 6

// awaitBudget bounds driveAwaiting when the message it was told to expect never comes.
//
// Generous on purpose. It is not an estimate of how long a subprocess takes — nothing
// pays it unless the test is already failing, and when that happens a few seconds buys a
// diagnosis instead of a guess.
const awaitBudget = 10 * time.Second

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

	for round := 0; round < driveRounds && len(pending) > 0; round++ {
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

// driveAwaiting settles a model like drive, but waits for the message the test is about
// rather than for a slice of wall time.
//
// Use it wherever an assertion depends on a command that shells out. drive gives every
// command cmdDeadline, and that same 120ms is then all a python or git subprocess gets:
// about 60ms of real work on a developer's machine, which held right up until a macOS CI
// runner made it slower and two after-edit tests began reporting that the feature had not
// fired. The failure is silent by construction — collect drops the message it did not
// wait long enough for, and the model simply looks as though nothing happened.
//
// Naming the message removes the guess. Timer commands are still never waited on: the
// round ends the moment a T lands, so the 700ms tick beside it is abandoned exactly as
// drive would have abandoned it.
func driveAwaiting[T tea.Msg](t *testing.T, m Model, msgs ...tea.Msg) Model {
	t.Helper()

	var model tea.Model = m
	pending := append([]tea.Msg{}, msgs...)
	var delivered bool

	for round := 0; round < driveRounds && len(pending) > 0; round++ {
		cmds := make([]tea.Cmd, 0, len(pending))
		for _, msg := range pending {
			if _, ok := msg.(T); ok {
				// Delivered, not merely received: the assertions are about what Update
				// did with it, so the wait is not over until it has been through Update.
				delivered = true
			}
			var cmd tea.Cmd
			model, cmd = model.Update(msg)
			cmds = append(cmds, cmd)
		}
		if delivered {
			// Back to drive's terms. Holding the wider budget open would spend it on
			// the timer commands that outlive the message we came for.
			pending = gather(cmds, cmdDeadline, nil)
			continue
		}
		pending = gather(cmds, awaitBudget, isMsg[T])
	}

	if !delivered {
		t.Fatalf("no %T arrived within %s: the command that produces it either never ran "+
			"or never finished", *new(T), awaitBudget)
	}
	return model.(Model)
}

// isMsg reports whether msg is a T, as a predicate gather can hold.
func isMsg[T tea.Msg](msg tea.Msg) bool {
	_, ok := msg.(T)
	return ok
}

// gather runs commands concurrently and returns the messages they produce, recursing
// into batches. It returns as soon as every command has answered, or stop accepts a
// message, or budget expires. A nil stop waits for the first two.
//
// Concurrent where collect is sequential, and the difference is the point: a round mixes
// the command under test with timer commands that never answer at all. Run one at a time,
// each timer costs the full budget and the round costs their sum — which is affordable at
// 120ms and absurd at ten seconds. Run together, the round costs whichever comes first.
//
// Abandoned commands are left running, as collect leaves them: a timer holds its
// goroutine until it fires, each has somewhere to put its message, and the test binary
// reaps them all on exit.
func gather(cmds []tea.Cmd, budget time.Duration, stop func(tea.Msg) bool) []tea.Msg {
	var (
		mu   sync.Mutex
		msgs []tea.Msg
		wg   sync.WaitGroup
		once sync.Once
	)
	done := make(chan struct{})
	finish := func() { once.Do(func() { close(done) }) }

	var run func(tea.Cmd)
	run = func(cmd tea.Cmd) {
		if cmd == nil {
			return
		}
		// Counted before the parent's Done runs, so the group cannot reach zero while a
		// batch is still handing out its children.
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch msg := cmd().(type) {
			case nil:
			case tea.BatchMsg:
				for _, c := range msg {
					run(c)
				}
			default:
				mu.Lock()
				msgs = append(msgs, msg)
				mu.Unlock()
				if stop != nil && stop(msg) {
					finish()
				}
			}
		}()
	}
	for _, cmd := range cmds {
		run(cmd)
	}
	go func() { wg.Wait(); finish() }()

	select {
	case <-done:
	case <-time.After(budget):
	}

	// Copied under the lock: the commands we walked away from are still holding a
	// reference to msgs and may append to it after this returns.
	mu.Lock()
	defer mu.Unlock()
	return append([]tea.Msg{}, msgs...)
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
