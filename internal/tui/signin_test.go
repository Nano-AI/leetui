package tui

import (
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/auth"
	"github.com/Nano-AI/leetui/internal/leetcode"
	tea "github.com/charmbracelet/bubbletea"
)

// openSignIn opens the sign-in panel with focus already in the session field.
//
// Focus lands on the browser list when this machine has one, and whether it does is a
// property of the developer's home directory rather than of the code under test. Tabbing
// to a known position makes these tests say the same thing on every machine.
func openSignIn(t *testing.T) Model {
	t.Helper()
	m := drive(t, boot(t, true, 120, 40), key("a"))
	if m.mode != modeAuth {
		t.Fatalf("pressing a did not open sign-in; mode = %v", m.mode)
	}
	for m.authFocus != authFocusSession {
		m = drive(t, m, key("tab"))
	}
	return m
}

// typeInto sends s one rune at a time, the way a keyboard would.
func typeInto(t *testing.T, m Model, s string) Model {
	t.Helper()
	for _, r := range s {
		m = drive(t, m, key(string(r)))
	}
	return m
}

// paste delivers s as a single event, the way a terminal delivers a bracketed paste.
func paste(t *testing.T, m Model, s string) Model {
	t.Helper()
	return drive(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
}

const (
	testSession = "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.testsessionpayload.testsignature"
	testCSRF    = "9aBcD3fG4hJ5kL6mN7pQ8rS0tU1vW2xY"
)

// TestRevealTogglesBothFields: the fields are masked by default and ctrl+r shows them.
//
// The reveal is the answer to the original complaint (you cannot check an 800-character
// JWT you cannot see) so both halves matter: that it reveals, and that it re-hides.
func TestRevealTogglesBothFields(t *testing.T) {
	m := openSignIn(t)
	m = paste(t, m, testSession)
	m = drive(t, m, key("tab"))
	m = paste(t, m, testCSRF)

	if out := m.View(); strings.Contains(out, testSession) || strings.Contains(out, testCSRF) {
		t.Fatal("cookies are legible before the user asked to see them")
	}

	m = drive(t, m, key("ctrl+r"))
	out := m.View()
	if !strings.Contains(out, testSession) {
		t.Error("ctrl+r did not reveal the session field")
	}
	if !strings.Contains(out, testCSRF) {
		t.Error("ctrl+r did not reveal the csrftoken field")
	}

	m = drive(t, m, key("ctrl+r"))
	if out := m.View(); strings.Contains(out, testSession) || strings.Contains(out, testCSRF) {
		t.Error("a second ctrl+r did not mask the fields again")
	}
}

// TestPastedHeaderFillsBothFields is the fix for the original complaint: one paste,
// two fields, without the user splitting it by hand.
func TestPastedHeaderFillsBothFields(t *testing.T) {
	header := "LEETCODE_SESSION=" + testSession + "; csrftoken=" + testCSRF + ";"

	for _, tc := range []struct {
		name  string
		focus int
	}{
		{"into the session field", authFocusSession},
		{"into the csrftoken field", authFocusCSRF},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := openSignIn(t)
			for m.authFocus != tc.focus {
				m = drive(t, m, key("tab"))
			}
			m = paste(t, m, header)

			if got := m.authFields[authFieldSession].Value(); got != testSession {
				t.Errorf("session = %q, want the pasted session", got)
			}
			if got := m.authFields[authFieldCSRF].Value(); got != testCSRF {
				t.Errorf("csrftoken = %q, want the pasted csrftoken", got)
			}
		})
	}
}

// TestBareValuesAreAccepted: copying the two values out of the devtools cookie table
// gives you bare values with no names attached, which the single-field form rejected
// outright even though it was the flow its own instructions described.
func TestBareValuesAreAccepted(t *testing.T) {
	m := openSignIn(t)
	m = paste(t, m, testSession)
	m = drive(t, m, key("tab"))
	m = paste(t, m, testCSRF)

	got := m.authCreds()
	if got.Session != testSession || got.CSRF != testCSRF {
		t.Fatalf("bare values did not survive the form: %+v", got)
	}
	if !got.Valid() {
		t.Error("two bare values did not produce valid credentials")
	}
}

// TestDigitInFieldDoesNotImport guards a real hazard: the old handler treated 1-9 as
// "import from browser N" whenever the field was empty, so a csrftoken beginning with a
// digit, typed rather than pasted, fired an import nobody asked for.
func TestDigitInFieldDoesNotImport(t *testing.T) {
	m := openSignIn(t)
	m = typeInto(t, m, "1234")

	if m.importing != "" {
		t.Fatalf("typing digits started a browser import from %q", m.importing)
	}
	if got := m.authFields[authFieldSession].Value(); got != "1234" {
		t.Errorf("session field = %q, want the typed digits", got)
	}
}

// TestEnterMovesToTheEmptyField: submitting a half-filled form is a step not finished,
// not an error, so it should move the cursor rather than scold.
func TestEnterMovesToTheEmptyField(t *testing.T) {
	m := openSignIn(t)
	m = paste(t, m, testSession)
	m = drive(t, m, key("enter"))

	if m.authFocus != authFocusCSRF {
		t.Errorf("focus = %d, want the empty csrftoken field", m.authFocus)
	}
	if m.authErr != "" {
		t.Errorf("an unfinished form produced an error: %q", m.authErr)
	}
	if m.authVerify {
		t.Error("an unfinished form started a verification request")
	}
}

