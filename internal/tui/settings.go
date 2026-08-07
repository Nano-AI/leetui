package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Nano-AI/leetui/internal/config"
	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// The settings screen.
//
// A list over the same registry the palette types into (internal/config/settings.go), so
// the two can never disagree about what exists. The palette is faster once you know a
// name; this is how you find out there is a name.
//
// Booleans toggle in place, which is every setting worth flipping while you are looking
// at it. Anything with a value hands off to the palette pre-filled, rather than growing a
// second text-entry path here that would have to re-implement validation.

func (m Model) openSettings() (tea.Model, tea.Cmd) {
	m.mode = modeSettings
	m.settingsIdx = 0
	return m, nil
}

func (m Model) handleSettingsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	rows := config.Settings()

	switch msg.String() {
	case "esc", "q", "ctrl+c":
		m.mode = modeBoard
		return m, nil

	case "j", "down":
		m.settingsIdx = minInt(m.settingsIdx+1, len(rows)-1)
		return m, nil
	case "k", "up":
		m.settingsIdx = maxInt(m.settingsIdx-1, 0)
		return m, nil
	case "g":
		m.settingsIdx = 0
		return m, nil
	case "G":
		m.settingsIdx = len(rows) - 1
		return m, nil

	case "enter", " ":
		if m.settingsIdx < 0 || m.settingsIdx >= len(rows) {
			return m, nil
		}
		s := rows[m.settingsIdx]
		if s.Kind != config.KindBool {
			// Values need typing, and the palette already knows how to validate one.
			m.mode = modeBoard
			model, cmd := m.openPalette()
			mm := model.(Model)
			mm.palette.input.SetValue("set " + s.Key + " " + s.Get(&mm.cfg))
			mm.palette.input.CursorEnd()
			return mm, cmd
		}
		return m.toggleSetting(s)
	}
	return m, nil
}

// toggleSetting flips a boolean and writes the config.
func (m Model) toggleSetting(s config.Setting) (tea.Model, tea.Cmd) {
	cfg := m.cfg
	if err := cfg.Apply(s.Key, ""); err != nil {
		return m, status(err.Error(), true)
	}
	if err := cfg.Save(); err != nil {
		return m, status("could not save: "+err.Error(), true)
	}
	m.cfg = cfg
	return m.afterSet(s.Key, s.Get(&m.cfg))
}

// viewSettings lists every option with its current value.
func (m Model) viewSettings() string {
	rows := config.Settings()

	var b strings.Builder
	b.WriteString("\n " + lipgloss.NewStyle().Foreground(theme.Amber).Bold(true).
		Render(theme.Display("settings")) + "\n\n")

	// The widest key decides the column, so the values line up and can be scanned down
	// rather than hunted for.
	keyW := 0
	for _, s := range rows {
		keyW = maxInt(keyW, len(s.Key))
	}

	for i, s := range rows {
		cursor := "  "
		style := theme.Body
		if i == m.settingsIdx {
			cursor = " " + theme.Chars().Cursor
			style = lipgloss.NewStyle().Foreground(theme.Bone).Bold(true)
		}

		value := s.Get(&m.cfg)
		if value == "" {
			value = "—"
		}

		b.WriteString(fmt.Sprintf("%s %s  %s  %s\n",
			theme.Label.Render(cursor),
			theme.Label.Width(keyW).Render(s.Key),
			style.Width(10).Render(value),
			theme.Meta.Render(truncate(s.Help, maxInt(m.width-keyW-20, 10)))))
	}

	b.WriteString("\n " + theme.Meta.Render("Written to "+m.cfg.Path()) + "\n")
	b.WriteString("\n " +
		theme.Label.Render("enter") + theme.Body.Render(" toggle or edit   ") +
		theme.Label.Render(":") + theme.Body.Render(" command line   ") +
		theme.Label.Render("esc") + theme.Body.Render(" back") + "\n")
	return b.String()
}

// viewPalette is the command line, drawn over the bottom of whatever is on screen.
func (m Model) viewPalette() string {
	if !m.paletteOpen {
		return ""
	}

	line := " " + theme.Label.Render(":") + m.palette.input.View()

	switch {
	case m.palette.err != "":
		// Errors are amber, not red: red belongs to the judge.
		return line + "\n " + theme.Label.Render(truncate(m.palette.err, maxInt(m.width-2, 10)))
	case m.palette.hint != "":
		return line + "\n " + theme.Meta.Render(truncate(m.palette.hint, maxInt(m.width-2, 10)))
	default:
		return line + "\n " + theme.Meta.Render("set · settings · sync · git · help   tab completes")
	}
}

// toggleSpoiler flips one of the two reveal settings from a single key.
//
// Tags and hints are hidden by default because both give the approach away, and reading
// one is not a decision — you cannot un-see "hash-table" while deciding whether to
// attempt the problem. But sometimes you are stuck and you WANT the hint, and having to
// remember a settings path in that moment is the wrong time to make someone navigate.
func (m Model) toggleSpoiler(key string) (tea.Model, tea.Cmd) {
	s, ok := config.FindSetting(key)
	if !ok {
		return m, nil
	}
	return m.toggleSetting(s)
}
