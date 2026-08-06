package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Nano-AI/leetui/internal/config"
	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/syncer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestLiveBoard renders the real board against real LeetCode data: GraphQL -> SQLite ->
// FTS -> layout -> Glamour. It is the only test that exercises the whole stack at once.
//
// Opt-in:
//
//	LEETUI_LIVE=1 go test ./internal/tui -run TestLiveBoard -v
func TestLiveBoard(t *testing.T) {
	if os.Getenv("LEETUI_LIVE") != "1" {
		t.Skip("set LEETUI_LIVE=1 to run against the real API")
	}
	lipgloss.SetColorProfile(termenv.TrueColor)

	st, err := store.OpenPath(filepath.Join(t.TempDir(), "live.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	cl := leetcode.New(leetcode.WithRateLimit(3))
	sy := syncer.New(cl, st, 100)

	syncCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	ch := make(chan syncer.Progress, 32)
	go func() { _ = sy.Problems(syncCtx, ch, true) }()
	for p := range ch {
		if p.Done >= 200 {
			cancel()
		}
	}
	cancel()

	n, err := st.Count(context.Background())
	if err != nil || n == 0 {
		t.Fatalf("synced %d problems: %v", n, err)
	}

	m := New(config.Default(), st, cl, sy)
	var initMsgs []tea.Msg
	collect(m.Init(), &initMsgs)
	model := drive(t, m, append([]tea.Msg{tea.WindowSizeMsg{Width: 120, Height: 34}}, initMsgs...)...)

	if len(model.rows) == 0 {
		t.Fatal("board is empty after a live sync")
	}
	t.Logf("BOARD (%d problems synced)\n%s", n, model.View())

	// Move down a few rows, which triggers a real statement fetch and render. Network
	// round-trips need more than the unit-test deadline, so raise it for this stretch.
	old := cmdDeadline
	cmdDeadline = 15 * time.Second
	model = drive(t, model, key("j"), key("j"))
	cmdDeadline = old

	if model.detailMD == "" {
		t.Error("no statement rendered after moving the cursor")
	}
	t.Logf("AFTER MOVING (detail fetched live)\n%s", model.View())

	// Search over live data.
	model = drive(t, model, key("/"), key("t"), key("r"), key("e"), key("e"))
	t.Logf("SEARCH \"tree\" -> %d matches\n%s", len(model.rows), model.View())
	if len(model.rows) == 0 {
		t.Error(`searching "tree" against live data returned nothing`)
	}

	for i, line := range strings.Split(model.View(), "\n") {
		if w := lipgloss.Width(line); w > 120 {
			t.Errorf("line %d overflows with live data: %d cols", i, w)
		}
	}
}
