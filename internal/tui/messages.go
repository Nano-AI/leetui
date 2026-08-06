package tui

import (
	"time"

	"github.com/Nano-AI/leetui/internal/auth"
	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/render"
	"github.com/Nano-AI/leetui/internal/runner"
	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/syncer"
)

// ---------------------------------------------------------------------------
// Messages
// ---------------------------------------------------------------------------

type (
	tickMsg time.Time

	// rowsMsg carries board rows from the store.
	rowsMsg struct {
		rows []store.Row
		err  error
	}

	// detailMsg carries a problem's rendered statement.
	detailMsg struct {
		slug     string
		seq      int
		detail   *store.Detail
		markdown string
		images   []render.Image
		err      error
	}

	// detailFetchMsg drives the debounced network fetch for a statement that is not
	// cached. ready is false on the first hop (schedule the timer) and true on the
	// second (actually fetch).
	detailFetchMsg struct {
		slug  string
		seq   int
		ready bool
	}

	// editorialMsg carries a problem's rendered official solution. A gated editorial
	// arrives with editorial set and err ErrPremiumRequired — the pane needs both.
	editorialMsg struct {
		slug      string
		editorial *store.Editorial
		markdown  string
		images    []render.Image
		err       error
	}

	// companiesMsg carries the company registry from the store.
	companiesMsg struct {
		companies []store.Company
		err       error
	}

	// packCountsMsg carries how much of each of a company's timeframes is stored.
	packCountsMsg struct {
		company string
		counts  map[leetcode.Timeframe]int
		err     error
	}

	// browserImportMsg carries the result of reading cookies out of a browser.
	browserImportMsg struct {
		from  auth.Browser
		creds auth.Credentials
		err   error
	}

	// editReadyMsg means the workspace is laid out and the editor can be launched.
	// The plan carries where it opens — a pane, its own window, or this terminal.
	editReadyMsg struct{ plan editPlan }

	// editDoneMsg arrives when an editor that took over the terminal exits.
	editDoneMsg struct{ err error }

	// runFinishedMsg carries a local run's result.
	runFinishedMsg struct {
		slug   string
		result runner.Result
		err    error
	}

	// judgeMsg carries a submission's outcome.
	judgeMsg struct {
		flapID    int
		judgement leetcode.Judgement
		err       error
	}

	// syncProgressMsg is one update from the sync worker.
	syncProgressMsg syncer.Progress

	// accountMsg carries the signed-in user's status.
	accountMsg struct {
		status leetcode.UserStatus
		err    error
	}

	// statusMsg sets the transient status line.
	statusMsg struct {
		text    string
		isError bool
	}

	// clearStatusMsg expires a transient status line.
	clearStatusMsg struct{ id int }
)
