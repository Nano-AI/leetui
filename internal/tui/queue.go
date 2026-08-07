package tui

import (
	"fmt"
	"strings"

	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
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
	rows := 0
	for _, s := range m.queue {
		if rows >= f.InnerHeight() {
			break
		}
		verdictW := maxInt(f.InnerWidth()-18, 8)
		b.WriteString(components.Row([]string{
			cell(theme.Meta.Render(fmt.Sprintf("%04d", s.ProblemID)), 4),
			cell(theme.Meta.Render(truncate(s.Lang, 7)), 7),
			cell(s.flap.View(verdictW), verdictW),
		}))
		b.WriteString("\n")
		rows++

		// The figures go on their own line under the verdict rather than beside it.
		// Beside it they would have to share width with a letterspaced verdict and get
		// truncated to nothing — and the flip is the moment, so nothing sits next to it.
		if st := s.stats(); st != "" && rows < f.InnerHeight() {
			b.WriteString("     " + theme.Meta.Render(truncate(st, maxInt(f.InnerWidth()-6, 4))))
			b.WriteString("\n")
			rows++
		}
	}
	return f.Render(b.String())
}
