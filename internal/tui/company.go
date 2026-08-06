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
// Company packs (D-006)
// ---------------------------------------------------------------------------
//
// The premium loop the website is built around: pick a company, pick how recently it
// asked, work the list. leetui models it as a two-step choice — company, then timeframe —
// because the second question only makes sense once the first is answered, and asking
// both at once would mean 984 × 5 rows.
//
// Choosing a pack filters the board and sorts it by frequency, so the problems that
// company asks most sit at the top. That ordering is the whole reason a pack beats a
// tag filter.

// pack is the company list currently filtering the board.
type pack struct {
	Company   string // slug, e.g. "facebook"
	Name      string // display name, e.g. "Meta"
	Timeframe leetcode.Timeframe
}

// Active reports whether a pack is filtering the board.
func (p pack) Active() bool { return p.Company != "" }

// Label is the pack in one phrase, for a bezel or the rail.
func (p pack) Label() string {
	if !p.Active() {
		return ""
	}
	return p.Name + " ┊ " + p.Timeframe.Label()
}

// openCompanies enters the company browser, refreshing the registry if it is empty.
//
// The registry needs no session and is one request, so a first press fills it rather
// than showing an empty list and an instruction.
func (m Model) openCompanies() (tea.Model, tea.Cmd) {
	m.mode = modeCompany
	m.companyIdx = 0
	m.companyFilter.SetValue("")
	m.companyFilter.Focus()

	if len(m.companies) == 0 {
		return m, tea.Batch(textinput.Blink, m.beginRegistry(), m.loadCompanies())
	}
	return m, textinput.Blink
}

// loadCompanies reads the registry out of the store.
func (m Model) loadCompanies() tea.Cmd {
	st := m.store
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cs, err := st.Companies(ctx, "")
		return companiesMsg{companies: cs, err: err}
	}
}

// visibleCompanies applies the typed filter. 984 rows filter instantly in memory, so
// there is no round trip and no debounce.
func (m Model) visibleCompanies() []store.Company {
	q := strings.ToLower(strings.TrimSpace(m.companyFilter.Value()))
	if q == "" {
		return m.companies
	}
	out := make([]store.Company, 0, 32)
	for _, c := range m.companies {
		if strings.Contains(strings.ToLower(c.Name), q) || strings.Contains(c.Slug, q) {
			out = append(out, c)
		}
	}
	return out
}

// applyPack filters the board to a company pack, pulling it first if it is not stored.
//
// The board is switched over immediately either way. A pack that is still syncing shows
// what has landed rather than an empty list with a spinner, which is the same bargain
// the first-run problem sync makes.
func (m Model) applyPack(c store.Company, tf leetcode.Timeframe, stored int) (tea.Model, tea.Cmd) {
	m.mode = modeBoard
	m.companyFilter.Blur()

	m.pack = pack{Company: c.Slug, Name: c.Name, Timeframe: tf}
	m.filter = store.Filter{
		Companies: []string{c.Slug},
		Timeframe: string(tf),
		// Frequency is what makes this a pack rather than a filtered list: the problems
		// this company asks most come first.
		Sort: "frequency",
	}
	m.cursor, m.scroll = 0, 0

	if stored > 0 {
		return m, tea.Batch(m.loadRows(),
			status("Showing "+m.pack.Label()+". Press esc to clear.", false))
	}
	return m, tea.Batch(m.loadRows(), m.beginPack(c.Slug, tf),
		status("Pulling "+c.Name+"'s "+tf.Label()+" list…", false))
}

// clearPack drops back to the whole problem set.
func (m Model) clearPack() (tea.Model, tea.Cmd) {
	m.pack = pack{}
	m.filter = store.Filter{}
	m.cursor, m.scroll = 0, 0
	return m, m.loadRows()
}
