package config

// DefaultKeymap maps action name to its default binding (D-013).
//
// Bindings are declared here and nowhere else. A key compared literally at a call site
// cannot be remapped and will not appear in the help overlay — that is a bug, not a
// shortcut. Arrows and PgUp/PgDn are handled as an always-on fallback in the TUI and
// are deliberately not listed, because they are not remappable.
var DefaultKeymap = map[string]string{
	"up":          "k",
	"down":        "j",
	"half_up":     "ctrl+u",
	"half_down":   "ctrl+d",
	"page_up":     "ctrl+b",
	"page_down":   "ctrl+f",
	"top":         "g",
	"bottom":      "G",
	"open":        "enter",
	"back":        "esc",
	"focus_next":  "tab",
	"focus_prev":  "shift+tab",
	"search":      "/",
	"palette":     ":",
	"next_match":  "n",
	"prev_match":  "N",
	"edit":        "e",
	"run":         "r",
	"lang":        "l",
	"editor":      "E",
	"submit":      "s",
	"sync":        "S",
	"editorial":   "d",
	"companies":   "c",
	"open_web":    "o",
	"timer":       "t",
	"timer_reset": "T",
	"auth":        "a",
	"help":        "?",
	"quit":        "q",
}
