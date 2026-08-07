package components

import (
	"strings"

	"github.com/Nano-AI/leetui/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// A Frame is the bezel around a pane.
//
// The board is a physical fixture, not floating text — a split-flap board is a housing
// with a bezel, a header strip, and rigid column dividers. Panes without frames read as
// soup, which is exactly how the first pass looked.
//
// Corners are rounded because a board housing is a moulded object, not a cut sheet.
//
// Focus is shown by the bezel turning amber. That is the ONLY focus indicator — no
// background shift, no border-weight change (docs/DESIGN.md § Structure).
type Frame struct {
	// Title is the left label set into the top bezel.
	Title string
	// Right is an optional right-aligned label in the top bezel, for counts.
	Right string

	Width, Height int
	Focused       bool

	// RuleRows lists body line indices that are horizontal dividers. Those lines get
	// tee joints (├ ┤) instead of plain side bezels, so the divider reads as part of
	// the frame rather than a stray line floating inside it.
	RuleRows []int

	// Columns, when set, are the column widths inside this frame. The bottom bezel
	// grows tee joints (┴) beneath each column rule so the grid closes instead of
	// running into a blank edge.
	Columns []int
}

// Box-drawing set. Rounded corners, light lines — the bezel frames the content, it does
// not compete with it.
const (
	cornerTL = "╭"
	cornerTR = "╮"
	cornerBL = "╰"
	cornerBR = "╯"
	edgeH    = "─"
	edgeV    = "│"

	// Column separators inside the frame.
	SepV     = "│"
	SepTeeT  = "┬"
	SepTeeB  = "┴"
	SepCross = "┼"
	SepLeft  = "├"
	SepRight = "┤"
)

// InnerWidth is the usable width inside the bezel.
func (f Frame) InnerWidth() int { return maxInt(f.Width-2, 1) }

// InnerHeight is the usable height inside the bezel.
func (f Frame) InnerHeight() int { return maxInt(f.Height-2, 1) }

// Render wraps body in the bezel, padding or truncating to exactly Width x Height.
func (f Frame) Render(body string) string {
	if f.Width < 4 || f.Height < 2 {
		return body
	}

	edge := theme.Rule
	label := theme.Utility
	if f.Focused {
		edge = theme.RuleFocused
		label = theme.Label
	}

	var out strings.Builder
	out.WriteString(f.top(edge, label))
	out.WriteString("\n")

	rules := map[int]bool{}
	for _, r := range f.RuleRows {
		rules[r] = true
	}

	inner := f.InnerWidth()
	lines := strings.Split(body, "\n")
	for i := 0; i < f.InnerHeight(); i++ {
		var line string
		if i < len(lines) {
			line = lines[i]
		}
		l, r := edgeV, edgeV
		if rules[i] {
			l, r = SepLeft, SepRight
		}
		// ONE BODY LINE IS ONE FRAME ROW. Always.
		//
		// lipgloss's Width() PADS a short line and WRAPS a long one, and the wrap is
		// what breaks the bezel: an over-long line becomes two visual rows, the right
		// edge lands after the second, and every row below is shifted by one — the
		// border appears to shear off mid-pane. A statement's code blocks are not
		// wrapped by Glamour, so any problem with a long example does this.
		//
		// Truncating here is the invariant, not the fix: content is wrapped to width
		// before it ever reaches a frame. This is the guarantee that a pane cannot be
		// corrupted by whatever it was handed.
		out.WriteString(edge.Render(l))
		out.WriteString(pad(ansi.Truncate(line, inner, ""), inner))
		out.WriteString(edge.Render(r))
		out.WriteString("\n")
	}

	out.WriteString(edge.Render(f.bottom(inner)))
	return out.String()
}

func (f Frame) bottom(inner int) string {
	if len(f.Columns) < 2 {
		return cornerBL + strings.Repeat(edgeH, inner) + cornerBR
	}
	parts := make([]string, len(f.Columns))
	for i, w := range f.Columns {
		parts[i] = strings.Repeat(edgeH, w+2)
	}
	mid := strings.Join(parts, SepTeeB)
	// Fall back to a plain edge if the columns do not add up — a mis-sized bezel is
	// worse than a plain one.
	if len([]rune(mid)) != inner {
		return cornerBL + strings.Repeat(edgeH, inner) + cornerBR
	}
	return cornerBL + mid + cornerBR
}

// top builds the header bezel with the title set into it.
func (f Frame) top(edge, label lipgloss.Style) string {
	inner := f.InnerWidth()

	left := ""
	if f.Title != "" {
		left = " " + theme.UtilityText(f.Title) + " "
	}
	right := ""
	if f.Right != "" {
		right = " " + theme.UtilityText(f.Right) + " "
	}

	// A dash on each side of the labels so they sit set into the bezel rather than
	// jammed against a corner.
	lead, tail := 1, 1
	if right == "" {
		tail = 0
	}

	fill := inner - lead - tail - lipgloss.Width(left) - lipgloss.Width(right)
	if fill < 0 {
		// Not enough room for both labels; the count is the one to drop.
		right, tail = "", 0
		fill = inner - lead - lipgloss.Width(left)
	}
	if fill < 0 {
		left, right, tail, fill = "", "", 0, inner-lead
	}
	if fill < 0 {
		return edge.Render(cornerTL + strings.Repeat(edgeH, inner) + cornerTR)
	}

	return edge.Render(cornerTL+strings.Repeat(edgeH, lead)) +
		label.Render(left) +
		edge.Render(strings.Repeat(edgeH, fill)) +
		label.Render(right) +
		edge.Render(strings.Repeat(edgeH, tail)+cornerTR)
}

// pad right-fills a line to exactly w cells.
//
// Deliberately not lipgloss's Width(): that wraps anything longer than w, which is the
// bug this replaced. The caller has already truncated, so this only ever adds.
func pad(s string, w int) string {
	gap := w - ansi.StringWidth(s)
	if gap <= 0 {
		return s
	}
	return s + strings.Repeat(" ", gap)
}
