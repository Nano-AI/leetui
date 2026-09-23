package tui

import (
	"github.com/Nano-AI/leetui/internal/auth"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// The sign-in form's focus ring.
//
// The browser list is a focus position rather than a set of always-live digit shortcuts.
// That is what makes the form safe to type into: the previous version treated a bare
// 1-9 as "import from browser number N" whenever the field was empty, so a csrftoken
// that happens to start with a digit, entered by hand rather than pasted, would fire an
// import the user never asked for. With a ring, a digit inside a field is always a digit.
const (
	authFocusBrowsers = iota
	authFocusSession
	authFocusCSRF
	authFocusCount
)

// Field indices into Model.authFields.
const (
	authFieldSession = iota
	authFieldCSRF
	authFieldCount
)

// authFieldForFocus maps a focus position to the field it edits, if any.
func authFieldForFocus(focus int) (int, bool) {
	switch focus {
	case authFocusSession:
		return authFieldSession, true
	case authFocusCSRF:
		return authFieldCSRF, true
	}
	return 0, false
}

// beginAuth opens the sign-in panel.
//
// Focus starts on the browser list when there is one, because importing is one keypress
// and typing two cookies by hand is not. With no browsers detected the list is not worth
// a stop on the ring, so focus starts in the first field.
func (m Model) beginAuth() (Model, tea.Cmd) {
	m.mode = modeAuth
	m.authErr = ""
	m.importing = ""
	m.authReveal = false
	m.authVerify = false
	m.browsers = auth.DetectBrowsers()
	m.browserIdx = 0

	for i := range m.authFields {
		m.authFields[i].SetValue("")
		m.authFields[i].EchoMode = textinput.EchoPassword
		m.authFields[i].Blur()
	}

	m.authFocus = authFocusSession
	if len(m.browsers) > 0 {
		m.authFocus = authFocusBrowsers
	}
	return m.applyAuthFocus(), textinput.Blink
}

// applyAuthFocus makes exactly one field hold the cursor.
func (m Model) applyAuthFocus() Model {
	for i := range m.authFields {
		m.authFields[i].Blur()
	}
	if f, ok := authFieldForFocus(m.authFocus); ok {
		m.authFields[f].Focus()
	}
	return m
}

// authFocusStart is the first position on the ring, skipping an empty browser list.
func (m Model) authFocusStart() int {
	if len(m.browsers) == 0 {
		return authFocusSession
	}
	return authFocusBrowsers
}

// moveAuthFocus steps the ring by delta, wrapping, skipping an absent browser list.
func (m Model) moveAuthFocus(delta int) Model {
	start := m.authFocusStart()
	span := authFocusCount - start
	next := ((m.authFocus-start+delta)%span + span) % span
	m.authFocus = start + next
	return m.applyAuthFocus()
}

// closeAuth abandons sign-in, clearing both secrets out of memory on the way.
func (m Model) closeAuth() Model {
	m.mode = modeBoard
	m.authErr = ""
	m.authVerify = false
	m.authReveal = false
	for i := range m.authFields {
		m.authFields[i].Blur()
		m.authFields[i].SetValue("")
	}
	return m
}

// authCreds is what is currently in the form, cleaned.
func (m Model) authCreds() auth.Credentials {
	return auth.Credentials{
		Session: auth.Clean(m.authFields[authFieldSession].Value()),
		CSRF:    auth.Clean(m.authFields[authFieldCSRF].Value()),
	}
}

func (m Model) handleAuthKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// A verification is in flight. Let the user back out, but do not let a second one
	// start on top of it or let the fields change under the request that is checking them.
	if m.authVerify {
		if key == "esc" || key == "ctrl+c" {
			return m.closeAuth(), nil
		}
		return m, nil
	}

	switch key {
	case "esc", "ctrl+c":
		return m.closeAuth(), nil

	case "tab", "shift+tab", "up", "down":
		// Up and down double as list navigation while the browser list holds focus,
		// which is what the arrows mean everywhere else in the app.
		if m.authFocus == authFocusBrowsers && (key == "up" || key == "down") {
			return m.moveBrowserCursor(key), nil
		}
		if key == "shift+tab" || key == "up" {
			return m.moveAuthFocus(-1), nil
		}
		return m.moveAuthFocus(1), nil

	case "ctrl+r":
		// Reveal both fields together. Unmasking one at a time would invite comparing a
		// visible value against a masked one, which is the check the user cannot do.
		m.authReveal = !m.authReveal
		echo := textinput.EchoPassword
		if m.authReveal {
			echo = textinput.EchoNormal
		}
		for i := range m.authFields {
			m.authFields[i].EchoMode = echo
		}
		return m, nil

	case "enter":
		if m.authFocus == authFocusBrowsers {
			if m.browserIdx < len(m.browsers) {
				return m.importFromBrowser(m.browsers[m.browserIdx])
			}
			return m, nil
		}
		return m.submitAuth()
	}

	// Digits pick a browser only while the list holds focus. Inside a field they are
	// ordinary characters, which is what makes a csrftoken beginning with one safe.
	if m.authFocus == authFocusBrowsers && len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
		if n := int(key[0] - '1'); n < len(m.browsers) {
			return m.importFromBrowser(m.browsers[n])
		}
		return m, nil
	}

	field, ok := authFieldForFocus(m.authFocus)
	if !ok {
		return m, nil
	}

	var cmd tea.Cmd
	m.authFields[field], cmd = m.authFields[field].Update(msg)

	// Smart paste. If what landed in this field turns out to be a whole cookie header,
	// a cURL command, or a JSON export, split it across both fields rather than making
	// the user do it. Either field accepts it, because the user has no reason to know
	// which one we would have preferred.
	if creds, ok := auth.Sniff(m.authFields[field].Value()); ok {
		m.authFields[authFieldSession].SetValue(creds.Session)
		m.authFields[authFieldCSRF].SetValue(creds.CSRF)
		m.authErr = ""
		m.authFocus = authFocusCSRF
		m = m.applyAuthFocus()
		m.authFields[authFieldCSRF].CursorEnd()
	}

	return m, cmd
}

