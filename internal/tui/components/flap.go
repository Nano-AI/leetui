// Package components holds the reusable widgets of the Departure Board.
package components

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// ReduceMotion disables the flip animation app-wide. Set from config `reduce_motion`,
// the --no-motion flag, or a prefers-reduced-motion signal.
//
// When true, FlipTo settles instantly on its final frame. The app must stay fully
// legible with motion off — the flip is never the only channel carrying information.
var ReduceMotion bool

// --- The signature -----------------------------------------------------------
//
// A Flap is a single cell of the departure board. When its state resolves it does not
// repaint — it flips, shearing through the half-block ramp before settling.
//
// This is the app's entire motion budget and its only loading state. There are no
// spinners in leetui: a pending job renders as a flap held mid-flip. If a new animation
// seems necessary somewhere, the answer is that the flip should cover it.
//
// See docs/DESIGN.md § Signature.

// flipFrames are the shear glyphs the cell passes through, in order. The final frame is
// the settled text, so the sequence is len(flipFrames) ticks of animation.
var flipFrames = []string{"▔", "▀", "▚", "▄", "▁"}

// FlipInterval is the per-frame duration. Six frames at 40ms reads as a physical flip
// without ever feeling like a delay the user is waiting on.
const FlipInterval = 40 * time.Millisecond

// FlipTickMsg advances one flap's animation. It carries the flap's ID so a board full of
// flaps can share the message type without cross-talk.
type FlipTickMsg struct {
	ID    int
	Frame int
}

// Flap is one cell of the board.
//
// The zero value is a settled, empty flap. Callers own the ID and must keep it unique
// within a program.
type Flap struct {
	ID int

	text  string         // settled content
	color lipgloss.Color // settled content's color
	frame int            // index into flipFrames; -1 means settled

	// pending holds what we are flipping toward while frame >= 0.
	pendingText  string
	pendingColor lipgloss.Color
}

// NewFlap returns a settled flap showing text.
func NewFlap(id int, text string, c lipgloss.Color) Flap {
	return Flap{ID: id, text: text, color: c, frame: -1}
}

// FlipTo animates the flap to new content and returns the command that drives it.
//
// Calling FlipTo while a flip is in progress restarts the animation toward the new
// target — correct for a queue row that goes PENDING -> JUDGING -> ACCEPTED faster than
// the animation completes.
//
// Returns a nil command when motion is reduced or the content is unchanged; callers can
// pass a nil tea.Cmd along safely.
func (f *Flap) FlipTo(text string, c lipgloss.Color) tea.Cmd {
	if ReduceMotion {
		f.text, f.color, f.frame = text, c, -1
		return nil
	}
	if text == f.text && f.frame < 0 {
		return nil
	}
	f.pendingText, f.pendingColor = text, c
	f.frame = 0
	return flipTick(f.ID, 0)
}

// Set changes the flap's content with no animation. Use for initial population and for
// content that did not "resolve" (a re-render, a resize) — flipping on every repaint
// would spend the motion budget on nothing.
func (f *Flap) Set(text string, c lipgloss.Color) {
	f.text, f.color, f.frame = text, c, -1
}

// Update advances the animation. Messages for other flap IDs are ignored, so every flap
// on screen can be fed the same message stream.
func (f Flap) Update(msg tea.Msg) (Flap, tea.Cmd) {
	tick, ok := msg.(FlipTickMsg)
	if !ok || tick.ID != f.ID || f.frame < 0 {
		return f, nil
	}
	// Stale tick from a superseded animation.
	if tick.Frame != f.frame {
		return f, nil
	}

	f.frame++
	if f.frame >= len(flipFrames) {
		// Settle.
		f.text, f.color, f.frame = f.pendingText, f.pendingColor, -1
		return f, nil
	}
	return f, flipTick(f.ID, f.frame)
}

// Flipping reports whether an animation is in progress.
func (f Flap) Flipping() bool { return f.frame >= 0 }

// Text returns the settled content, ignoring any in-progress flip.
func (f Flap) Text() string { return f.text }

// View renders the flap padded to width.
//
// Mid-flip the cell is amber rather than its target color: the system is still speaking,
// because the judge has not. The verdict's own color only appears once it settles.
func (f Flap) View(width int) string {
	if width < 1 {
		return ""
	}
	if f.frame < 0 {
		return lipgloss.NewStyle().
			Foreground(f.color).
			Bold(true).
			Width(width).
			MaxWidth(width).
			Render(f.text)
	}

	shear := repeat(flipFrames[f.frame], width)
	return lipgloss.NewStyle().
		Foreground(theme.Amber).
		Width(width).
		MaxWidth(width).
		Render(shear)
}

func flipTick(id, frame int) tea.Cmd {
	return tea.Tick(FlipInterval, func(time.Time) tea.Msg {
		return FlipTickMsg{ID: id, Frame: frame}
	})
}

func repeat(s string, n int) string {
	if n <= 0 {
		return ""
	}
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
