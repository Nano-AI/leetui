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

	// Board state.
	rows      []store.Row
	cursor    int
	scroll    int // index of the first visible row
	filter    store.Filter
	totalRows int

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

	// Auth.
	authInput textinput.Model
	authErr   string
	browsers  []auth.Browser
	importing string // label of the browser currently being read, if any

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

	authIn := textinput.New()
	authIn.Prompt = ""
	authIn.Placeholder = "paste cookies here"
	authIn.CharLimit = 4096
	// The pasted blob contains a session token. Echo it masked so it cannot be read off
	// the screen or captured in a screen recording.
	authIn.EchoMode = textinput.EchoPassword

	companyIn := textinput.New()
	companyIn.Prompt = ""
	companyIn.Placeholder = "type to narrow"
	companyIn.CharLimit = 60

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
		authInput:     authIn,
		companyFilter: companyIn,
	}
}

// Init loads the board from the store and refreshes the account in the background.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		secondTick(),
		watchCmd(),
		m.loadRows(),
		m.loadAccount(),
	)
}

func secondTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}
