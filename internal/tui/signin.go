package tui

import (
	"fmt"
	"strings"

	"github.com/Nano-AI/leetui/internal/auth"
	"github.com/Nano-AI/leetui/internal/tui/components"
	"github.com/Nano-AI/leetui/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// authInnerWidth is the sign-in frame's usable width.
//
// Wider than the other panels on purpose. A LeetCode session JWT runs past 800
// characters, and revealing one inside a 60-column frame shows a window so narrow that
// checking it against the browser is still guesswork. This does not make the whole value
// visible (nothing at terminal width would) but it is the difference between reading a
// recognisable span of it and reading eight characters.
func authInnerWidth(width int) int {
	return minInt(maxInt(width-8, 44), 76)
}

// authFieldWidth sizes an input to the frame that draws it.
//
// It has to be derived from the panel and not from the terminal: the fields used to be
// sized to the full window, so on a wide terminal the input believed it had 96 columns,
// scrolled its contents to fit them, and the frame then clipped the overflow. Revealing
// a cookie showed a value cut off mid-token, which is indistinguishable from a truncated
// paste at exactly the moment the user is checking for one.
func authFieldWidth(width int) int {
	return maxInt(authInnerWidth(width)-4, 20)
}

// authMaxBrowsers caps the browser list. Nine is the ceiling the digit shortcuts impose,
// and a machine with ten browser profiles is better served by the fields below anyway.
const authMaxBrowsers = 9

// authErrLines bounds how much of an error the panel will show.
//
// The old panel truncated errors to a single line, which is how a Linux keyring failure
// arrived as a clipped D-Bus sentence with the actionable half missing. Three lines is
// enough for every message this package produces, including a chmod the user must run.
const authErrLines = 3

// viewAuth is the sign-in panel.
//
// Two routes, cheapest first. Importing from a browser is one keypress; entering the
// cookies is the portable fallback that works everywhere and needs no permissions.
//
// The fields are separate because the two cookies are separate. Asking for both in one
// box meant the user had to assemble "LEETCODE_SESSION=…; csrftoken=…" by hand, which is
// not the shape devtools gives you when you copy the values out of its cookie table, and
// with the box masked there was no way to see what had gone wrong.
func (m Model) viewAuth() string {
	inner := authInnerWidth(m.width)
	f := components.Frame{
		Title:   "sign in",
		Width:   inner + 2,
		Height:  m.authPanelHeight(),
		Focused: true,
	}

	var b strings.Builder
	b.WriteString("\n")

	// --- Route 1: import -----------------------------------------------------
	b.WriteString(" " + theme.Utility.Render(theme.UtilityText("import from browser")) + "\n")
	switch {
	case m.importing != "":
		b.WriteString("  " + theme.Label.Render("Reading "+m.importing+"…") + "\n")
		b.WriteString("  " + theme.Meta.Render("Allow the keychain prompt if it appears.") + "\n")
	case len(m.browsers) == 0:
		b.WriteString("  " + theme.Meta.Render("No supported browser found.") + "\n")
	default:
		for i, br := range m.browsers {
			if i >= authMaxBrowsers {
				break
			}
			// The highlight only shows while the list holds focus, so it never implies
			// that enter would import while the user is typing in a field below.
			marker := " "
			if m.authFocus == authFocusBrowsers && i == m.browserIdx {
				marker = theme.Chars().Cursor
			}
			b.WriteString(" " + theme.Label.Render(marker) + theme.Label.Render(fmt.Sprintf("%d", i+1)) + "  " +
				theme.Body.Render(br.Label()) + "\n")
		}
		b.WriteString("  " + theme.Meta.Render("Reads only your leetcode.com cookies.") + "\n")
	}

	b.WriteString(" " + theme.Rule.Render(strings.Repeat(theme.Chars().DashRule, maxInt(inner-2, 1))) + "\n")

	// --- Route 2: the two cookies -------------------------------------------
	b.WriteString(" " + theme.Utility.Render(theme.UtilityText("or enter cookies")) + "\n")
	b.WriteString(m.authFieldRows(authFieldSession, authFocusSession, "LEETCODE_SESSION", inner))
	b.WriteString(m.authFieldRows(authFieldCSRF, authFocusCSRF, "csrftoken", inner))

	for _, line := range auth.PasteSteps() {
		b.WriteString("  " + theme.Meta.Render(truncate(line, inner-3)) + "\n")
	}

	if m.authVerify {
		b.WriteString("\n " + theme.Label.Render("Checking with LeetCode…") + "\n")
	}

	// Errors wrap instead of truncating. The one that mattered most was the one that
	// used to get cut off.
	if m.authErr != "" {
		b.WriteString("\n")
		for _, line := range authErrorLines(m.authErr, inner-2) {
			b.WriteString(" " + theme.Label.Render(line) + "\n")
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, "", f.Render(b.String()),
		m.authHint(), " "+theme.Meta.Render(truncate(authStorageNote(), inner)))
}

// authFieldRows renders one labelled field: its name, a filled marker, and the input.
func (m Model) authFieldRows(field, focus int, label string, inner int) string {
	// The marker answers "is this one done" at a glance, which is the question the
	// masked field cannot answer on its own.
	mark, style := theme.Chars().Cross, theme.Meta
	if auth.Clean(m.authFields[field].Value()) != "" {
		mark, style = theme.Chars().Check, theme.Label
	}

	head := " " + theme.Meta.Render(label)
	pad := maxInt(inner-lipgloss.Width(label)-3, 1)
	head += strings.Repeat(" ", pad) + style.Render(mark)

	cursor := " "
	if m.authFocus == focus {
		cursor = theme.Chars().Cursor
	}
	return head + "\n " + theme.Label.Render(cursor+" ") + m.authFields[field].View() + "\n"
}

// authHint is the key legend. It names the reveal toggle in the state it will move to,
// because a toggle labelled with its current state reads as an assertion, not an action.
func (m Model) authHint() string {
	reveal := "ctrl+r  reveal"
	if m.authReveal {
		reveal = "ctrl+r  hide"
	}
	// While a check is in flight the only key that does anything is esc. Listing the
	// others would offer actions that silently do nothing, which reads as a hang.
	if m.authVerify {
		return " " + theme.Meta.Render("esc  cancel")
	}

	action := "enter  verify"
	if m.authFocus == authFocusBrowsers {
		action = "enter  import"
	}
	return " " + theme.Meta.Render("tab  next") +
		theme.Rule.Render(sep2()) + theme.Meta.Render(reveal) +
		theme.Rule.Render(sep2()) + theme.Meta.Render(action) +
		theme.Rule.Render(sep2()) + theme.Meta.Render("esc  cancel")
}

// authStorageNote says where the cookies are about to go, before the user types them.
//
// This line used to read "Cookies go to your OS keychain, never to disk." unconditionally,
// which on a machine with no keychain was simply false, and false in the direction that
// reassures. It is now driven by what the machine actually has.
func authStorageNote() string {
	switch auth.Target() {
	case auth.BackendKeyring:
		return "Cookies go to your OS keychain, never to disk."
	case auth.BackendHelper:
		return "Cookies go to your credential helper, never to disk."
	default:
		return "No OS keychain here - cookies go to a 0600 file. See: leetui doctor"
	}
}

// authErrorLines wraps an error to the panel, bounded so a long one cannot push the
// frame past the terminal.
func authErrorLines(msg string, width int) []string {
	lines := wrapPlain(msg, width)
	if len(lines) <= authErrLines {
		return lines
	}
	lines = lines[:authErrLines]
	lines[authErrLines-1] = truncate(lines[authErrLines-1], width)
	return lines
}

// authPanelHeight sizes the panel to its content so the frame never clips a line.
func (m Model) authPanelHeight() int {
	rows := 1 // leading blank
	rows++    // "import from browser"
	switch {
	case m.importing != "":
		rows += 2
	case len(m.browsers) == 0:
		rows++
	default:
		rows += minInt(len(m.browsers), authMaxBrowsers) + 1
	}
	rows++ // divider
	rows++ // "or enter cookies"
	rows += 2 * authFieldCount
	rows += len(auth.PasteSteps())
	if m.authVerify {
		rows += 2
	}
	if m.authErr != "" {
		rows += 1 + len(authErrorLines(m.authErr, authInnerWidth(m.width)-2))
	}
	return rows + 2 // bezels
}
