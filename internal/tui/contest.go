package tui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/store"
)

// ---------------------------------------------------------------------------
// Contests (D-036)
// ---------------------------------------------------------------------------
//
// The third curated list, and the only one with a clock. Picking a contest filters the
// board to its four problems and sorts them by credit, which is the contest's own running
// order — the same bargain a plan makes with its curriculum.
//
// What is different is the countdown. While a contest is live the rail shows the time
// left, and that is the whole reason this is a mode and not a saved filter: a filter does
// not need to know what time it is.

// contestSel is the contest currently filtering the board.
type contestSel struct {
	Slug  string // e.g. "weekly-contest-517"
	Title string // e.g. "Weekly Contest 517"

	// StartTime and Duration are carried on the selection rather than looked up, so the
	// rail can redraw the countdown every second without a query per tick.
	StartTime int64
	Duration  int
}

// Active reports whether a contest is filtering the board.
func (c contestSel) Active() bool { return c.Slug != "" }

// Label is the contest in one phrase, for a bezel or the rail.
func (c contestSel) Label() string {
	if !c.Active() {
		return ""
	}
	return c.Title
}

// brief converts to the API shape, so phase and countdown are computed by the same code
// the CLI uses rather than reimplemented here.
func (c contestSel) brief() leetcode.ContestBrief {
	return leetcode.ContestBrief{
		Title: c.Title, Slug: c.Slug,
		StartTime: c.StartTime, Duration: c.Duration,
	}
}

// PhaseAt reports whether the selected contest is upcoming, live, or ended.
func (c contestSel) PhaseAt(now time.Time) leetcode.Phase { return c.brief().PhaseAt(now) }

// Remaining is the time to the selected contest's next transition.
func (c contestSel) Remaining(now time.Time) time.Duration { return c.brief().Remaining(now) }

// openContests enters the contest browser, refreshing the schedule if it is empty.
//
// One request, so a first press fills the list rather than showing an empty box and an
// instruction — the same call the plan registry makes, and cheaper.
func (m Model) openContests() (tea.Model, tea.Cmd) {
	m.mode = modeContest
	m.contestIdx = 0
	m.contestFilter.SetValue("")
	m.contestFilter.Focus()

	if len(m.contests) == 0 {
		return m, tea.Batch(textinput.Blink, m.beginContestSchedule(), m.loadContests())
	}
	return m, textinput.Blink
}

// loadContests reads the schedule out of the store.
func (m Model) loadContests() tea.Cmd {
	st := m.store
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cs, err := st.Contests(ctx, "")
		return contestsMsg{contests: cs, err: err}
	}
}

// visibleContests applies the typed filter, in memory — the schedule is a short list.
func (m Model) visibleContests() []store.Contest {
	q := strings.ToLower(strings.TrimSpace(m.contestFilter.Value()))
	if q == "" {
		return m.contests
	}
	out := make([]store.Contest, 0, 8)
	for _, c := range m.contests {
		if strings.Contains(strings.ToLower(c.Title), q) || strings.Contains(c.Slug, q) {
			out = append(out, c)
		}
	}
	return out
}

// applyContest filters the board to a contest, pulling it first if it has no problems.
//
// ALWAYS REFETCHES WHILE THE CONTEST IS NOT YET OVER. Everywhere else in this package a
// stored list is trusted and a refresh is a refresh; here an empty list is what LeetCode
// returns before the start, so the stored answer for the contest you are about to sit is
// guaranteed to be the wrong one until you ask again.
func (m Model) applyContest(c store.Contest) (tea.Model, tea.Cmd) {
	m.mode = modeBoard
	m.contestFilter.Blur()

	// Three curated lists, one board. Two at once would leave it sorted by one and
	// labelled with another.
	m.pack = pack{}
	m.plan = planSel{}
	m.contest = contestSel{
		Slug: c.Slug, Title: c.Title,
		StartTime: c.StartTime, Duration: c.Duration,
	}
	m.filter = store.Filter{
		Contest: c.Slug,
		// Credit is the contest's running order: the third problem is meant to be read
		// third, and a board sorted by id would scatter them.
		Sort: "contest",
	}
	m.cursor, m.scroll = 0, 0
	// Record the phase now so the tick has an edge to compare against. Picking a contest
	// that has not started is the normal case, and it is what arms the auto-pull.
	m.contestPhase = c.PhaseAt(time.Now())

	stale := c.Stored == 0 || c.PhaseAt(time.Now()) != leetcode.PhaseEnded
	if !stale {
		return m, tea.Batch(m.loadRows(), m.checkRegistration(),
			status("Showing "+c.Title+". Press esc to clear.", false))
	}
	return m, tea.Batch(m.loadRows(), m.beginContest(c.Slug), m.checkRegistration(),
		status("Pulling "+c.Title+"…", false))
}

// checkRegistration asks whether this account is signed up for a contest.
//
// Worth a request of its own because the answer changes what a submission MEANS: an
// unregistered submission is judged, comes back Accepted, and scores nothing. Skipped for
// a contest that is already over, where the answer cannot matter.
//
// Reads the SELECTED contest off the model rather than taking one, so every caller asks
// about the contest the board is actually showing and none can ask about another.
func (m Model) checkRegistration() tea.Cmd {
	c := m.contest
	if !c.Active() || c.PhaseAt(time.Now()) == leetcode.PhaseEnded {
		return nil
	}
	// Not asked while signed out, because the endpoint answers `registered: false` for an
	// expired session rather than refusing it, and a confident wrong warning is worse
	// than none. m.username is the account the app has already confirmed.
	if m.username == "" {
		return nil
	}
	cl := m.client
	slug := c.Slug
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		info, err := cl.Info(ctx, slug)
		if err != nil {
			return contestRegistrationMsg{contest: slug, err: err}
		}
		return contestRegistrationMsg{contest: slug, registered: info.Registered}
	}
}

// contestOpened reports the moment the selected contest crosses from upcoming to live.
//
// This is the transition the whole feature is pointed at. You pick the contest before it
// starts — there is nothing else to do while waiting — and the questions do not exist
// until the clock runs out. Without this the countdown would reach zero and the board
// would sit there empty until you thought to press C and enter again, which is the worst
// possible minute to be navigating menus.
//
// Compares against the phase recorded on the last tick, so it fires ONCE rather than on
// every tick of a live contest.
func (m Model) contestOpened(now time.Time) bool {
	return m.contest.Active() &&
		m.contestPhase == leetcode.PhaseUpcoming &&
		m.contest.PhaseAt(now) == leetcode.PhaseLive
}

// contestClock is the phrase the rail shows for the selected contest.
//
// Empty once the contest has ended: a countdown that has run out is not information, and
// the board is still filtered, which is the part that still matters.
func (m Model) contestClock(now time.Time) string {
	if !m.contest.Active() {
		return ""
	}
	switch m.contest.PhaseAt(now) {
	case leetcode.PhaseUpcoming:
		return "starts in " + formatDuration(m.contest.Remaining(now))
	case leetcode.PhaseLive:
		return formatDuration(m.contest.Remaining(now)) + " left"
	default:
		return ""
	}
}

// contestLive reports whether the selected contest is running right now.
//
// The rail colours the clock on this: amber while it is the thing you are racing, dim
// otherwise. It is the one place in the app where a colour means "hurry".
func (m Model) contestLive(now time.Time) bool {
	return m.contest.Active() && m.contest.PhaseAt(now) == leetcode.PhaseLive
}
