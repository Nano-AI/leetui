package tui

import (
	"strings"
	"testing"
	"time"
)

// The rail's stopwatch mirrors LeetCode's own (D-006). It is a plain timer, not
// assessment scoring — mock assessments are explicitly out of scope.
func TestTimerRunsAndResets(t *testing.T) {
	m := boot(t, true, 120, 32)

	if strings.Contains(m.viewRail(), "00:00:00") {
		t.Error("a stopped timer at zero should read as dashes, not as a running clock")
	}

	m = drive(t, m, key("t"))
	if !m.timerRunning {
		t.Fatal("t did not start the timer")
	}

	// Few ticks on purpose: each one reschedules the clock, and the harness waits out
	// every command it cannot resolve. The formatting is checked separately below.
	for i := 0; i < 5; i++ {
		m = drive(t, m, tickMsg(time.Now()))
	}
	if m.elapsed != 5*time.Second {
		t.Fatalf("elapsed is %v after 5 ticks, want 5s", m.elapsed)
	}

	m.elapsed = 65 * time.Second
	if !strings.Contains(m.viewRail(), "00:01:05") {
		t.Errorf("the rail does not show the elapsed time:\n%s", m.viewRail())
	}

	// Pausing must hold the reading, not zero it.
	m = drive(t, m, key("t"))
	m = drive(t, m, tickMsg(time.Now()))
	if m.timerRunning {
		t.Error("t did not stop the timer")
	}
	if m.elapsed != 65*time.Second {
		t.Errorf("a paused timer advanced to %v", m.elapsed)
	}

	m = drive(t, m, key("T"))
	if m.timerRunning || m.elapsed != 0 {
		t.Errorf("T left the timer at %v running=%v, want 0 and stopped", m.elapsed, m.timerRunning)
	}
}
