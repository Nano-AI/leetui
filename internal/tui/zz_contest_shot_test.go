package tui

import (
	"fmt"
	"testing"
)

// TestZZContestShot prints the contest browser for visual inspection, the same trick
// zz_docs_test.go uses for the reference screen. No assertions: it exists so a change to
// the layout is visible in `go test -v` output rather than only in a terminal.
func TestZZContestShot(t *testing.T) {
	m := boot(t, true, 100, 30)
	seedContestSchedule(t, m)
	m = drive(t, m, key("C"))
	m = drive(t, m, m.loadContests()())
	fmt.Println(stripANSI(m.View()))
}