func (m Model) moveBrowserCursor(key string) Model {
	if len(m.browsers) == 0 {
		return m
	}
	limit := minInt(len(m.browsers), authMaxBrowsers)
	if key == "up" {
		if m.browserIdx == 0 {
			// Off the top of the list is into the fields, so the ring stays traversable
			// with the arrows alone.
			m.authFocus = authFocusCSRF
			return m.applyAuthFocus()
		}
		m.browserIdx--
		return m
	}
	if m.browserIdx+1 >= limit {
		m.authFocus = authFocusSession
		return m.applyAuthFocus()
	}
	m.browserIdx++
	return m
}

// submitAuth verifies the form against LeetCode, and only then stores it.
//
// Verifying first is the point. Storing whatever parses and reporting success meant a
// typo'd or expired cookie produced a cheerful "Signed in" and then failed later, mid
// action, as "Session expired", at which point the user has no reason to connect the
// failure to what they typed minutes ago.
func (m Model) submitAuth() (tea.Model, tea.Cmd) {
	creds := m.authCreds()

	// An empty field is a step the user has not reached yet, not a mistake worth an
	// error message. Send them to it.
	if creds.Session == "" {
		m.authFocus = authFocusSession
		return m.applyAuthFocus(), nil
	}
	if creds.CSRF == "" {
		m.authFocus = authFocusCSRF
		return m.applyAuthFocus(), nil
	}

	// Write back the cleaned values so the form shows what will actually be sent.
	m.authFields[authFieldSession].SetValue(creds.Session)
	m.authFields[authFieldCSRF].SetValue(creds.CSRF)

	m.authErr = ""
	m.authVerify = true
	return m, verifyCredentials(creds, authSourcePaste)
}
