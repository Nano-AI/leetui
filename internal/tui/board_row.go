package tui

import (
	"fmt"
	"strings"

	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// viewProblemRow renders one row, banded so the eye can track along it.
//
// index decides the band; selected overrides it. Without banding a wide row is a set of
// columns that happen to be on the same line, and finding a title's state means counting
// across — which is what the vertical rules alone could never fix.
func (m Model) viewProblemRow(r store.Row, index int, selected bool, c boardCols, terms []string) string {
	diff := difficultyOf(r.Difficulty)

	title := truncate(r.Title, c.title)
	styled := highlight(title, terms, selected)

	cells := []string{
		cell(theme.ID(r.NumericID, selected), c.id),
		cell(todoMark(m.todo[r.Slug]), colTodo),
		cell(styled, c.title),
		cell(diff.Render(), c.diff),
		cell(theme.Meta.Render(acceptance(r.AcRate)), c.ac),
	}
	if c.status > 0 {
		cells = append(cells, cell(rowState(r, m.premium), c.status))
	}
	if c.comp > 0 {
		cells = append(cells,
			cell(theme.Meta.Render(truncate(strings.Join(r.Companies, " "), c.comp)), c.comp))
	}
	row := components.Row(cells)
	switch {
	case selected:
		return components.Paint(row, theme.Cursor)
	case index%2 == 1:
		return components.Paint(row, theme.Band)
	default:
		return row
	}
}

// todoMark is the one-cell column showing whether a problem is on the user's list.
//
// A filled dot, in amber because it is the user's own annotation rather than LeetCode's
// data. One cell wide: it is a yes-or-no, and anything larger would compete with the
// title for the eye.
func todoMark(on bool) string {
	if !on {
		return " "
	}
	return lipgloss.NewStyle().Foreground(theme.Amber).Render("●")
}

// rowState is the progress column.
//
// It says what the reader most needs to know about the row, in one word: whether they
// have solved it, tried it, or cannot open it. Words rather than glyphs — a "✓" needs a
// legend, "SOLVED" does not.
//
// LOCKED depends on the ACCOUNT, not on the problem.
//
// It used to read straight off PaidOnly, which meant a Premium subscriber saw "LOCKED"
// on problems they could open perfectly well — the column was reporting a property of
// the problem while claiming to report the reader's own state. With a subscription
// nothing is locked, and leetcode.com shows no lock either; a premium problem is just a
// problem. Signed out counts as locked, because signed out you genuinely cannot read it.
//
// Solved is bone, never green: green belongs to the judge alone.
func rowState(r store.Row, premium bool) string {
	switch {
	case r.Solved():
		return lipgloss.NewStyle().Foreground(theme.Bone).Bold(true).Render("SOLVED")
	case r.Status == "notac":
		return theme.Meta.Render("TRIED")
	case r.PaidOnly && !premium:
		return theme.Meta.Render("LOCKED")
	default:
		return ""
	}
}

// highlight marks the searched terms inside a title so a match is visible at a glance
// rather than left for the reader to find.
func highlight(title string, terms []string, selected bool) string {
	base := theme.Body
	if selected {
		base = lipgloss.NewStyle().Foreground(theme.Bone).Bold(true)
	}
	if len(terms) == 0 {
		return base.Render(title)
	}

	hit := lipgloss.NewStyle().Foreground(theme.Amber).Bold(true)
	lower := strings.ToLower(title)

	// Mark every matched byte range, then render runs. Marking first means overlapping
	// terms cannot double-wrap a character in escape codes.
	marked := make([]bool, len(title))
	for _, t := range terms {
		for from := 0; ; {
			i := strings.Index(lower[from:], t)
			if i < 0 {
				break
			}
			start := from + i
			for j := start; j < start+len(t) && j < len(marked); j++ {
				marked[j] = true
			}
			from = start + len(t)
		}
	}

	var b strings.Builder
	for i := 0; i < len(title); {
		j := i
		for j < len(title) && marked[j] == marked[i] {
			j++
		}
		seg := title[i:j]
		if marked[i] {
			b.WriteString(hit.Render(seg))
		} else {
			b.WriteString(base.Render(seg))
		}
		i = j
	}
	return b.String()
}

func searchTerms(q string) []string {
	var out []string
	for _, f := range strings.Fields(strings.ToLower(q)) {
		if len(f) >= 2 {
			out = append(out, f)
		}
	}
	return out
}

// viewEmptyBoard states what to do next. An empty screen is an invitation to act, never
// a dead end. It picks a shorter phrasing at narrow widths rather than wrapping, since
// the board's row budget is fixed.
func (m Model) viewEmptyBoard(w int) string {
	var long, short string
	switch {
	case m.syncing:
		long, short = "Syncing…", "Syncing…"
	case m.filterActive():
		long, short = "No problems match. Press esc to clear filters.", "No matches. esc clears."
	default:
		long, short = "No problems yet. Press S to sync.", "Press S to sync."
	}
	msg := long
	if len(msg)+2 > w {
		msg = short
	}
	return theme.Meta.Render("  " + truncate(msg, maxInt(w-2, 1)))
}

// acceptance renders a rate as a whole percentage.
//
// No decimal: the column is scanned, and 57% versus 57.9% never changes a decision. The
// exact figure is in the problem's own heading, where precision is the point.
func acceptance(pct float64) string {
	if pct <= 0 {
		return ""
	}
	return fmt.Sprintf("%.0f%%", pct)
}

func difficultyOf(s string) theme.Difficulty {
	switch s {
	case "Hard":
		return theme.Hard
	case "Medium":
		return theme.Medium
	default:
		return theme.Easy
	}
}
