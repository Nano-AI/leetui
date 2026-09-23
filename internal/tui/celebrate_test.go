package tui

import (
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/config"
	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// TestTierThresholds pins the grading, including the case the user asked for by name:
// beating half on BOTH axes is the one that earns DOUBLE 50.
func TestTierThresholds(t *testing.T) {
	for _, tc := range []struct {
		name    string
		rt, mem float64
		want    celebrationTier
		badge   string
	}{
		{"no figures at all", 0, 0, tierPass, ""},
		{"fast but heavy", 94, 12, tierFast, "✦ FAST"},
		{"lean but slow", 20, 88, tierFast, "✦ FAST"},
		{"both over half", 62, 71, tierDouble, "✦ DOUBLE 50"},
		{"exactly half is not over it", 50, 50, tierPass, ""},
		{"both over ninety", 94, 91, tierElite, "✦ TOP 10"},
		{"below on both", 20, 30, tierPass, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tierFor(tc.rt, tc.mem)
			if got != tc.want {
				t.Errorf("tierFor(%v, %v) = %v, want %v", tc.rt, tc.mem, got, tc.want)
			}
			if got.Badge() != tc.badge {
				t.Errorf("badge = %q, want %q", got.Badge(), tc.badge)
			}
		})
	}
}

// TestMissingPercentileIsNotZeroPercent: LeetCode omits percentiles for some problems and
// languages. An absent figure must never read as "beats nobody".
func TestMissingPercentileIsNotZeroPercent(t *testing.T) {
	// Only runtime reported, and it is excellent. The missing memory figure must not drag
	// the result below tierFast.
	if got := tierFor(97, 0); got != tierFast {
		t.Errorf("tierFor(97, missing) = %v, want tierFast", got)
	}

	// And the stats line must not print a memory percentile that was never reported.
	q := queueItem{Verdict: theme.Accepted, Runtime: "58 ms", Memory: "20.2 MB", Percentile: 97}
	if strings.Contains(q.stats(), "beats 0%") {
		t.Errorf("a missing memory percentile rendered as a figure: %q", q.stats())
	}
	q.MemoryPct = 71
	if !strings.Contains(q.stats(), "beats 71%") {
		t.Errorf("a real memory percentile is missing: %q", q.stats())
	}
}

// TestCelebrateLevelResolves covers the setting, including the empty string an existing
// config file written before this option existed will have.
func TestCelebrateLevelResolves(t *testing.T) {
	for _, tc := range []struct {
		set    string
		motion bool
		want   string
	}{
		{"", false, config.CelebrateFull},
		{"nonsense", false, config.CelebrateFull},
		{config.CelebrateOff, false, config.CelebrateOff},
		{config.CelebrateSubtle, false, config.CelebrateSubtle},
		{config.CelebrateFull, false, config.CelebrateFull},

		// Reduced motion outranks full, and demotes it to subtle rather than off: asking
		// for no animation is not asking to stop being told you did well.
		{config.CelebrateFull, true, config.CelebrateSubtle},
		{config.CelebrateOff, true, config.CelebrateOff},
		{config.CelebrateSubtle, true, config.CelebrateSubtle},
	} {
		cfg := config.Default()
		cfg.UI.Celebrate = tc.set
		cfg.UI.ReduceMotion = tc.motion
		if got := cfg.CelebrateLevel(); got != tc.want {
			t.Errorf("celebrate=%q motion=%v resolved to %q, want %q",
				tc.set, tc.motion, got, tc.want)
		}
	}
}

// TestCelebrationRespectsTheLevel: `off` and `subtle` must schedule no frames at all,
// which is what makes them genuinely cheaper rather than merely quieter.
func TestCelebrationRespectsTheLevel(t *testing.T) {
	for _, level := range []string{config.CelebrateOff, config.CelebrateSubtle} {
		m := boot(t, true, 120, 32)
		m.cfg.UI.Celebrate = level
		if cmd := m.beginCelebration(1, tierDouble); cmd != nil {
			t.Errorf("celebrate=%s scheduled an animation", level)
		}
		if m.celebrate.active() {
			t.Errorf("celebrate=%s started a sweep", level)
		}
	}

	m := boot(t, true, 120, 32)
	m.cfg.UI.Celebrate = config.CelebrateFull
	if cmd := m.beginCelebration(1, tierDouble); cmd == nil {
		t.Error("celebrate=full scheduled nothing")
	}
	if !m.celebrate.active() {
		t.Error("celebrate=full did not start a sweep")
	}
}

