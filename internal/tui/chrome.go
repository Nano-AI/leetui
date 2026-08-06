package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/grootbeat/leetui/internal/tui/theme"
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

// viewHints is the key strip. Hints are ordered most-useful first and dropped from the
// end when they do not fit — a truncated hint names a key without saying what it does.
func (m Model) viewHints() string {
	keys := []string{"/ search", "1·2·3 difficulty", "u unsolved", "S sync", "o open", "t timer", "? keys", "q quit"}
	if m.focus == paneDetail && len(m.detailImages) > 0 {
		keys[0] = fmt.Sprintf("1-%d images", minInt(len(m.detailImages), 9))
	}

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
