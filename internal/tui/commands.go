package tui

import (
	"context"
	"errors"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/auth"
	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/render"
	"github.com/Nano-AI/leetui/internal/store"
)

// ---------------------------------------------------------------------------
// Commands
// ---------------------------------------------------------------------------

func (m Model) loadRows() tea.Cmd {
	f := m.filter
	st := m.store
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		rows, err := st.Query(ctx, f)
		return rowsMsg{rows: rows, err: err}
	}
}

// detailDebounce is how long the cursor must rest before an uncached statement is
// fetched over the network.
//
// Without it, holding j fires one request per row: the rate limiter queues them all, the
// pane flashes through every problem in between, and every request but the last is
// wasted. 140ms sits below the point where a deliberate move feels delayed and well
// above key-repeat speed.
const detailDebounce = 140 * time.Millisecond

// loadDetailForCursor shows the selected problem's statement.
//
// Two paths, and the split is what removes the flicker:
//
//   - CACHED: read from SQLite and render immediately. No timer, no loading state.
//     After the first visit, every problem takes this path.
//   - UNCACHED: debounce, then fetch once the cursor settles.
//
// The pane's heading is NOT cleared here — view.go renders it from the board row, which
// is always in memory. Only the statement body waits. Clearing the whole pane on every
// keypress is what made scrolling flash.
func (m *Model) loadDetailForCursor() tea.Cmd {
	slug := m.currentSlug()
	if slug == "" {
		m.detail, m.detailMD, m.detailImages = nil, "", nil
		return nil
	}

	// Drop the previous problem's body so the pane never shows one problem's text under
	// another problem's heading.
	if m.detail != nil && m.detail.Slug != slug {
		m.detail, m.detailMD, m.detailImages = nil, "", nil
		m.detailScroll = 0
	}

	m.detailSeq++
	seq := m.detailSeq
	st := m.store
	width := m.detailWidth()

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		d, err := st.Get(ctx, slug)
		if err != nil || d == nil || !d.HasDetail {
			// Not cached — hand off to the debounced network path.
			return detailFetchMsg{slug: slug, seq: seq}
		}
		md, images, rerr := renderStatement(d, width)
		return detailMsg{slug: slug, seq: seq, detail: d, markdown: md, images: images, err: rerr}
	}
}

// fetchDetail pulls a statement from LeetCode and caches it for next time.
func (m Model) fetchDetail(slug string, seq int) tea.Cmd {
	sy := m.sync
	width := m.detailWidth()

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		d, err := sy.Detail(ctx, slug, false)
		if err != nil && d == nil {
			return detailMsg{slug: slug, seq: seq, err: err}
		}
		md, images, rerr := renderStatement(d, width)
		if rerr != nil {
			return detailMsg{slug: slug, seq: seq, detail: d, err: rerr}
		}
		return detailMsg{slug: slug, seq: seq, detail: d, markdown: md, images: images, err: err}
	}
}

// renderDetail re-renders an already-loaded statement, for resizes.
func (m Model) renderDetail(d *store.Detail) tea.Cmd {
	width := m.detailWidth()
	slug := d.Slug
	content := d.Content
	return func() tea.Msg {
		md, images, err := render.HTML(content, width)
		return detailMsg{slug: slug, detail: d, markdown: md, images: images, err: err}
	}
}

func renderStatement(d *store.Detail, width int) (string, []render.Image, error) {
	if d == nil || strings.TrimSpace(d.Content) == "" {
		return "", nil, nil
	}
	return render.HTML(d.Content, width)
}

// importFromBrowser reads the LeetCode session out of a browser's cookie store.
//
// On macOS this opens a Keychain permission prompt, which is the right place for the
// user to grant or refuse a terminal app access to browser cookies. The prompt is modal
// and blocks the goroutine, not the UI.
func (m Model) importFromBrowser(b auth.Browser) (tea.Model, tea.Cmd) {
	m.importing = b.Label()
	m.authErr = ""

	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), auth.ImportTimeout)
		defer cancel()

		creds, err := auth.Import(ctx, b)
		return browserImportMsg{from: b, creds: creds, err: err}
	}
}

// browserImportHint turns an import failure into something the user can act on.
//
// The common cases are "you are not signed in there", "you declined the keychain
// prompt", and, on Linux, "this profile is encrypted with a key we cannot reach".
// None of them is served by showing a raw error.
func browserImportHint(b auth.Browser, err error) string {
	msg := err.Error()
	switch {
	case errors.Is(err, auth.ErrNoBrowserCookies):
		return "No LeetCode session in " + b.Label() + ". Sign in there, or enter them below."

	case strings.Contains(msg, "keychain"):
		return "Keychain access denied. Allow it, or enter the cookies below."

	// Chromium on Linux encrypts its cookie store with a key held in the Secret
	// Service, and falls back to a fixed passphrase when there is none. Decrypting a
	// profile written under the other assumption fails deep in the cipher, so what
	// surfaces is "bad padding" — true, and useless. Say what it means instead.
	case strings.Contains(msg, "bad padding"), strings.Contains(msg, "unsupported cookie encryption"):
		return "Cannot decrypt " + b.Label() + " cookies on this machine. Enter them below."

	// No Secret Service at all, which is the ordinary case on a headless box or over
	// SSH. The raw godbus text names an interface the user has never heard of.
	case strings.Contains(msg, "org.freedesktop.secrets"), strings.Contains(msg, "session bus"):
		return "No keychain on this machine to unlock " + b.Label() + ". Enter them below."

	default:
		return "Could not read " + b.Label() + ": " + msg
	}
}

// authSource records which route produced a set of credentials, so the confirmation
// can say where they came from after the round trip that verifies them.
type authSource int

const (
	authSourcePaste authSource = iota
	authSourceImport
)

// verifyCredentials asks LeetCode whether these cookies actually work.
//
// It builds a throwaway client rather than calling SetCredentials on the live one, so
// an unverified session never becomes the one the rest of the app is using. If the
// check fails, there is nothing to roll back.
func verifyCredentials(c auth.Credentials, src authSource) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), authVerifyTimeout)
		defer cancel()

		cl := leetcode.New(leetcode.WithCredentials(c))
		st, err := cl.Status(ctx)
		return authVerifiedMsg{creds: c, source: src, status: st, err: err}
	}
}

// authVerifyTimeout bounds the check. Short, because the user is waiting on it with a
// half-finished form on screen and a slow answer is worse than a retry.
const authVerifyTimeout = 15 * time.Second

func (m Model) loadAccount() tea.Cmd {
	sy := m.sync
	authed := m.client.Authenticated()
	return func() tea.Msg {
		if !authed {
			return accountMsg{}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		st, err := sy.Account(ctx)
		return accountMsg{status: st, err: err}
	}
}
