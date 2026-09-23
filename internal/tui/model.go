package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/auth"
	"github.com/Nano-AI/leetui/internal/config"
	"github.com/Nano-AI/leetui/internal/editor"
	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/render"
	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/syncer"
	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
	"github.com/charmbracelet/bubbles/textinput"
)

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

// Model is the application root.
type Model struct {
	// Wiring, injected by New.
	cfg    config.Config
	store  *store.Store
	client *leetcode.Client
	sync   *syncer.Syncer
	keys   map[string]string // key -> action, from config (D-013)

	width, height int
	mode          mode
	focus         pane

	// todo holds the slugs on the user's list, so a row can be marked without a query
	// per row. Refreshed with the rows.
	todo map[string]bool

	// marks holds the importance verdict per slug (D-032), on the same terms as todo:
	// read once with the rows, written through on a keypress. Absent means no opinion.
	marks map[string]store.Mark

	// Board state.
	rows       []store.Row
	rowsSeq    int
	rowsFilter store.Filter // filter used by the last accepted row snapshot
	cursor     int
	scroll     int // index of the first visible row
	filter     store.Filter
	totalRows  int

	// Search.
	searching bool
	search    textinput.Model

	// Detail pane.
	detail        *store.Detail
	detailMD      string
	detailImages  []render.Image
	detailScroll  int
	detailLoading bool

	// detailSeq rises on every cursor move so a slow in-flight fetch for a problem the
	// user has already scrolled past can be recognised as stale and dropped.
	detailSeq int

	// Editorial (D-006). The detail pane shows either the statement or the editorial,
	// never both; showEditorial says which.
	showEditorial    bool
	editorial        *store.Editorial
	editorialMD      string
	editorialImages  []render.Image
	editorialLoading bool
	editorialSeq     int

	// Company packs (D-006).
	//
	// pack is the pack currently filtering the board, zero when browsing everything.
	// companies is the registry, loaded once and filtered in memory — 984 rows is
	// nothing to search locally and a round trip per keystroke would be.
	pack          pack
	companies     []store.Company
	companyIdx    int
	companyFilter textinput.Model

	// packChoice is the company picked in the browser, waiting on a timeframe.
	// packCounts is how much of each of its timeframes is already stored, so the
	// timeframe picker can say which are a keypress away and which need a pull.
	packChoice store.Company
	packCounts map[leetcode.Timeframe]int

	// Study plans (D-031).
	//
	// plan is the plan currently filtering the board, zero when browsing everything.
	// plans is the registry, loaded once and filtered in memory — a couple of dozen rows.
	// Mutually exclusive with pack: the board is sorted by one or the other, never both.
	plan       planSel
	plans      []store.Plan
	planIdx    int
	planFilter textinput.Model

	// Contests (D-036).
	//
	// The same four fields as a plan, and one difference in how they are used: contest
	// SURVIVES leaving the board filter, because the rail counts the contest down and a
	// cleared selection would stop the clock mid-contest.
	contest       contestSel
	contests      []store.Contest
	contestIdx    int
	contestFilter textinput.Model

	// contestPhase is the phase the selected contest was in on the last tick, and exists
	// only so the upcoming→live crossing can be noticed once. Without a remembered value
	// there is no edge to detect, only a level.
	contestPhase leetcode.Phase

	// Auth.
	//
	// Two fields rather than one box holding both secrets. The single-field form asked
	// the user to assemble "LEETCODE_SESSION=…; csrftoken=…" themselves, which is not
	// the shape devtools hands you when you copy the two values out of its cookie table.
	authFields [authFieldCount]textinput.Model
	authFocus  int  // authFocusBrowsers, or authFocusSession/authFocusCSRF
	authReveal bool // unmasked, so a bad paste can actually be seen
	authVerify bool // a check against LeetCode is in flight
	authErr    string
	browsers   []auth.Browser
	browserIdx int    // highlighted row in the browser list
	importing  string // label of the browser currently being read, if any

	// importedFrom is the browser a set of credentials came from, remembered across
	// the verification round trip so the confirmation can name it.
	importedFrom string

	// Solve loop.
	engine    runner.Engine
	lang      runner.Lang
	picking   pickKind // which picker is open; pickNone when closed
	pickIdx   int
	editors   []editor.Editor
	runResult *runner.Result
	runSlug   string
	running   bool
	// watched remembers each solution file's last modification time, keyed by
	// slug/lang, so a save can be told apart from a first sighting.
	watched map[string]time.Time

	// queueOnTop makes the side pane show submissions rather than the local run.
	// Set by submitting and cleared by running, so the pane shows whichever the user
	// asked for last rather than letting a stale run result hide a live verdict.
	queueOnTop bool

	// Submission queue.
	queue      []queueItem
	nextFlapID int

	// celebrate is the colour sweep running over an Accepted verdict (D-033). The zero
	// value is not celebrating, which is also what it settles back to.
	celebrate celebration

	// Sync.
	syncing      bool
	syncProgress syncer.Progress
	syncCh       chan syncer.Progress
	syncCancel   context.CancelFunc

	// booted guards the one-time first-run check, so a later empty result (a filter
	// that matches nothing) never re-triggers setup.
	booted bool

	// Account.
	username string
	premium  bool

	// Timer — mirrors LeetCode's own top-right stopwatch (D-006). A plain timer, not
	// assessment scoring; mock assessments are explicitly out of scope.
	timerRunning bool
	elapsed      time.Duration

	// Floating notice over the top right, for something that happened and is now needed
	// — a created file's path, above all. Separate from the status line, which comments
	// on what you just did rather than handing you something.
	toast   *toast
	toastID int

	// Command palette and settings — two views onto one registry
	// (internal/config/settings.go), so they cannot disagree about what exists.
	paletteOpen bool
	palette     paletteState
	settingsIdx int
	helpScroll  int
	docsScroll  int

	// Inline figures (D-007, finally wired). graphics is decided once at startup;
	// images caches what has been fetched and uploaded, keyed by URL so a figure
	// shared between the statement and the editorial is transmitted once.
	graphics    config.Graphics
	images      map[string]*imageState
	nextImageID int

	// git is the repository view's state (D-011). Read on demand rather than kept
	// current: a status refresh per keystroke would take the index lock away from
	// whatever the user has running in the next pane.
	git gitPane

	// Transient status line.
	status    string
	statusErr bool
	statusID  int
}

