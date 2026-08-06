package tui

import (
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
)

func TestSearchFiltersRows(t *testing.T) {
	m := boot(t, true, 120, 32)
	m = drive(t, m, key("/"), key("c"), key("a"), key("c"), key("h"))

	if m.filter.Text != "cach" {
		t.Fatalf("filter text = %q, want %q", m.filter.Text, "cach")
	}
	if len(m.rows) != 1 || m.rows[0].Slug != "lru-cache" {
		t.Fatalf("search returned %d rows, want just lru-cache", len(m.rows))
	}
}

func TestDifficultyToggle(t *testing.T) {
	m := boot(t, true, 120, 32)

	m = drive(t, m, key("3")) // Hard
	if len(m.rows) != 1 || m.rows[0].Difficulty != "Hard" {
		t.Fatalf("after toggling hard: %d rows", len(m.rows))
	}

	m = drive(t, m, key("2")) // Hard + Medium
	if len(m.rows) != 3 {
		t.Fatalf("after adding medium: %d rows, want 3", len(m.rows))
	}

	m = drive(t, m, key("3")) // Medium only
	if len(m.rows) != 2 {
		t.Fatalf("after removing hard: %d rows, want 2", len(m.rows))
	}

	m = drive(t, m, key("0")) // clear
	if len(m.rows) != 4 {
		t.Fatalf("after clearing: %d rows, want 4", len(m.rows))
	}
}

// TestCursorStaysVisible guards the scroll window: the rendered window and the scroll
// arithmetic must agree, or the cursor disappears off-screen and the app feels broken.
func TestCursorStaysVisible(t *testing.T) {
	m := boot(t, true, 120, 14)

	for i := 0; i < 10; i++ {
		m = drive(t, m, key("j"))
		visible := m.visibleRows()
		if m.cursor < m.scroll || m.cursor >= m.scroll+visible {
			t.Fatalf("cursor %d outside window [%d,%d) after %d moves",
				m.cursor, m.scroll, m.scroll+visible, i+1)
		}
	}
}

func TestEmptyStateInvitesAction(t *testing.T) {
	m := boot(t, false, 120, 32)
	out := m.View()
	if !strings.Contains(out, "Press S to sync") {
		t.Errorf("empty board does not say what to do next:\n%s", out)
	}
}

// TestAuthInputIsMasked is a security check: the pasted blob contains a session token,
// and it must never be legible on screen.
func TestAuthInputIsMasked(t *testing.T) {
	m := drive(t, boot(t, true, 120, 32), key("a"))

	secret := "supersecrettoken"
	for _, r := range secret {
		m = drive(t, m, key(string(r)))
	}
	if got := m.authInput.Value(); got != secret {
		t.Fatalf("input value = %q, want the typed text", got)
	}
	if strings.Contains(m.View(), secret) {
		t.Error("session token is rendered in plain text on screen")
	}
}

// TestNoColorStillDistinguishesVerdicts: under NO_COLOR the verdict colors vanish, so
// the letterspaced display treatment is the only thing left marking a verdict.
func TestNoColorStillDistinguishesVerdicts(t *testing.T) {
	if got := theme.Accepted.Render(); !strings.Contains(stripANSI(got), "A C C E P T E D") {
		t.Errorf("verdict lost its display treatment: %q", stripANSI(got))
	}
}

func TestFlipSettles(t *testing.T) {
	f := components.NewFlap(1, "PENDING", theme.Amber)

	cmd := f.FlipTo(theme.Display("accepted"), theme.AC)
	if cmd == nil {
		t.Fatal("FlipTo returned no command; the animation would never start")
	}

	for i := 0; i < 32 && f.Flipping(); i++ {
		msg := cmd().(components.FlipTickMsg)
		f, cmd = f.Update(msg)
		if cmd == nil {
			break
		}
	}
	if f.Flipping() {
		t.Fatal("flap never settled — a stuck flap reads as a hung submission")
	}
	if want := theme.Display("accepted"); f.Text() != want {
		t.Errorf("settled on %q, want %q", f.Text(), want)
	}
}

func TestReduceMotionSettlesInstantly(t *testing.T) {
	components.ReduceMotion = true
	t.Cleanup(func() { components.ReduceMotion = false })

	f := components.NewFlap(1, "PENDING", theme.Amber)
	if cmd := f.FlipTo(theme.Display("accepted"), theme.AC); cmd != nil {
		t.Error("FlipTo issued an animation command with ReduceMotion set")
	}
	if f.Flipping() {
		t.Error("flap is animating with ReduceMotion set")
	}
	if want := theme.Display("accepted"); f.Text() != want {
		t.Errorf("text is %q, want %q", f.Text(), want)
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			inEsc = true
		case inEsc && (r == 'm' || r == 'K'):
			inEsc = false
		case !inEsc:
			b.WriteRune(r)
		}
	}
	return b.String()
}
