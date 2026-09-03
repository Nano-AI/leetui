package theme

// The characters the frame is drawn with.
//
// D-023 gave the state column an ASCII fallback and left the frame in box-drawing, which
// made ASCII mode half a promise: a terminal that cannot render `✓` cannot render `╭─┼─╯`
// either, and a bezel of question marks is worse than no bezel. This is the other half.
//
// Chosen once at startup from the same theme.ASCII the glyphs use, so a terminal is never
// asked to draw one set and not the other.
//
// The ASCII set is deliberately the plainest thing that still reads as a box. `+---+` is
// what every terminal on earth has drawn since VT100, and it is unambiguous in a way a
// cleverer approximation would not be.

// Charset is every non-ASCII character the interface draws.
type Charset struct {
	// Frame bezel.
	CornerTL, CornerTR string
	CornerBL, CornerBR string
	EdgeH, EdgeV       string

	// Column rules inside a frame, and where they meet the bezel.
	SepV     string
	SepTeeT  string // ┬ top
	SepTeeB  string // ┴ bottom
	SepCross string // ┼ header crossing
	SepLeft  string // ├
	SepRight string // ┤

	// DashRule is the lighter divider drawn inside a pane, distinct from the bezel so
	// the two do not read as the same line.
	DashRule string

	// Bullet separates inline items — tags, key hints, a branch and its state.
	Bullet string

	// Dot is the smaller separator used inside one phrase.
	Dot string

	// Cursor is the bar marking the selected row. It costs no width and disturbs no
	// alignment, which is the whole reason it is a half-block and not a caret.
	Cursor string

	// Check and Cross mark a form field as filled or still empty.
	//
	// They carry real information in the sign-in panel, where both fields are masked
	// and the marker is the only thing that answers "did that paste land". So they get
	// an ASCII fallback like everything else here: a terminal that cannot draw them
	// would otherwise show the user two question marks in place of the one signal.
	Check, Cross string
}

var (
	unicodeChars = Charset{
		CornerTL: "╭", CornerTR: "╮",
		CornerBL: "╰", CornerBR: "╯",
		EdgeH: "─", EdgeV: "│",

		SepV:     "│",
		SepTeeT:  "┬",
		SepTeeB:  "┴",
		SepCross: "┼",
		SepLeft:  "├",
		SepRight: "┤",

		DashRule: "╌",
		Bullet:   "┊",
		Dot:      "·",
		Cursor:   "▌",
		Check:    "✓",
		Cross:    "✗",
	}

	// asciiChars survives anything, including a non-UTF-8 locale.
	//
	// Every joint is "+". Distinguishing ┬ from ┼ from ┴ in ASCII would mean inventing a
	// convention the reader has to learn, and the grid's structure is already carried by
	// alignment — the joints only have to not look broken.
	asciiChars = Charset{
		CornerTL: "+", CornerTR: "+",
		CornerBL: "+", CornerBR: "+",
		EdgeH: "-", EdgeV: "|",

		SepV:     "|",
		SepTeeT:  "+",
		SepTeeB:  "+",
		SepCross: "+",
		SepLeft:  "+",
		SepRight: "+",

		DashRule: "-",
		Bullet:   "|",
		// "*" rather than "." — a period between two words reads as the end of a
		// sentence, and these separate items inside one.
		Dot:    "*",
		Cursor: ">",
		// "*" and "-" rather than "y"/"n" or "!": one column each, unambiguous at a
		// glance, and the empty state reads as "nothing here yet" rather than as a
		// warning. An unfilled field is a step not reached, not a mistake.
		Check: "*",
		Cross: "-",
	}
)

// Chars returns the set this terminal can draw.
func Chars() Charset {
	if ASCII {
		return asciiChars
	}
	return unicodeChars
}
