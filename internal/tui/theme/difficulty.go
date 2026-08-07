package theme

import "github.com/charmbracelet/lipgloss"

// ---------------------------------------------------------------------------
// Difficulty
// ---------------------------------------------------------------------------

// Difficulty is a problem's rating.
//
// These are LEETCODE'S OWN COLOURS, read off leetcode.com's dark theme on 2026-08-06:
//
//	--difficulty-easy    --dark-teal-60     #1CBABA
//	--difficulty-medium  --dark-yellow-60   #FFB700
//	--difficulty-hard    --dark-red-60      #F63737
//
// This is a deliberate exception to the rule that green and red belong to the judge
// alone (DESIGN.md, D-017). The user asked for it, and the borrowed palette is worth the
// dilution: everyone who has used LeetCode already reads teal/amber/red as easy/medium/
// hard without a legend, which no invented ramp achieves.
//
// The dilution is real and is contained deliberately. Difficulty appears ONLY in a
// three-character column and in the detail heading — never near the submission queue,
// where the flip lands. Nothing else in the app may borrow these.
type Difficulty int

const (
	Easy Difficulty = iota
	Medium
	Hard
)

// LeetCode's difficulty palette.
const (
	DiffEasy   = lipgloss.Color("#1CBABA")
	DiffMedium = lipgloss.Color("#FFB700")
	DiffHard   = lipgloss.Color("#F63737")
)

// Color returns the difficulty's hue.
func (d Difficulty) Color() lipgloss.Color {
	switch d {
	case Hard:
		return DiffHard
	case Medium:
		return DiffMedium
	default:
		return DiffEasy
	}
}

// Label is the difficulty spelled out, as LeetCode writes it.
func (d Difficulty) Label() string {
	switch d {
	case Hard:
		return "Hard"
	case Medium:
		return "Medium"
	default:
		return "Easy"
	}
}

// Render returns the three-letter difficulty tag in LeetCode's colour.
//
// Three letters rather than the full word: the column is scanned, not read, and a fixed
// width is what keeps the grid rigid.
func (d Difficulty) Render() string {
	tag := "ESY"
	switch d {
	case Hard:
		tag = "HRD"
	case Medium:
		tag = "MED"
	}
	return lipgloss.NewStyle().Foreground(d.Color()).Bold(true).Render(tag)
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
		return lipgloss.NewStyle().Foreground(Amber).Bold(true).Render(Chars().Cursor + " " + num)
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
