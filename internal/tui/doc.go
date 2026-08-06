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
//	model.go        the Model struct, New, Init
//	messages.go     every tea.Msg this package defines
//	update.go       Update dispatch; update_sync.go handles every background job,
//	                update_solve.go the run and submit loop
//	keys.go         the main keymap; keys_modal.go covers search and sign-in,
//	                keys_filter.go the board's filter cycles
//	cursor.go       cursor movement, scrolling, difficulty filters
//	commands.go     store and network commands; commands_sync.go starts the jobs
//	panes.go        pane and mode enums, layout breakpoints
//	solve.go        edit, run, submit; solve_run.go and solve_actions.go
//	solve_files.go  the two earned exceptions to never-overwrite (D-010a, D-014)
//	solve_edit.go   where the editor opens — pane, own window, or this terminal
//	                (D-012a); update_edit.go launches it
//	watch.go        the solution-file poll
//	picker.go       the selection lists — language, editor, company timeframe
//	company.go      company packs (D-006); company_keys.go drives the browser
//	editorial.go    the editorial pane's state and fetch
//
// File map — rendering:
//
//	layout.go       height and width arithmetic, View, viewBody
//	rail.go         the header strip
//	board.go        the problem grid; board_row.go renders one row
//	detail.go       the statement pane; editorial_view.go its editorial half
//	workbench.go    the solution's path and whether saves re-run (D-016)
//	result.go       the local-run diff panel
//	queue.go        the submission queue; judge_text.go wording for verdicts
//	search.go       the search panel
//	setup.go        the first-run screen
//	help.go         the key reference, folding to columns when it must
//	signin.go       the sign-in panel
//	company_view.go the company browser; picker_view.go the selection lists
//	chrome.go       status line, hints, cell sizing
//	format.go       small shared formatting helpers
package tui