// TestRejectedCookiesKeepThePanelOpen: nothing is stored until LeetCode agrees, and a
// rejection has to stay on screen where the user can fix it.
func TestRejectedCookiesKeepThePanelOpen(t *testing.T) {
	m := openSignIn(t)
	m = paste(t, m, testSession)
	m = drive(t, m, key("tab"))
	m = paste(t, m, testCSRF)
	m.authVerify = true

	// isSignedIn false is the case the old flow could not tell apart from success.
	m = drive(t, m, authVerifiedMsg{creds: m.authCreds(), source: authSourcePaste})

	if m.mode != modeAuth {
		t.Fatal("rejected cookies closed the sign-in panel")
	}
	if m.authErr == "" {
		t.Error("rejected cookies produced no explanation")
	}
	if m.authVerify {
		t.Error("the in-flight flag survived the response")
	}
	// The form keeps what was typed, so the user can fix one character rather than
	// paste both cookies again.
	if m.authFields[authFieldSession].Value() != testSession {
		t.Error("a rejection cleared the form")
	}
}

// TestStorageNoteMatchesTheMachine: the footer used to claim the OS keychain
// unconditionally, which on a machine without one was false in the reassuring direction.
func TestStorageNoteMatchesTheMachine(t *testing.T) {
	note := authStorageNote()
	switch auth.Target() {
	case auth.BackendKeyring:
		if !strings.Contains(note, "keychain") {
			t.Errorf("note %q does not mention the keychain it will use", note)
		}
	case auth.BackendHelper:
		if !strings.Contains(note, "helper") {
			t.Errorf("note %q does not mention the helper it will use", note)
		}
	default:
		if !strings.Contains(note, "file") {
			t.Errorf("note %q claims something other than the file it will use", note)
		}
		if strings.Contains(note, "never to disk") {
			t.Error("note promises the secret never reaches disk while writing it to disk")
		}
	}
}

// TestLongAuthErrorIsNotTruncatedToOneLine: the keyring failure that started all of
// this arrived as a clipped sentence with the actionable half missing.
func TestLongAuthErrorIsNotTruncatedToOneLine(t *testing.T) {
	const msg = "Could not reach LeetCode to check these: Post " +
		"\"https://leetcode.com/graphql\": dial tcp: lookup leetcode.com: no such host"

	lines := authErrorLines(msg, 50)
	if len(lines) < 2 {
		t.Fatalf("a %d-character error wrapped to %d line(s)", len(msg), len(lines))
	}
	if len(lines) > authErrLines {
		t.Errorf("error wrapped to %d lines, over the %d-line bound", len(lines), authErrLines)
	}
}

// TestSignedInMessageNamesTheStore: when leetui falls back, the moment it tells the user
// is this one. A confirmation that stays silent about it is how a user ends up believing
// their session is in a keychain that does not exist.
func TestSignedInMessageNamesTheStore(t *testing.T) {
	secure := signedInMessage("ada", "", auth.BackendKeyring)
	if !strings.Contains(secure, "ada") {
		t.Errorf("confirmation %q does not name the account", secure)
	}

	fallback := signedInMessage("ada", "", auth.BackendFile)
	if !strings.Contains(fallback, "file") {
		t.Errorf("fallback confirmation %q does not say where the cookies went", fallback)
	}
}

// TestEscDuringVerificationCancels: a cancel has to mean cancelled. The request is
// already in flight and cannot be recalled, so the guard is on the answer: it arrives a
// second later and must be dropped rather than signing the user in behind their back.
func TestEscDuringVerificationCancels(t *testing.T) {
	m := openSignIn(t)
	m = paste(t, m, testSession)
	m = drive(t, m, key("tab"))
	m = paste(t, m, testCSRF)
	creds := m.authCreds()

	// Update directly rather than through drive. The command enter returns really
	// calls leetcode.com -- verifyCredentials builds its own client, so the harness's
	// offline transport does not reach it -- and it lands either side of drive's 120ms
	// deadline depending on the network. Under it, the answer arrives, clears
	// authVerify, and this test fails; over it, the answer is dropped and it passes.
	// Dropping the command removes the coin flip: the answer that matters here is the
	// synthetic one below, which is the whole point of the test.
	next, _ := m.Update(key("enter"))
	m = next.(Model)
	if !m.authVerify {
		t.Fatal("a complete form did not start a verification")
	}

	m = drive(t, m, key("esc"))
	if m.mode != modeBoard {
		t.Fatal("esc did not close the sign-in panel")
	}

	// The answer the user cancelled, arriving late and saying the cookies were good.
	m = drive(t, m, authVerifiedMsg{
		creds:  creds,
		source: authSourcePaste,
		status: leetcode.UserStatus{IsSignedIn: true, Username: "ada"},
	})

	if m.username == "ada" {
		t.Error("a cancelled sign-in completed anyway")
	}
}
