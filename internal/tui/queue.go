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

// verdictCell draws one submission's verdict, sweeping it through the palette while a
// celebration is running on that row (D-033).
//
// The flap is left to render itself in every other case, including while it is still
// flipping: the sweep only ever replaces a SETTLED Accepted, so the two animations never
// run over each other and the flip keeps its job of being the thing that resolves.
func (m Model) verdictCell(s queueItem, w int) string {
	if !m.celebrate.active() || m.celebrate.flapID != s.flap.ID || s.flap.Flipping() {
		return s.flap.View(w)
	}
	return rainbow(theme.Display(s.Verdict.Text()), m.celebrate.frame)
}

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
			cell(m.verdictCell(s, verdictW), verdictW),
		}))
		b.WriteString("\n")
		rows++

		// The badge sits on its own line above the figures, not beside the verdict: the
		// verdict is letterspaced and already spends the width, and D-021's rule holds
		// here too — the flip is the moment, so nothing shares a line with it.
		if badge := m.celebrationBadge(s.Tier); badge != "" && rows < f.InnerHeight() {
			b.WriteString("     " + badge + "\n")
			rows++
		}

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
