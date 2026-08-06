package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/grootbeat/leetui/internal/tui/components"
	"github.com/grootbeat/leetui/internal/tui/theme"
)

// ---------------------------------------------------------------------------
// Rail
// ---------------------------------------------------------------------------

// viewRail is the instrument strip: wordmark, sync state, timer, account.
//
// It carries a bezel like every other pane. A departure board's header is part of the
// same housing as the board, not a caption floating above it.
func (m Model) viewRail() string {
	mark := lipgloss.NewStyle().Foreground(theme.Amber).Bold(true).Render(theme.Display("leetui"))

	var right []string
	if m.syncing {
		p := m.syncProgress
		note := p.Note
		if note == "" {
			note = fmt.Sprintf("%d/%d", p.Done, p.Total)
		}
		right = append(right, theme.Label.Render("⟳ "+note))
	}

	timer := theme.Meta.Render("⏱ --:--:--")
	if m.timerRunning || m.elapsed > 0 {
		timer = theme.Label.Render("⏱ " + formatDuration(m.elapsed))
	}
	right = append(right, timer)

	switch {
	case m.username == "":
		right = append(right, theme.Meta.Render("signed out"))
	case m.premium:
		right = append(right, theme.Label.Render("◆ premium"), theme.Meta.Render(m.username))
	default:
		right = append(right, theme.Meta.Render("free"), theme.Meta.Render(m.username))
	}

	rightStr := strings.Join(right, theme.Rule.Render(" ┊ "))

	f := components.Frame{Width: m.width, Height: railHeight}
	gap := f.InnerWidth() - lipgloss.Width(mark) - lipgloss.Width(rightStr) - 2
	if gap < 1 {
		gap = 1
	}
	return f.Render(" " + mark + strings.Repeat(" ", gap) + rightStr + " ")
}
