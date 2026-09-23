package tui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/store"
)

// ---------------------------------------------------------------------------
// Study plans (D-031)
// ---------------------------------------------------------------------------
//
// The free half of the curated-list loop: pick a plan, work it in order. Where a company
// pack asks two questions — who, and how recently — a plan asks one, because a plan has
// no timeframe. Pressing enter goes straight to the board.
//
// Choosing a plan filters the board and sorts it by the plan's own running order, so
// Array/String comes before Two Pointers and the board reads as a curriculum. That
// ordering is the whole reason a plan beats a tag filter, exactly as frequency is for a
// pack.

// planSel is the study plan currently filtering the board.
type planSel struct {
	Slug string // e.g. "top-interview-150"
	Name string // e.g. "Top Interview 150"

	// Groups maps problem slug to its chapter, so the board's GROUP column costs no
	// query per row. Loaded with the plan and dropped when it is cleared.
	Groups map[string]string
}

// Active reports whether a plan is filtering the board.
func (p planSel) Active() bool { return p.Slug != "" }

// Label is the plan in one phrase, for a bezel or the rail.
func (p planSel) Label() string {
	if !p.Active() {
		return ""
	}
	return p.Name
}

// openPlans enters the study plan browser, refreshing the registry if it is empty.
//
// Like the company registry, a first press fills it rather than showing an empty list and
// an instruction. Unlike the company registry it costs ~22 requests rather than one, so
// it is only ever done when the list is genuinely empty.
func (m Model) openPlans() (tea.Model, tea.Cmd) {
	m.mode = modePlan
	m.planIdx = 0
	m.planFilter.SetValue("")
	m.planFilter.Focus()

	if len(m.plans) == 0 {
		return m, tea.Batch(textinput.Blink, m.beginPlanRegistry(), m.loadPlans())
	}
	return m, textinput.Blink
}

// loadPlans reads the registry out of the store.
func (m Model) loadPlans() tea.Cmd {
	st := m.store
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ps, err := st.Plans(ctx, "")
		return plansMsg{plans: ps, err: err}
	}
}

// visiblePlans applies the typed filter, in memory — the registry is a couple of dozen
// rows, so there is no round trip and no debounce.
func (m Model) visiblePlans() []store.Plan {
	q := strings.ToLower(strings.TrimSpace(m.planFilter.Value()))
	if q == "" {
		return m.plans
	}
	out := make([]store.Plan, 0, 8)
	for _, p := range m.plans {
		if strings.Contains(strings.ToLower(p.Name), q) ||
			strings.Contains(p.Slug, q) ||
			strings.Contains(strings.ToLower(p.Highlight), q) {
			out = append(out, p)
		}
	}
	return out
}

// applyPlan filters the board to a study plan, pulling it first if it is not stored.
//
// The board switches over immediately either way, the same bargain applyPack makes: a
// plan that is still syncing shows what has landed rather than an empty list and a
// spinner. A plan is one request, so that window is usually a blink.
func (m Model) applyPlan(p store.Plan) (tea.Model, tea.Cmd) {
	m.mode = modeBoard
	m.planFilter.Blur()

	// A plan and a pack are both "the board is showing a curated list", and two of those
	// at once would leave the board sorted by one and labelled with the other.
	m.pack = pack{}
	m.plan = planSel{Slug: p.Slug, Name: p.Name}
	m.filter = store.Filter{
		Plan: p.Slug,
		// The plan's running order is what makes this a curriculum rather than a
		// filtered list.
		Sort: "plan",
	}
	m.cursor, m.scroll = 0, 0

	if p.Stored > 0 {
		return m, tea.Batch(m.loadRows(), m.loadPlanGroups(p.Slug),
			status("Showing "+p.Name+". Press esc to clear.", false))
	}
	return m, tea.Batch(m.loadRows(), m.beginPlan(p.Slug),
		status("Pulling "+p.Name+"…", false))
}

// loadPlanGroups reads each stored problem's chapter, so the board can label rows.
func (m Model) loadPlanGroups(plan string) tea.Cmd {
	st := m.store
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		groups, err := st.PlanGroups(ctx, plan)
		return planGroupsMsg{plan: plan, groups: groups, err: err}
	}
}
