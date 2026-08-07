package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/Nano-AI/leetui/internal/config"
	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// asciiMode runs a test with the ASCII glyph set, restoring it afterwards. The flag is
// package-level (set once at startup in production), so a test that changes it must put
// it back or it leaks into everything after.
func asciiMode(t *testing.T) {
	t.Helper()
	prev := theme.ASCII
	theme.ASCII = true
	t.Cleanup(func() { theme.ASCII = prev })
}

func TestStateGlyphs(t *testing.T) {
	m := boot(t, true, 120, 20)
	out := stripANSI(m.View())

	g := theme.Glyphs()
	if !strings.Contains(out, g.Solved) {
		t.Errorf("no solved mark on the board:\n%s", out)
	}
	if !strings.Contains(out, g.Tried) {
		t.Errorf("no tried mark on the board:\n%s", out)
	}
	// Words are gone; the header still names the column.
	for _, gone := range []string{"SOLVED", "TRIED", "LOCKED"} {
		if strings.Contains(out, gone) {
			t.Errorf("%q is still being spelled out", gone)
		}
	}
	if !strings.Contains(out, "STATE") {
		t.Error("the state column lost its header — the glyphs then have no legend")
	}
}

func TestAsciiFallbackSwapsEveryGlyph(t *testing.T) {
	asciiMode(t)

	m := boot(t, true, 120, 20)
	m = drive(t, m, key("m"))
	out := stripANSI(m.View())

	for _, unicodeGlyph := range []string{"✓", "◐", "⊘", "●"} {
		if strings.Contains(out, unicodeGlyph) {
			t.Errorf("ASCII mode still rendered %q", unicodeGlyph)
		}
	}
	if !strings.Contains(out, "x") || !strings.Contains(out, "*") {
		t.Errorf("ASCII marks are missing:\n%s", out)
	}
}

// TestGridSurvivesAmbiguousWidth is the reason the fallback exists. `✓` and `⊘` are
// East-Asian Ambiguous: a terminal may draw them two cells wide, and in a grid ruled to
// the column that shears every row below. Every row must measure the same either way.
func TestGridSurvivesAmbiguousWidth(t *testing.T) {
	for _, ascii := range []bool{false, true} {
		prev := theme.ASCII
		theme.ASCII = ascii
		m := boot(t, true, 120, 20)
		m = drive(t, m, key("m"))

		for i, line := range strings.Split(m.viewBoard(120, m.boardHeight()), "\n") {
			if line == "" {
				continue
			}
			if w := ansi.StringWidth(line); w != 120 {
				t.Errorf("ascii=%v: row %d is %d cells, want 120: %q",
					ascii, i, w, stripANSI(line))
			}
		}
		theme.ASCII = prev
	}
}

func TestCenterPutsTheGlyphInTheMiddle(t *testing.T) {
	for _, tc := range []struct {
		w    int
		want string
	}{
		{5, "  ✓"},
		{4, " ✓"},
		{1, "✓"},
	} {
		if got := theme.Center("✓", tc.w); got != tc.want {
			t.Errorf("Center(✓, %d) = %q, want %q", tc.w, got, tc.want)
		}
	}
	if got := theme.Center("", 5); got != "" {
		t.Errorf("Center of nothing produced %q", got)
	}
}

// TestPreferASCII covers the signals that actually predict a terminal cannot draw these.
func TestPreferASCII(t *testing.T) {
	cfg := config.Default()

	t.Run("config flag wins", func(t *testing.T) {
		t.Setenv("TERM", "xterm-256color")
		t.Setenv("LANG", "en_US.UTF-8")
		on := cfg
		on.UI.ASCII = true
		if !config.PreferASCII(on) {
			t.Error("ui.ascii = true was ignored")
		}
	})

	t.Run("utf-8 locale is trusted", func(t *testing.T) {
		t.Setenv("TERM", "xterm-256color")
		t.Setenv("LC_ALL", "")
		t.Setenv("LC_CTYPE", "")
		t.Setenv("LANG", "en_US.UTF-8")
		if config.PreferASCII(cfg) {
			t.Error("a UTF-8 locale was treated as unable to draw glyphs")
		}
	})

	t.Run("non utf-8 locale falls back", func(t *testing.T) {
		t.Setenv("TERM", "xterm-256color")
		t.Setenv("LC_ALL", "")
		t.Setenv("LC_CTYPE", "")
		t.Setenv("LANG", "C")
		if !config.PreferASCII(cfg) {
			t.Error("a non-UTF-8 locale should fall back")
		}
	})

	t.Run("kernel console falls back", func(t *testing.T) {
		t.Setenv("LANG", "en_US.UTF-8")
		t.Setenv("TERM", "linux")
		if !config.PreferASCII(cfg) {
			t.Error("TERM=linux should fall back; its font has 256 glyphs")
		}
	})
}
