package tui

import (
	"fmt"
	"strings"

	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
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

	// An active company pack is announced here as well as on the board bezel: it changes
	// what every row means, and the rail is where the app's current state lives.
	if m.pack.Active() {
		right = append(right, theme.Label.Render("◈ "+m.pack.Name))
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

	rightStr := strings.Join(right, theme.Rule.Render(sep1()))

	f := components.Frame{Width: m.width, Height: railHeight}
	gap := f.InnerWidth() - lipgloss.Width(mark) - lipgloss.Width(rightStr) - 2
	if gap < 1 {
		gap = 1
	}
	return f.Render(" " + mark + strings.Repeat(" ", gap) + rightStr + " ")
}
