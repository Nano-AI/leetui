// Package tui is the Bubbletea application.
//
// Architectural rule (docs/ARCHITECTURE.md): Update NEVER blocks. Every network call,
// subprocess, and disk read returns a tea.Cmd and comes back as a tea.Msg. That is what
// keeps the flip smooth while a submission is in flight and search responsive while a
// sync runs.
//
// Second rule: views read the STORE, not the network. Browsing and searching hit SQLite
// only; the network populates SQLite through explicit sync jobs. A keystroke must never
// become an HTTP request.
//
// File map — state and behaviour:
//
//	model.go     the Model struct, New, Init
//	messages.go  every tea.Msg this package defines
//	update.go    Update dispatch and sync progress
//	keys.go      the main keymap; keys_modal.go covers search and sign-in
//	cursor.go    cursor movement, scrolling, filters
//	commands.go  store and network commands; commands_sync.go covers the sync job
//	panes.go     pane and mode enums, layout breakpoints
//
// File map — rendering:
//
//	layout.go    height and width arithmetic, View, viewBody
//	rail.go      the header strip
//	board.go     the problem grid; board_row.go renders one row
//	detail.go    the statement pane
//	queue.go     the submission queue
//	search.go    the search panel
//	setup.go     the first-run screen
//	help.go      the key reference
//	signin.go    the sign-in panel
//	chrome.go    status line, hints, cell sizing
//	format.go    small shared formatting helpers
package tui
