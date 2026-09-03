package tui

import (
	"context"
	"errors"
	"strings"

	"github.com/Nano-AI/leetui/internal/auth"
	tea "github.com/charmbracelet/bubbletea"
)

// handleAuthVerified is what happens after LeetCode has been asked about the cookies.
//
// Nothing is stored before this point, so every failure path here leaves both the form
// and whatever was already stored exactly as they were. That is the whole reason for the
// round trip: the previous flow wrote first and reported success unconditionally, so a
// typo replaced a working session with a broken one and said "Signed in" about it.
func (m Model) handleAuthVerified(msg authVerifiedMsg) (tea.Model, tea.Cmd) {
	// The user pressed esc while the request was in flight. A cancel has to mean
	// cancelled: without this the answer lands a second later and signs them in anyway,
	// which is the opposite of what they just asked for. The panel is already closed and
	// the fields already cleared, so there is nothing to report and nothing to store.
	if m.mode != modeAuth {
		return m, nil
	}

	m.authVerify = false

	if msg.err != nil {
		m.authErr = verifyFailureHint(msg.err)
		m.importedFrom = ""
		return m, nil
	}

	// Reached LeetCode, and LeetCode does not recognise the session. This is the case
	// the old flow could not tell apart from success.
	if !msg.status.IsSignedIn {
		if msg.source == authSourceImport {
			m.authErr = "The session in " + m.importedFrom + " has expired. Sign in there again, or enter cookies below."
		} else {
			m.authErr = "LeetCode rejected these cookies. They may have expired - copy them again."
		}
		m.importedFrom = ""
		return m, nil
	}

	// Only now is it worth keeping. Username comes from the judge rather than the
	// paste, which is why it is finally a value that can be trusted enough to store.
	creds := msg.creds
	creds.Username = msg.status.Username

	backend, err := auth.Store(creds)
	if err != nil {
		m.authErr = storeFailureHint(err)
		m.importedFrom = ""
		return m, nil
	}

	from := m.importedFrom
	m.importedFrom = ""
	m.client.SetCredentials(creds)
	m.username, m.premium = msg.status.Username, msg.status.IsPremium

	return m.closeAuth(), tea.Batch(
		m.loadAccount(),
		status(signedInMessage(creds.Username, from, backend), false),
	)
}

// signedInMessage names the account AND the store.
//
// Naming the store is not a detail. When leetui falls back from the keychain, that is
// the one moment the user is present, thinking about credentials, and able to do
// something about it. Saying nothing here is how the GitHub CLI ended up with a queue
// of issues from people who had no idea their token was sitting in a readable file.
func signedInMessage(username, from string, b auth.Backend) string {
	var sb strings.Builder
	sb.WriteString("Signed in")
	if username != "" {
		sb.WriteString(" as " + username)
	}
	if from != "" {
		sb.WriteString(" from " + from)
	}

	if b.Secure() {
		sb.WriteString(". Press S to sync.")
		return sb.String()
	}
	// The fallback says so every time, not once. A message the user can miss is a
	// message that has not been delivered.
	sb.WriteString(", saved to " + b.Describe() + " (no keychain here). Press S to sync.")
	return sb.String()
}

// verifyFailureHint separates "we could not ask" from "the answer was no".
//
// They call for opposite responses (retry versus re-copy the cookies) and a user
// staring at a transport error has no way to tell which one they are looking at.
func verifyFailureHint(err error) string {
	if errors.Is(err, auth.ErrSessionExpired) {
		return "LeetCode rejected these cookies. They may have expired - copy them again."
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "Timed out reaching LeetCode. The cookies were not saved; press enter to retry."
	}
	return "Could not reach LeetCode to check these: " + err.Error()
}

// storeFailureHint explains a store that failed rather than one that was unavailable.
//
// Store already falls through unavailable backends on its own, so arriving here means
// the chosen backend exists and refused. That is a real fault with a real cause, and
// the cause is usually one of two things worth naming outright.
func storeFailureHint(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "readable by others"):
		// file.go refuses a loosened credentials file rather than repairing it, and
		// its error already carries the exact chmod to run.
		return msg
	case strings.Contains(msg, "credential helper"):
		return msg + " - fix it, or clear [auth] helper in config.toml to use a file."
	default:
		return "Verified, but could not save: " + msg
	}
}
