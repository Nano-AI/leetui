package theme

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Colour, and terminals that have less of it than this palette assumes.
//
// The Departure Board is specified in truecolor hex. A 256-colour terminal quantises
// those to its nearest cube entry, which is fine — amber stays amber. Below that it stops
// being fine: on a 16-colour terminal the six system tokens collapse toward two or three
// indistinguishable greys, and `dim` metadata can land on the same colour as `bone`
// content. At that point colour is carrying no information and is actively misleading.
//
// So the rule is the one the design already states for motion: nothing may be encoded in
// colour alone. Under NO_COLOR the letterspaced Display treatment still separates a
// verdict from body text, the STATE column still has a header, and every hint still says
// what it does in words.

// Profile is how much colour this terminal has.
type Profile int

const (
	// TrueColor is 24-bit, the palette as specified.
	TrueColor Profile = iota
	// ANSI256 quantises to the 256-colour cube. Close enough that nothing is lost.
	ANSI256
	// Basic is 16 colours. The palette does not survive; see Degraded.
	Basic
	// NoColor is monochrome, by capability or by NO_COLOR.
	NoColor
)

// DetectProfile asks the terminal, honouring NO_COLOR.
//
// NO_COLOR is checked by lipgloss/termenv already, but asking here as well keeps the
// answer in one place — and the app branches on it for more than styling.
func DetectProfile() Profile {
	switch lipgloss.ColorProfile() {
	case termenv.TrueColor:
		return TrueColor
	case termenv.ANSI256:
		return ANSI256
	case termenv.ANSI:
		return Basic
	default:
		return NoColor
	}
}

// Degraded reports whether colour has stopped carrying information.
//
// True for 16-colour and monochrome. Callers use it to add a channel that is not colour:
// the difficulty tag gains a marker, the verdict keeps its letterspacing, and a passing
// case says PASS rather than relying on green.
func (p Profile) Degraded() bool { return p == Basic || p == NoColor }

// Active is the profile this run is using, decided once at startup alongside ASCII.
//
// Package-level for the same reason ASCII is: it is read from a dozen render paths, it
// never changes while the program runs, and threading it through every signature would
// add a parameter to functions that have nothing else to say about colour.
var Active = TrueColor

// FocusMark is how the focused pane is shown when colour cannot show it.
//
// Focus is normally the bezel turning amber, and DESIGN.md is firm that it is never a
// border-style change. That rule assumes colour exists. On a monochrome terminal it
// leaves no indicator at all, and "which pane does tab move" becomes unanswerable — so
// monochrome is the one stated exception, and the focused pane's title gets a marker.
//
// Empty at full colour, where the amber bezel already says it and a second marker would
// be the same fact twice.
func FocusMark() string {
	if !Active.Degraded() {
		return ""
	}
	return Chars().Cursor + " "
}
