package tui

import (
	"context"
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

	lang, ok := runner.Lookup(cfg.DefaultLang)
	if !ok {
		lang, _ = runner.Lookup("python3")
	}

	return Model{
		engine:    runner.NewLocal(),
		lang:      lang,
		cfg:       cfg,
		store:     st,
		client:    cl,
		sync:      sy,
		keys:      cfg.Actions(),
		focus:     paneBoard,
		search:    search,
		authInput: authIn,
	}
}

// Init loads the board from the store and refreshes the account in the background.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		secondTick(),
		m.loadRows(),
		m.loadAccount(),
	)
}

func secondTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}