// TestReduceMotionSilencesTheSweep: the flip's own switch governs all motion, or a
// screen that asked for none gets some anyway.
func TestReduceMotionSilencesTheSweep(t *testing.T) {
	components.ReduceMotion = true
	t.Cleanup(func() { components.ReduceMotion = false })

	m := boot(t, true, 120, 32)
	m.cfg.UI.Celebrate = config.CelebrateFull
	if cmd := m.beginCelebration(1, tierElite); cmd != nil {
		t.Error("reduce_motion still scheduled an animation")
	}
	if m.celebrate.active() {
		t.Error("reduce_motion still started a sweep")
	}
}

// TestBadgeSurvivesWithoutMotion: the badge is information, not decoration, so it shows
// at subtle. Only `off` withholds it.
func TestBadgeSurvivesWithoutMotion(t *testing.T) {
	m := boot(t, true, 120, 32)

	m.cfg.UI.Celebrate = config.CelebrateSubtle
	if got := m.celebrationBadge(tierDouble); !strings.Contains(got, "DOUBLE 50") {
		t.Errorf("subtle withheld the badge: %q", got)
	}

	m.cfg.UI.Celebrate = config.CelebrateOff
	if got := m.celebrationBadge(tierDouble); got != "" {
		t.Errorf("off still rendered a badge: %q", got)
	}

	// A plain pass has nothing to boast about and gets no badge at any level.
	m.cfg.UI.Celebrate = config.CelebrateFull
	if got := m.celebrationBadge(tierPass); got != "" {
		t.Errorf("a plain pass got a badge: %q", got)
	}
}

// TestSweepAdvancesAndSettles walks the animation to its end and checks it hands the
// verdict back rather than leaving colour on the row.
func TestSweepAdvancesAndSettles(t *testing.T) {
	m := boot(t, true, 120, 32)
	m.cfg.UI.Celebrate = config.CelebrateFull
	m.beginCelebration(7, tierPass)

	total := tierPass.frames()
	if m.celebrate.total != total {
		t.Fatalf("sweep length is %d, want %d", m.celebrate.total, total)
	}

	var model interface{} = m
	for i := 0; i < total; i++ {
		mm := model.(Model)
		if !mm.celebrate.active() {
			t.Fatalf("the sweep stopped early, at frame %d of %d", i, total)
		}
		next, _ := mm.handleCelebrateTick(celebrateTickMsg{flapID: 7, frame: i})
		model = next
	}
	if final := model.(Model); final.celebrate.active() {
		t.Error("the sweep never settled")
	}
}

// TestStaleTickIsIgnored: a second submission landing mid-sweep must not leave colour
// running on the wrong row.
func TestStaleTickIsIgnored(t *testing.T) {
	m := boot(t, true, 120, 32)
	m.cfg.UI.Celebrate = config.CelebrateFull
	m.beginCelebration(7, tierElite)

	next, cmd := m.handleCelebrateTick(celebrateTickMsg{flapID: 99, frame: 3})
	if cmd != nil {
		t.Error("a tick for another submission scheduled another frame")
	}
	if got := next.(Model).celebrate.frame; got != 0 {
		t.Errorf("a stale tick advanced the sweep to frame %d", got)
	}
}

// TestRainbowColoursTextAndMovesIt: the band has to travel, or it is a flashing word
// rather than a sweep.
func TestRainbowColoursTextAndMovesIt(t *testing.T) {
	first := rainbow("ACCEPTED", 0)
	second := rainbow("ACCEPTED", 1)

	if first == second {
		t.Error("the sweep renders identically on consecutive frames; the band is not moving")
	}
	// The letters survive the colouring.
	for _, r := range "ACCEPTED" {
		if !strings.ContainsRune(first, r) {
			t.Fatalf("the sweep dropped %q from the verdict", r)
		}
	}
	// A full cycle of the palette returns to where it started.
	if rainbow("ACCEPTED", 0) != rainbow("ACCEPTED", len(sweepPalette)) {
		t.Error("the palette does not cycle cleanly")
	}
}
