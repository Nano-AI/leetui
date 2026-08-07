package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Nano-AI/leetui/internal/config"
)

// The command palette.
//
// One typed surface over the settings registry, so every option is reachable by NAME and
// discoverable without memorising a key. It is the same argument D-013 makes about the
// keymap: a thing that can only be reached by a key nobody can find is a thing that does
// not exist.
//
// It writes through to config.toml. A setting changed here survives the session, because
// a preference you have to set again every morning is not a preference.
//
// Deliberately small. This is `:set`, not a scripting language — anything that wants
// conditionals belongs in a shell script calling the CLI (docs/AGENTS.md).

// paletteState is the typed command line.
type paletteState struct {
	input textinput.Model
	// err is the last command's complaint, held so it survives the frame.
	err string
	// hint is what the current input would do, updated per keystroke.
	hint string
}

func newPalette() paletteState {
	in := textinput.New()
	in.Prompt = ""
	in.Placeholder = "set default_lang go"
	in.CharLimit = 200
	return paletteState{input: in}
}

// openPalette focuses the command line.
func (m Model) openPalette() (tea.Model, tea.Cmd) {
	m.palette = newPalette()
	m.palette.input.Focus()
	m.paletteOpen = true
	return m, textinput.Blink
}

func (m Model) closePalette() Model {
	m.paletteOpen = false
	m.palette.input.Blur()
	m.palette.input.SetValue("")
	return m
}

// handlePaletteKey drives the command line.
func (m Model) handlePaletteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		return m.closePalette(), nil

	case "enter":
		line := strings.TrimSpace(m.palette.input.Value())
		if line == "" {
			return m.closePalette(), nil
		}
		return m.runCommand(line)

	case "tab":
		// Complete the key, not the value: the keys are a closed set and the values
		// are not, so this is the half that can be completed honestly.
		if done, ok := completeCommand(m.palette.input.Value()); ok {
			m.palette.input.SetValue(done)
			m.palette.input.CursorEnd()
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.palette.input, cmd = m.palette.input.Update(msg)
	m.palette.err = ""
	m.palette.hint = describeCommand(m.palette.input.Value())
	return m, cmd
}

// runCommand executes one line.
func (m Model) runCommand(line string) (tea.Model, tea.Cmd) {
	verb, rest, _ := strings.Cut(line, " ")

	switch verb {
	case "set":
		return m.runSet(rest)

	case "sync":
		m = m.closePalette()
		return m.startSync()

	case "q", "quit":
		return m, tea.Quit

	case "help":
		m = m.closePalette()
		m.mode = modeHelp
		return m, nil

	case "settings":
		m = m.closePalette()
		return m.openSettings()

	case "docs":
		m = m.closePalette()
		return m.openDocs()

	case "git":
		m = m.closePalette()
		return m.openGit()

	default:
		m.palette.err = fmt.Sprintf("no command %q — try docs, set, settings, sync, git, help", verb)
		return m, nil
	}
}

// runSet applies one setting and writes the config.
func (m Model) runSet(rest string) (tea.Model, tea.Cmd) {
	key, value, _ := strings.Cut(strings.TrimSpace(rest), " ")
	key = strings.TrimSpace(key)
	if key == "" {
		m.palette.err = "set what? e.g. set default_lang go"
		return m, nil
	}

	cfg := m.cfg
	if err := cfg.Apply(key, strings.TrimSpace(value)); err != nil {
		m.palette.err = err.Error()
		return m, nil
	}

	// Write through before anything else. A setting that only holds until you quit is a
	// setting you will change again tomorrow.
	if err := cfg.Save(); err != nil {
		m.palette.err = "could not save: " + err.Error()
		return m, nil
	}
	m.cfg = cfg

	s, _ := config.FindSetting(key)
	m = m.closePalette()
	return m.afterSet(key, s.Get(&m.cfg))
}

// afterSet applies anything that has to happen beyond storing the value, and says what
// changed.
//
// Some settings are read once at startup, and pretending otherwise would be the worse
// failure: the user sets it, nothing visibly happens, and they conclude it is broken.
func (m Model) afterSet(key, value string) (tea.Model, tea.Cmd) {
	note := fmt.Sprintf("%s = %s", key, value)

	switch key {
	case "ui.show_tags", "ui.show_hints":
		// Both are pure render state, so the next frame already has it.

	case "ui.reduce_motion":
		components := value == "true"
		_ = components
		note += " · takes effect on the next flip"

	case "ui.ascii":
		note += " · restart to redraw the frame"

	case "default_lang", "workspace", "editor":
		// Read per use, so they apply immediately.

	default:
		if strings.HasPrefix(key, "keys.") {
			m.keys = m.cfg.Actions()
		}
	}
	return m, status(note, false)
}

// completeCommand extends a partial command to the longest unambiguous prefix.
func completeCommand(line string) (string, bool) {
	verb, rest, hasSpace := strings.Cut(line, " ")
	if !hasSpace {
		if full, ok := longestPrefix(verbs(), verb); ok {
			return full + " ", true
		}
		return "", false
	}
	if verb != "set" {
		return "", false
	}

	key, value, hasValue := strings.Cut(rest, " ")
	if hasValue {
		// The value side is open — a workspace path or a language slug — so there is
		// nothing honest to complete.
		_ = value
		return "", false
	}
	if full, ok := longestPrefix(config.SettingKeys(), key); ok {
		return "set " + full + " ", true
	}
	return "", false
}

// describeCommand says what the current input would do, or "" when it is not yet a
// command.
func describeCommand(line string) string {
	verb, rest, _ := strings.Cut(strings.TrimSpace(line), " ")
	if verb != "set" {
		return ""
	}
	key, _, _ := strings.Cut(strings.TrimSpace(rest), " ")
	s, ok := config.FindSetting(strings.TrimSpace(key))
	if !ok {
		return ""
	}
	return s.Help
}

func verbs() []string {
	return []string{"docs", "set", "settings", "sync", "git", "help", "quit"}
}

// longestPrefix returns the longest string every candidate with this prefix shares.
//
// Completing to the longest COMMON prefix rather than to the first match: jumping to one
// arbitrary candidate out of several is how a completion loses your place.
func longestPrefix(candidates []string, prefix string) (string, bool) {
	var matches []string
	for _, c := range candidates {
		if strings.HasPrefix(c, prefix) {
			matches = append(matches, c)
		}
	}
	if len(matches) == 0 {
		return "", false
	}
	sort.Strings(matches)

	common := matches[0]
	for _, c := range matches[1:] {
		for !strings.HasPrefix(c, common) {
			common = common[:len(common)-1]
		}
	}
	if common == prefix {
		return "", false
	}
	return common, true
}
