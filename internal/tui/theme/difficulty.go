package theme

import "github.com/charmbracelet/lipgloss"

// ---------------------------------------------------------------------------
// Difficulty
// ---------------------------------------------------------------------------

// Difficulty is a problem's rating. Because green and red are reserved for the judge,
// difficulty is encoded by weight and bracket mass rather than hue.
type Difficulty int

const (
	Easy Difficulty = iota
	Medium
	Hard
)

// Render returns the three-letter difficulty tag on the dim -> bone -> amber weight ramp.
func (d Difficulty) Render() string {
	switch d {
	case Hard:
		return lipgloss.NewStyle().Foreground(Amber).Bold(true).Render("HRD")
	case Medium:
		return lipgloss.NewStyle().Foreground(Bone).Render("MED")
	default:
		return lipgloss.NewStyle().Foreground(Dim).Render("ESY")
	}
}

// ID renders a problem's number in the board's leftmost cell.
//
// It is a plain zero-padded figure, like a flight number on a departure board: a rigid
// four-digit field that keeps the left margin dead straight.
//
// An earlier version wrapped the number in half-block brackets whose mass encoded
// difficulty. That was wrong twice over — the D column already states difficulty, so it
// was the same fact twice, and the three different bracket widths made the left edge
// ragged and noisy. Difficulty belongs to Difficulty.Render() alone.
//
// The selected row is marked by a single amber bar in the cell's first column. It costs
// no width, disturbs no alignment, and is the row's only selection indicator.
func ID(id int, selected bool) string {
	num := pad4(id)
	if selected {
		return lipgloss.NewStyle().Foreground(Amber).Bold(true).Render("▌ " + num)
	}
	return lipgloss.NewStyle().Foreground(Dim).Render("  " + num)
}

// IDWidth is the width ID always renders to.
const IDWidth = 6

func pad4(n int) string {
	s := itoa(n)
	for len(s) < 4 {
		s = "0" + s
	}
	return s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}
