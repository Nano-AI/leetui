package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Painting a background across an already-styled string.
//
// A board row is assembled from cells that each carry their own foreground, and lipgloss
// ends every one with a full reset. A background set on the outer string therefore
// survives only as far as the first cell's reset — which is why wrapping a finished row
// in a Background style paints the first few cells and nothing else.
//
// So the background is re-asserted after every reset. That is what makes zebra banding
// possible on rows whose contents were styled independently.

// bgSequence returns the escape that sets a background, or "" when the terminal profile
// has no colour.
//
// Derived from lipgloss rather than written by hand, so it downgrades with the profile:
// under NO_COLOR or a plain TERM this comes back empty and Paint becomes a no-op, which
// is the correct behaviour rather than a special case.
func bgSequence(bg lipgloss.TerminalColor) string {
	const probe = "\x00"
	rendered := lipgloss.NewStyle().Background(bg).Render(probe)
	if i := strings.Index(rendered, probe); i > 0 {
		return rendered[:i]
	}
	return ""
}

// reset is what lipgloss closes every styled segment with.
const reset = "\x1b[0m"

// Paint lays a background under s, including the gaps between its styled segments.
//
// Returns s unchanged when the profile has no colour, so callers do not need to check.
func Paint(s string, bg lipgloss.TerminalColor) string {
	seq := bgSequence(bg)
	if seq == "" {
		return s
	}
	// Re-assert after each reset, so the band continues under the next cell.
	return seq + strings.ReplaceAll(s, reset, reset+seq) + reset
}
