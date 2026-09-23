package tui

import (
	"fmt"
	"strings"

	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// ---------------------------------------------------------------------------
// Board
// ---------------------------------------------------------------------------

// boardCols is the resolved column layout.
//
// Columns are separated by real vertical rules, the way a departure board's flap groups
// are. That grid is what makes a dense list scannable; without it the rows read as soup.
//
// The acceptance column is a PERCENTAGE, not a sparkline.
//
// It was three sparkline cells, on the reasoning that a bar and a "55.1%" label are the
// same fact twice. They are not: "58%" says what it is, and "█▆▂" needs a legend nobody
// has. The first person to see the board asked what it meant, which is the whole answer.
// Clever loses to legible on a scanning surface.
type boardCols struct {
	id     int
	todo   int
	mark   int // 0 when dropped
	title  int // flexes to absorb the remaining space
	diff   int
	ac     int
	status int // 0 when dropped
	comp   int // 0 when dropped
}

const (
	colID     = theme.IDWidth
	colTodo   = 4
	colMark   = 4
	colDiff   = 3
	colAC     = 4
	colStatus = 5
	colComp   = 16
	minTitle  = 16
)

// widths returns the column widths in render order, omitting dropped columns.
//
// MARK sits beside TODO because both are the user's own annotations rather than
// LeetCode's data, and keeping them together means everything you wrote yourself is at
// the left edge.
func (c boardCols) widths() []int {
	w := []int{c.id, c.todo}
	if c.mark > 0 {
		w = append(w, c.mark)
	}
	w = append(w, c.title, c.diff, c.ac)
	if c.status > 0 {
		w = append(w, c.status)
	}
	if c.comp > 0 {
		w = append(w, c.comp)
	}
	return w
}

// boardLayout solves the columns so a row is exactly inner cells wide.
//
// Columns are dropped in order of how little they carry: companies first (premium
// metadata), then the importance mark, then the progress column. The ID, title,
// difficulty, and acceptance rate are never dropped — they are the row.
//
// MARK yields before STATE because solved-or-not is the more fundamental fact about a
// row, and a verdict you recorded yourself is one you can still reach with `i`.
func boardLayout(inner int) boardCols {
	c := boardCols{id: colID, todo: colTodo, mark: colMark, diff: colDiff, ac: colAC,
		status: colStatus, comp: colComp}

	fit := func(cc boardCols) int {
		w := cc.widths()
		// Solve title so components.RowWidth(w) == inner.
		fixed := components.RowWidth(w) - cc.title
		return inner - fixed
	}

	if c.title = fit(c); c.title >= minTitle {
		return c
	}
	c.comp = 0
	if c.title = fit(c); c.title >= minTitle {
		return c
	}
	c.mark = 0
	if c.title = fit(c); c.title >= minTitle {
		return c
	}
	c.status = 0
	if c.title = fit(c); c.title < 8 {
		c.title = 8
	}
	return c
}

func (m Model) viewBoard(w, h int) string {
	focused := m.focus == paneBoard
	f := components.Frame{
		Title:   "problems",
		Right:   m.boardSummary(),
		Width:   w,
		Height:  h,
		Focused: focused,
	}

	// The divider sits at body line 1, so the frame can tee it into the bezel, and the
	// column widths let the bottom bezel close the grid.
	f.RuleRows = []int{1}

	c := boardLayout(f.InnerWidth())
	widths := c.widths()
	f.Columns = widths

	head := []string{
		cell(theme.Utility.Render(theme.UtilityText("#")), c.id),
		cell(theme.Utility.Render(theme.UtilityText("todo")), c.todo),
	}
	if c.mark > 0 {
		head = append(head, cell(theme.Utility.Render(theme.UtilityText("mark")), c.mark))
	}
	head = append(head,
		cell(theme.Utility.Render(theme.UtilityText("problem")), c.title),
		cell(theme.Utility.Render(theme.UtilityText("dif")), c.diff),
		cell(theme.Utility.Render(theme.UtilityText("acc")), c.ac),
	)
	if c.status > 0 {
		head = append(head, cell(theme.Utility.Render(theme.UtilityText("state")), c.status))
	}
	if c.comp > 0 {
		// One slot, two jobs (D-031). Under a study plan the companies that ask a problem
		// are beside the point and its chapter is the whole story — and a plan and a pack
		// can never both be active, so the column is never asked to be both at once.
		// Reusing it beats a seventh column on a board that already drops this one first.
		head = append(head, cell(theme.Utility.Render(theme.UtilityText(m.lastColLabel())), c.comp))
	}

	var b strings.Builder
	b.WriteString(components.Row(head))
	b.WriteString("\n")
	b.WriteString(components.RuleRow(widths, focused))
	b.WriteString("\n")

	visible := maxInt(h-frameChrome-headerRows, 1)

	if len(m.rows) == 0 {
		b.WriteString(m.viewEmptyBoard(f.InnerWidth()))
		b.WriteString("\n")
		// Keep the column rules running to the bottom bezel even with nothing to show;
		// a grid that stops halfway reads as broken rather than empty.
		b.WriteString(blankRows(widths, visible-1))
		return f.Render(b.String())
	}

	end := minInt(m.scroll+visible, len(m.rows))
	terms := searchTerms(m.filter.Text)
	for i := m.scroll; i < end; i++ {
		b.WriteString(m.viewProblemRow(m.rows[i], i, i == m.cursor, c, terms))
		b.WriteString("\n")
	}
	// Pad short result sets so the grid stays a solid block down to the bezel.
	b.WriteString(blankRows(widths, visible-(end-m.scroll)))

	return f.Render(b.String())
}

// lastColLabel names the final column for whichever list is filtering the board.
func (m Model) lastColLabel() string {
	if m.plan.Active() {
		return "chapter"
	}
	return "asked by"
}

// blankRows draws n empty grid rows, preserving the column rules.
func blankRows(widths []int, n int) string {
	if n <= 0 {
		return ""
	}
	cells := make([]string, len(widths))
	for i, w := range widths {
		cells[i] = strings.Repeat(" ", w)
	}
	row := components.Row(cells) + "\n"
	return strings.Repeat(row, n)
}

// boardSummary is the right-hand bezel label: what is filtering the list and how many
// rows survived. The count on screen is never left unexplained.
func (m Model) boardSummary() string {
	var parts []string
	// The pack comes first: it is the strongest claim about what the list is, and
	// "Meta ┊ last 3 months" explains an ordering that would otherwise look arbitrary.
	if m.filter.TodoOnly {
		parts = append(parts, "my list")
	}
	if m.filter.Mark != store.MarkNone {
		parts = append(parts, m.filter.Mark.Label())
	}
	if m.pack.Active() {
		parts = append(parts, m.pack.Label())
	}
	if m.plan.Active() {
		parts = append(parts, m.plan.Label())
	}
	if m.filter.PaidOnly != nil {
		if *m.filter.PaidOnly {
			parts = append(parts, "premium")
		} else {
			parts = append(parts, "free")
		}
	}
	if len(m.filter.Difficulty) > 0 {
		parts = append(parts, strings.ToLower(strings.Join(m.filter.Difficulty, "+")))
	}
	switch m.filter.Status {
	case "ac":
		parts = append(parts, "solved")
	case "notac":
		parts = append(parts, "tried")
	case "todo":
		parts = append(parts, "unsolved")
	}
	if m.filter.Text != "" {
		parts = append(parts, `"`+m.filter.Text+`"`)
	}

	count := fmt.Sprintf("%d", len(m.rows))
	if len(parts) == 0 {
		return count
	}
	return strings.Join(parts, sep1()) + sep1() + count
}
