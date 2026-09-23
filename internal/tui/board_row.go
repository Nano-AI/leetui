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

	// A marked row is styled to match where the sort has put it (D-032b). Unimportant
	// goes grey throughout and sinks; important is gilded and floats. Sinking or floating
	// moves a row out of or into the way; the colour is what makes it read that way
	// without counting positions. Neither half does the job alone.
	//
	// The cursor outranks both. A row you have deliberately moved onto must render
	// normally, whatever you decided about it earlier — the selection has to be the
	// loudest thing on the board or it is not a selection.
	mark := m.marks[r.Slug]
	dim := !selected && mark == store.MarkDown
	gold := !selected && mark == store.MarkUp

	title := truncate(r.Title, c.title)
	styled := highlight(title, terms, selected)
	switch {
	case dim:
		// Search terms are not highlighted on a dimmed row: amber on grey would be the one
		// bright thing in a row whose whole job is to recede.
		styled = theme.Meta.Render(title)
	case gold:
		// Gilded, but NOT bold. Bold is the cursor's, and a board with a dozen bold rows
		// on it has no cursor.
		styled = lipgloss.NewStyle().Foreground(theme.Amber).Render(title)
	}

	cells := []string{
		cell(rowID(r.NumericID, selected, gold), c.id),
		cell(todoMark(m.todo[r.Slug], dim), colTodo),
	}
	if c.mark > 0 {
		cells = append(cells, cell(importanceMark(m.marks[r.Slug]), c.mark))
	}
	diffCell := diff.Render()
	if dim {
		diffCell = theme.Meta.Render(diff.Tag())
	}
	cells = append(cells,
		cell(styled, c.title),
		cell(diffCell, c.diff),
		cell(theme.Meta.Render(acceptance(r.AcRate)), c.ac),
	)
	if c.status > 0 {
		cells = append(cells, cell(rowState(r, m.premium, dim), c.status))
	}
	if c.comp > 0 {
		cells = append(cells, cell(theme.Meta.Render(truncate(m.lastColText(r), c.comp)), c.comp))
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

// lastColText fills the final column: the plan's chapter when one is active, otherwise
// the companies that ask this problem. See lastColLabel for why they share a slot.
//
// A missing chapter renders empty rather than falling back to companies. The header says
// CHAPTER, and a row that quietly answered a different question than its header would be
// the exact mistake the state column already made twice (D-023).
func (m Model) lastColText(r store.Row) string {
	if m.plan.Active() {
		return m.plan.Groups[r.Slug]
	}
	return strings.Join(r.Companies, " ")
}

// todoMark shows whether a problem is on the user's list.
//
// A glyph is only allowed to be a glyph when something else names it, and the FIRST thing
// asked about this column was what the dot meant — because its header was blank. The
// header is now "TODO"; the mark sits under it and needs no legend of its own. Do not
// remove that header. Amber because it is the user's own annotation, not LeetCode's data.
func todoMark(on, dim bool) string {
	if !on {
		return ""
	}
	if dim {
		return theme.Center(theme.Meta.Render(theme.Glyphs().Todo), colTodo)
	}
	return theme.Center(
		lipgloss.NewStyle().Foreground(theme.Amber).Render(theme.Glyphs().Todo), colTodo)
}

// rowID draws the problem number, gilded when the row is marked important.
//
// theme.ID owns the selected and normal cases, including the cursor bar, so this only
// intercepts the gold one — a second copy of the selection rule is a second place for it
// to drift.
func rowID(id int, selected, gold bool) string {
	if !gold {
		return theme.ID(id, selected)
	}
	return lipgloss.NewStyle().Foreground(theme.Amber).Render("  " + theme.Pad4(id))
}

// importanceMark is the user's verdict on a problem (D-032).
//
// Green is not available here — it belongs to the judge alone — so the two directions are
// separated by weight rather than by hue: an important problem is bold amber and pulls the
// eye down a column, a written-off one is dim and recedes. That is the right emphasis for
// a second pass, where you are hunting for the pluses.
//
// The glyphs are literally the keys that set them, so the column needs no legend beyond
// its MARK header.
func importanceMark(mk store.Mark) string {
	g := theme.Glyphs()
	switch mk {
	case store.MarkUp:
		return theme.Center(
			lipgloss.NewStyle().Foreground(theme.Amber).Bold(true).Render(g.Important), colMark)
	case store.MarkDown:
		return theme.Center(theme.Meta.Render(g.Unimportant), colMark)
	default:
		return ""
	}
}

// rowState is the progress column: solved, tried, or out of reach.
//
// A glyph rather than a word (D-023). `✓ ✓ ◐ ⊘ ✓` scans down a column in a way
// `SOLVED SOLVED TRIED LOCKED SOLVED` does not, and it costs two fewer cells. That trade
// only holds because the column is headed STATE — an unheaded glyph is the mistake this
// board has already made twice.
//
// theme.Glyphs falls back to ASCII where the terminal cannot be trusted with the width of
// `✓`, which is Ambiguous and may be drawn two cells wide.
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
func rowState(r store.Row, premium, dim bool) string {
	g := theme.Glyphs()

	// A dimmed row keeps its state glyph — whether it is solved is still true, and losing
	// the mark entirely would make an unimportant row look untouched. It just stops
	// announcing it.
	if dim {
		switch {
		case r.Solved():
			return theme.Center(theme.Meta.Render(g.Solved), colStatus)
		case r.Status == "notac":
			return theme.Center(theme.Meta.Render(g.Tried), colStatus)
		case r.PaidOnly && !premium:
			return theme.Center(theme.Meta.Render(g.Locked), colStatus)
		default:
			return ""
		}
	}

	switch {
	case r.Solved():
		// Bone and bold, never green: green belongs to the judge alone.
		return theme.Center(
			lipgloss.NewStyle().Foreground(theme.Bone).Bold(true).Render(g.Solved), colStatus)
	case r.Status == "notac":
		return theme.Center(theme.Label.Render(g.Tried), colStatus)
	case r.PaidOnly && !premium:
		return theme.Center(theme.Meta.Render(g.Locked), colStatus)
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
