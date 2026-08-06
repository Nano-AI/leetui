package tui

import (
	"time"

	"github.com/grootbeat/leetui/internal/auth"
	"github.com/grootbeat/leetui/internal/leetcode"
	"github.com/grootbeat/leetui/internal/render"
	"github.com/grootbeat/leetui/internal/store"
	"github.com/grootbeat/leetui/internal/syncer"
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

	// browserImportMsg carries the result of reading cookies out of a browser.
	browserImportMsg struct {
		from  auth.Browser
		creds auth.Credentials
		err   error
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
