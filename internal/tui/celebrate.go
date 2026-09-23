package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Nano-AI/leetui/internal/config"
	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// ---------------------------------------------------------------------------
// Celebrating an Accepted verdict (D-033)
// ---------------------------------------------------------------------------
//
// The one place leetui is allowed to be pleased with you.
//
// It is a deliberate exception to the rule in components/flap.go that the flip is the
// app's entire motion budget, and it earns the exception by being rare, brief, and
// self-cancelling: it fires only on Accepted, runs about a second, and settles into
// exactly the frame that would have been there anyway. Nothing is ever only conveyed by
// the animation — the verdict, the badge and the figures are all still legible with
// `ui.celebrate off`.

// celebrationTier is how much there is to be pleased about.
//
// The thresholds read the judge's own percentiles rather than anything leetui measures,
// so the badge means the same thing the website means.
type celebrationTier int

const (
	// tierNone is not celebrating.
	tierNone celebrationTier = iota

	// tierPass is Accepted, with nothing notable about the figures — or with no figures
	// at all, which is what a judge that returned no percentiles looks like.
	tierPass

	// tierFast beats half the submissions on ONE of runtime or memory.
	tierFast

	// tierDouble beats half on BOTH. The one the user actually asked for.
	tierDouble

	// tierElite beats ninety percent on both.
	tierElite
)

// Badge is the short tag shown beside the verdict, or "" for a plain pass.
//
// It names the achievement rather than grading it: "DOUBLE 50" says what happened, where
// a star rating would need a legend nobody has (the same argument that made the board's
// acceptance column a percentage rather than a sparkline, D-020).
func (t celebrationTier) Badge() string {
	switch t {
	case tierFast:
		return "✦ FAST"
	case tierDouble:
		return "✦ DOUBLE 50"
	case tierElite:
		return "✦ TOP 10"
	default:
		return ""
	}
}

// frames is how long the sweep runs for this tier. A better result is celebrated longer,
// which is the cheapest possible way to make the rare thing feel rare.
func (t celebrationTier) frames() int {
	switch t {
	case tierFast:
		return 18
	case tierDouble:
		return 26
	case tierElite:
		return 34
	case tierPass:
		return 12
	default:
		return 0
	}
}

// tierFor grades a submission from the judge's percentiles.
//
// A zero percentile means "not reported", not "beats nobody" — LeetCode omits them for
// some problems and languages — so a missing figure can never demote a result below
// tierPass, and both being missing is a plain pass rather than a failure to impress.
func tierFor(runtimePct, memoryPct float64) celebrationTier {
	switch {
	case runtimePct >= 90 && memoryPct >= 90:
		return tierElite
	case runtimePct > 50 && memoryPct > 50:
		return tierDouble
	case runtimePct > 50 || memoryPct > 50:
		return tierFast
	default:
		return tierPass
	}
}

// celebration is the running animation. The zero value is not celebrating.
type celebration struct {
	tier celebrationTier

	// flapID ties the sweep to one submission, so a second verdict landing mid-animation
	// cannot leave the colour running on the wrong row.
	flapID int

	frame int
	total int
}

// active reports whether a sweep is on screen.
func (c celebration) active() bool { return c.tier != tierNone && c.frame < c.total }

// celebrateTickMsg advances the sweep.
type celebrateTickMsg struct {
	flapID int
	frame  int
}

// celebrateInterval is a touch slower than the flip. The flip is a mechanism resolving;
// this is a flourish, and at 40ms it read as a flicker rather than a sweep.
const celebrateInterval = 55 * time.Millisecond

// beginCelebration starts the sweep for a submission, or returns nil when there is
// nothing to run.
//
// Returns nil for `off`, and for `subtle` — at that level the badge and the figures do
// the work and no frames are ever scheduled, which is what makes it genuinely cheaper
// rather than merely quieter.
func (m *Model) beginCelebration(flapID int, tier celebrationTier) tea.Cmd {
	if tier == tierNone || m.cfg.CelebrateLevel() != config.CelebrateFull {
		return nil
	}
	// The flip is the app's motion budget and its own switch governs it; a celebration
	// that ignored it would put motion back on a screen that asked for none.
	if components.ReduceMotion {
		return nil
	}

	m.celebrate = celebration{tier: tier, flapID: flapID, total: tier.frames()}
	return celebrateTick(flapID, 0)
}

func celebrateTick(flapID, frame int) tea.Cmd {
	return tea.Tick(celebrateInterval, func(time.Time) tea.Msg {
		return celebrateTickMsg{flapID: flapID, frame: frame}
	})
}

// handleCelebrateTick advances one frame, or lets the sweep settle.
func (m Model) handleCelebrateTick(msg celebrateTickMsg) (tea.Model, tea.Cmd) {
	// Stale: another submission started celebrating while this was in flight.
	if msg.flapID != m.celebrate.flapID || !m.celebrate.active() {
		return m, nil
	}
	m.celebrate.frame = msg.frame + 1
	if !m.celebrate.active() {
		// Settled. Clearing the tier is what hands the verdict back to the flap's own
		// colour, so the final frame is the one that would have been there anyway.
		m.celebrate = celebration{}
		return m, nil
	}
	return m, celebrateTick(msg.flapID, m.celebrate.frame)
}

// sweepPalette is the hue cycle the verdict travels through.
//
// It ends on AC green so the last frame before settling already matches the colour the
// flap is about to hold — the animation resolves into the calm state rather than
// snapping back to it.
var sweepPalette = []lipgloss.Color{
	theme.WA,                  // #D65A5A
	lipgloss.Color("#E8A33D"), // amber, the system's own voice
	lipgloss.Color("#D8D14A"), // straw
	theme.AC,                  // #4FB477
	lipgloss.Color("#3FB6A8"), // teal
	lipgloss.Color("#5AA8E0"), // sky
	lipgloss.Color("#8E7FE8"), // violet
	lipgloss.Color("#C77FD4"), // orchid
}

// rainbow renders text with the palette travelling across it.
//
// Each character takes its colour from its position plus the frame, so the band moves
// left along the word rather than the whole word blinking. Spaces are emitted unstyled:
// colouring them costs escape codes for nothing, and letterspaced display text is mostly
// spaces.
func rainbow(text string, frame int) string {
	out := make([]byte, 0, len(text)*20)

	// Count VISIBLE characters, not rune positions. Verdicts are letterspaced by
	// theme.Display, so indexing by position would step the palette by two and drop half
	// of it — the gradient came out coarse and stripey until this counted letters.
	visible := 0
	for _, r := range text {
		if r == ' ' {
			out = append(out, ' ')
			continue
		}
		// Subtracting the frame moves the band forward along the text; adding it would
		// send the colours backwards, which reads as the word sliding the wrong way.
		idx := ((visible-frame)%len(sweepPalette) + len(sweepPalette)) % len(sweepPalette)
		style := lipgloss.NewStyle().Foreground(sweepPalette[idx]).Bold(true)
		out = append(out, []byte(style.Render(string(r)))...)
		visible++
	}
	return string(out)
}

// celebrationBadge renders a tier tag, or "" when the level says not to.
//
// Shown at `subtle` as well as `full`: the badge is information — it is the judge's
// percentiles said in two words — and only the motion is decoration.
func (m Model) celebrationBadge(tier celebrationTier) string {
	if tier == tierNone || m.cfg.CelebrateLevel() == config.CelebrateOff {
		return ""
	}
	badge := tier.Badge()
	if badge == "" {
		return ""
	}
	return lipgloss.NewStyle().Foreground(theme.Amber).Bold(true).Render(badge)
}
