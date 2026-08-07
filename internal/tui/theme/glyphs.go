package theme

// Glyphs, and the terminals that cannot draw them.
//
// The board's state column is a glyph rather than a word because a column of
// `✓ ✓ ◐ ⊘ ✓` scans down in a way `SOLVED SOLVED TRIED LOCKED SOLVED` does not, and it
// costs two fewer columns. That trade is only sound because the column carries a header —
// a glyph is allowed to be a glyph when something else names it (DESIGN.md).
//
// The risk is width. `✓` (U+2713) and `⊘` (U+2298) are East-Asian Ambiguous: a terminal
// is free to draw them two cells wide, and in a grid ruled to the column that shears
// every row below. So there is an ASCII set, and ASCII is set from main when the
// environment says the terminal cannot be trusted with more.

// ASCII replaces every non-ASCII glyph with a plain equivalent.
//
// Package-level rather than threaded through, matching components.ReduceMotion: it is
// decided once at startup from the config and the environment, and never changes while
// the program runs.
var ASCII bool

// GlyphSet is the marks the board draws.
type GlyphSet struct {
	Solved string
	Tried  string
	Locked string
	Todo   string
}

var (
	unicodeGlyphs = GlyphSet{
		Solved: "✓", // U+2713
		Tried:  "◐", // U+25D0 — half filled, for partly done
		Locked: "⊘", // U+2298 — only ever drawn without premium
		Todo:   "●", // U+25CF
	}

	// asciiGlyphs are chosen to survive anything, including a non-UTF-8 locale.
	//
	// "~" for tried reads as "approximately there", which is the same idea the half-filled
	// circle carries.
	asciiGlyphs = GlyphSet{
		Solved: "x",
		Tried:  "~",
		Locked: "-",
		Todo:   "*",
	}
)

// Glyphs returns the set this terminal can draw.
func Glyphs() GlyphSet {
	if ASCII {
		return asciiGlyphs
	}
	return unicodeGlyphs
}

// Center pads a single-cell glyph to width w with the glyph in the middle.
//
// Centred rather than left-aligned because the column is read as a stack: a mark hugging
// the left rule sits directly beside the acceptance figure and reads as part of it.
func Center(glyph string, w int) string {
	if glyph == "" || w <= 1 {
		return glyph
	}
	left := (w - 1) / 2
	out := make([]byte, 0, w+len(glyph))
	for i := 0; i < left; i++ {
		out = append(out, ' ')
	}
	return string(out) + glyph
}
