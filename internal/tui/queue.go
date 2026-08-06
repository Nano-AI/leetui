package tui

import (
	"fmt"
	"strings"

	"github.com/grootbeat/leetui/internal/tui/components"
	"github.com/grootbeat/leetui/internal/tui/theme"
)

// ---------------------------------------------------------------------------
// Queue
// ---------------------------------------------------------------------------

func (m Model) viewQueue(w, h int) string {
	f := components.Frame{
		Title:   "submissions",
		Width:   w,
		Height:  h,
		Focused: m.focus == paneQueue,
	}

	if len(m.queue) == 0 {
		return f.Render(" " + theme.Meta.Render("Nothing submitted yet."))
	}

	var b strings.Builder
	for i, s := range m.queue {
		if i >= f.InnerHeight() {
			break
		}
		verdictW := maxInt(f.InnerWidth()-18, 8)
		b.WriteString(components.Row([]string{
			cell(theme.Meta.Render(fmt.Sprintf("%04d", s.ProblemID)), 4),
			cell(theme.Meta.Render(truncate(s.Lang, 7)), 7),
			cell(s.flap.View(verdictW), verdictW),
		}))
		b.WriteString("\n")
	}
	return f.Render(b.String())
}