type queueItem struct {
	ProblemID int
	Lang      string
	Verdict   theme.Verdict
	flap      components.Flap

	// What the judge said, kept so the row can show it once the flap settles.
	//
	// The verdict alone answers "did it pass". These answer the question you ask half a
	// second later — how fast, and against whom — and going to leetcode.com to find out
	// defeats the point of submitting from here.
	Runtime    string // "58 ms"
	Memory     string // "20.2 MB"
	Percentile float64

	// MemoryPct is the memory percentile. Kept beside the runtime one because a result
	// is only a "double 50" if both halves say so (D-033), and because "beats 94% on
	// time, 12% on memory" is a different sentence from "beats 94%".
	MemoryPct float64

	// Tier grades the figures once, when the verdict lands, so the badge does not have
	// to be recomputed on every frame of every repaint.
	Tier celebrationTier

	// Correct and Total are how far a failing submission got. Nothing else on screen
	// distinguishes "wrong on case 3 of 63" from "wrong on 62 of 63", and those call for
	// completely different next moves.
	Correct, Total int
}

// stats is the one-line summary shown under a settled verdict, or "" when there is
// nothing worth saying yet.
func (q queueItem) stats() string {
	switch {
	case q.Verdict == theme.Accepted:
		out := q.Runtime
		if q.Percentile > 0 {
			out += fmt.Sprintf(" · beats %.0f%%", q.Percentile)
		}
		if q.Memory != "" {
			out += " · " + q.Memory
		}
		// The memory percentile only earns its width when it is there. A judge that
		// reported no figure must not render as "beats 0%", which would read as a result
		// rather than as an absence.
		if q.MemoryPct > 0 {
			out += fmt.Sprintf(" · beats %.0f%%", q.MemoryPct)
		}
		return strings.TrimPrefix(out, " · ")
	case q.Total > 0:
		return fmt.Sprintf("%d/%d cases", q.Correct, q.Total)
	default:
		return ""
	}
}

// New builds the root model. Everything it needs is injected so main owns lifetimes and
// the model stays testable without a network or a real database.
func New(cfg config.Config, st *store.Store, cl *leetcode.Client, sy *syncer.Syncer) Model {
	search := textinput.New()
	search.Prompt = ""
	search.Placeholder = "search titles, tags, statements"
	search.CharLimit = 120

	// Both cookie fields are masked by default, because they are secrets and the
	// terminal they are typed into may be shared or recorded. ctrl+r unmasks them:
	// verifying an 800-character JWT you cannot see is not possible, and the old form
	// offered no way to look.
	var authFields [authFieldCount]textinput.Model
	for i := range authFields {
		in := textinput.New()
		in.Prompt = ""
		in.EchoMode = textinput.EchoPassword
		in.Placeholder = "paste the value, or a whole cookie header"
		// Sized for a LeetCode session JWT, which runs past 800 characters, with room
		// for the cookie header a smart paste arrives as.
		in.CharLimit = 4096
		authFields[i] = in
	}

	companyIn := textinput.New()
	companyIn.Prompt = ""
	companyIn.Placeholder = "type to narrow"
	companyIn.CharLimit = 60

	planIn := textinput.New()
	planIn.Prompt = ""
	planIn.Placeholder = "type to narrow"
	planIn.CharLimit = 60

	contestIn := textinput.New()
	contestIn.Prompt = ""
	contestIn.Placeholder = "type to narrow"
	contestIn.CharLimit = 60

	// The remembered language wins over the configured default: what you were writing
	// last time is a better guess than a preference set once during setup.
	lang, ok := runner.Lookup(cfg.LastLang)
	if !ok {
		if lang, ok = runner.Lookup(cfg.DefaultLang); !ok {
			lang, _ = runner.Lookup("python3")
		}
	}

	return Model{
		engine:        runner.NewLocal(),
		graphics:      detectGraphics(),
		images:        map[string]*imageState{},
		lang:          lang,
		cfg:           cfg,
		store:         st,
		client:        cl,
		sync:          sy,
		keys:          cfg.Actions(),
		focus:         paneBoard,
		search:        search,
		authFields:    authFields,
		companyFilter: companyIn,
		planFilter:    planIn,
		contestFilter: contestIn,
	}
}

// Init loads the board from the store and refreshes the account in the background.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		secondTick(),
		watchCmd(),
		m.queryRows(),
		m.loadAccount(),
	)
}

func secondTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}
