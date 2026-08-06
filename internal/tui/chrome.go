package tui

import (
	"fmt"
	"strings"

	"github.com/Nano-AI/leetui/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// Chrome
// ---------------------------------------------------------------------------

// cell sizes s to exactly n display cells, truncating rather than wrapping.
func cell(s string, n int) string {
	return lipgloss.NewStyle().Width(n).MaxWidth(n).Render(s)
}

func (m Model) viewStatus() string {
	if m.status == "" {
		return ""
	}
	style := theme.Meta
	if m.statusErr {
		// Errors are amber, not red: red belongs to the judge. An error is the system
		// speaking.
		style = theme.Label
	}
	return " " + style.Render(truncate(m.status, m.width-2))
}

// hintKeys is what the current screen can actually do.
//
// The two screens have different verbs, and offering "r run" while someone is scrolling a
// list of four thousand problems is noise that pushes the one key they need — enter — off
// the end of the strip.
func (m Model) hintKeys() []string {
	if m.mode != modeSolve {
		return []string{"enter open", "/ search", "c companies", "1·2·3 difficulty",
			"u unsolved", "p premium", "S sync", "? keys", "q quit"}
	}

	keys := []string{"e edit", "r run", "s submit", "d editorial", "l lang",
		"o open", "esc back", "? keys"}
	if imgs := m.paneImages(); len(imgs) > 0 {
		keys = append([]string{fmt.Sprintf("1-%d open", minInt(len(imgs), 9))}, keys...)
	}
	return keys
}

// viewHints is the key strip. Hints are ordered most-useful first and dropped from the
// end when they do not fit — a truncated hint names a key without saying what it does.
func (m Model) viewHints() string {
	keys := m.hintKeys()

	used := 1
	kept := keys[:0:0]
	for _, k := range keys {
		next := used + len(k)
		if len(kept) > 0 {
			next += 3
		}
		if next > m.width {
			break
		}
		kept = append(kept, k)
		used = next
	}
	return theme.Utility.Render(" " + strings.Join(kept, theme.Rule.Render(" ┊ ")))
}
