package components

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// sparkBars is the eight-bucket vertical ramp.
var sparkBars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// AcceptanceSpark renders a problem's acceptance rate as a three-cell sparkline.
//
// This is a structural device carrying real data, not decoration: the three cells are
// fixed reference points at 25%, 50%, and 75% of the scale, so a glance across the board
// compares problems against each other rather than against nothing.
//
// It is deliberately dim — acceptance rate is metadata, and the row's amber belongs to
// the selection and the ID slot.
func AcceptanceSpark(ratePct float64) string {
	if ratePct < 0 {
		ratePct = 0
	}
	if ratePct > 100 {
		ratePct = 100
	}
	out := make([]rune, 3)
	for i, ref := range []float64{25, 50, 75} {
		// How far this problem's rate carries past each reference point.
		frac := (ratePct - ref + 25) / 50
		out[i] = bar(frac)
	}
	return lipgloss.NewStyle().Foreground(theme.Dim).Render(string(out))
}

func bar(frac float64) rune {
	switch {
	case frac <= 0:
		return sparkBars[0]
	case frac >= 1:
		return sparkBars[len(sparkBars)-1]
	default:
		return sparkBars[int(frac*float64(len(sparkBars)-1)+0.5)]
	}
}
