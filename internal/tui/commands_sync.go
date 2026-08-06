package tui

import (
	"context"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/grootbeat/leetui/internal/leetcode"
	"github.com/grootbeat/leetui/internal/syncer"
)

// beginSync starts the background sync and returns the command that pumps its progress.
// It is a no-op if a sync is already running.
func (m *Model) beginSync() tea.Cmd {
	if m.syncing {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan syncer.Progress, 8)

	m.syncing = true
	m.syncCh = ch
	m.syncCancel = cancel
	m.syncProgress = syncer.Progress{}

	sy := m.sync
	go func() {
		// Errors are reported through the channel's final Progress, so the return value
		// is intentionally discarded here.
		_ = sy.Problems(ctx, ch, true)
	}()

	return waitForSync(ch)
}

// startSync is the sync key's handler.
//
// Pressing it while a sync runs cancels it. The job is checkpointed, so cancelling keeps
// everything already written and the next run resumes from there (D-008).
func (m Model) startSync() (tea.Model, tea.Cmd) {
	if m.syncing {
		if m.syncCancel != nil {
			m.syncCancel()
		}
		return m, status("Stopping sync…", false)
	}
	return m, tea.Batch(m.beginSync(), status("Syncing problems…", false))
}

// waitForSync blocks in a goroutine (never in Update) for the next progress update.
func waitForSync(ch <-chan syncer.Progress) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		p, ok := <-ch
		if !ok {
			return syncProgressMsg(syncer.Progress{Finished: true})
		}
		return syncProgressMsg(p)
	}
}

// openInBrowser opens the selected problem on leetcode.com.
func (m Model) openInBrowser() (tea.Model, tea.Cmd) {
	slug := m.currentSlug()
	if slug == "" {
		return m, nil
	}
	return m, openURL(leetcode.BaseURL + "/problems/" + slug + "/")
}

// openImage opens the nth bracket marker's image (1-indexed, matching the label).
func (m Model) openImage(n int) (tea.Model, tea.Cmd) {
	if n < 1 || n > len(m.detailImages) {
		return m, nil
	}
	return m, openURL(m.detailImages[n-1].URL)
}

// openURL hands a URL to the OS.
//
// The URL is passed as a separate argument, never interpolated into a shell string —
// these come from LeetCode's HTML and are not ours to trust.
func openURL(url string) tea.Cmd {
	return func() tea.Msg {
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			return statusMsg{text: "Refused to open a non-web URL.", isError: true}
		}
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default:
			cmd = exec.Command("xdg-open", url)
		}
		if err := cmd.Start(); err != nil {
			return statusMsg{text: "Could not open the browser: " + err.Error(), isError: true}
		}
		return statusMsg{text: "Opened in your browser."}
	}
}

func status(text string, isError bool) tea.Cmd {
	return func() tea.Msg { return statusMsg{text: text, isError: isError} }
}
