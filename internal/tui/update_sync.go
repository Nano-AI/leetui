package tui

import (
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/syncer"
)

// handleSyncProgress drives every background pull — the problem list, the company
// registry, and a company pack all report through this one path (see syncJob).
//
// What differs between them is only the closing sentence, which is why the phase is read
// at the end rather than branched on at the top.
func (m Model) handleSyncProgress(p syncer.Progress) (tea.Model, tea.Cmd) {
	m.syncProgress = p

	if !p.Finished {
		// Refresh the board as the first pages land, so the setup screen hands over to
		// a populated list rather than an empty one.
		if m.mode == modeSetup && p.Done > 0 && p.Done%500 == 0 {
			return m, tea.Batch(waitForSync(m.syncCh), m.loadRows())
		}
		return m, waitForSync(m.syncCh)
	}

	m.syncing = false
	m.syncCh = nil
	m.syncCancel = nil
	if m.mode == modeSetup {
		m.mode = modeBoard
	}

	if p.Err != nil {
		return m, m.syncFailed(p)
	}
	if p.Note == "paused" {
		return m, tea.Batch(
			m.loadRows(),
			status(fmt.Sprintf("Sync paused at %d problems. Press S to resume.", p.Done), false),
		)
	}

	switch p.Phase {
	case syncer.PhaseCompanies:
		return m, tea.Batch(m.loadRows(), m.loadCompanies(), status(m.packDone(p), false))
	default:
		return m, tea.Batch(m.loadRows(), status(fmt.Sprintf("Synced %d problems.", p.Done), false))
	}
}

// packDone words the outcome of a company job. The registry job carries no company name
// in its Note; a pack job does, and naming it is what confirms which list was pulled.
func (m Model) packDone(p syncer.Progress) string {
	if p.Note == "done" && m.pack.Company == "" {
		return fmt.Sprintf("Loaded %d companies. Press c to browse them.", p.Done)
	}
	return fmt.Sprintf("Pulled %d problems for %s, %s.",
		p.Done, m.pack.Name, m.pack.Timeframe.Label())
}

// syncFailed turns a job's final error into something actionable.
//
// A premium gate is the one worth naming precisely: a company pack that comes back empty
// is not a broken sync, it is an account without a subscription, and telling the user to
// "press S to resume" would send them in a circle (D-006).
func (m Model) syncFailed(p syncer.Progress) tea.Cmd {
	switch {
	case errors.Is(p.Err, leetcode.ErrSessionExpired):
		return status("Session expired mid-sync. Press a to re-authenticate, then S to resume.", true)

	case errors.Is(p.Err, leetcode.ErrPremiumRequired):
		return status("Company lists need LeetCode Premium. Everything else still works.", true)

	case errors.Is(p.Err, leetcode.ErrNotFound) && p.Phase == syncer.PhaseCompanies:
		return status("LeetCode has no such company list.", true)

	default:
		return tea.Batch(
			m.loadRows(),
			status("Sync stopped: "+p.Err.Error()+" Press S to resume.", true),
		)
	}
}
