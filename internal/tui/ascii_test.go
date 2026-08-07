package tui

import (
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// TestASCIIModeDrawsNoBoxCharacters is the other half of D-023. Glyphs got an ASCII
// fallback and the frame did not, so a terminal that could not render ✓ still got
// ╭─┼─╯ — half a promise, and a bezel of question marks is worse than no bezel.
func TestASCIIModeDrawsNoBoxCharacters(t *testing.T) {
	t.Cleanup(func() { theme.ASCII = false })
	theme.ASCII = true

	m := boot(t, true, 120, 34)
	views := map[string]string{
		"board": m.View(),
		"solve": drive(t, m, key("enter")).View(),
		"help":  drive(t, m, key("?")).View(),
		"git":   drive(t, m, key("v")).View(),
	}

	// Every non-ASCII character the interface draws deliberately. If one of these
	// reaches the screen in ASCII mode, the fallback has a hole in it.
	banned := []string{
		"╭", "╮", "╰", "╯", "─", "│", "┬", "┴", "┼", "├", "┤",
		"╌", "┊", "▌", "✓", "◐", "⊘", "●",
	}
	for name, out := range views {
		for _, ch := range banned {
			if strings.Contains(out, ch) {
				t.Errorf("%s renders %q in ASCII mode", name, ch)
			}
		}
	}
}

// TestUnicodeModeStillDrawsTheBezel guards the other direction: the ASCII path must not
// leak into a terminal that can draw properly.
func TestUnicodeModeStillDrawsTheBezel(t *testing.T) {
	theme.ASCII = false
	out := boot(t, true, 120, 34).View()
	for _, ch := range []string{"╭", "╯", "─", "│"} {
		if !strings.Contains(out, ch) {
			t.Errorf("unicode mode is missing %q", ch)
		}
	}
}
