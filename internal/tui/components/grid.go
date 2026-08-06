package components

import (
	"strings"

	"github.com/grootbeat/leetui/internal/tui/theme"
)

// RuleRow builds a horizontal divider that lines up with column separators, e.g.
//
//	├──────┼──────────┼─────┤
//
// widths are the column widths, in order. Used for the line under a column header.
func RuleRow(widths []int, focused bool) string {
	edge := theme.Rule
	if focused {
		edge = theme.RuleFocused
	}
	parts := make([]string, len(widths))
	for i, w := range widths {
		// +2 for the single space of padding on each side of a cell.
		parts[i] = strings.Repeat(edgeH, w+2)
	}
	return edge.Render(strings.Join(parts, SepCross))
}

// Row joins cells with padded vertical separators:
//
//	cell │ cell │ cell
//
// Cells must already be sized; Row only adds the separators and padding.
func Row(cells []string) string {
	sep := theme.Rule.Render(" " + SepV + " ")
	return " " + strings.Join(cells, sep) + " "
}

// RowWidth returns the total width Row produces for the given column widths, so callers
// can solve a flexible column without guessing.
func RowWidth(widths []int) int {
	total := 2 // leading and trailing pad
	for i, w := range widths {
		total += w
		if i > 0 {
			total += 3 // " │ "
		}
	}
	return total
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
