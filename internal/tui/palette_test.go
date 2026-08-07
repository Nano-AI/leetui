package tui

import (
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/config"
	"github.com/Nano-AI/leetui/internal/store"
)

// TestPaletteSetsAndPersists is the point of the palette: a preference you have to set
// again every morning is not a preference.
func TestPaletteSetsAndPersists(t *testing.T) {
	m := boot(t, true, 120, 34)

	m = drive(t, m, key(":"))
	if !m.paletteOpen {
		t.Fatal(": did not open the command line")
	}

	m.palette.input.SetValue("set default_lang go")
	m = drive(t, m, key("enter"))

	if m.cfg.DefaultLang != "go" {
		t.Errorf("default_lang = %q, want go", m.cfg.DefaultLang)
	}
	if m.paletteOpen {
		t.Error("the command line stayed open after a successful command")
	}

	// Written through, not just held in memory.
	saved, err := config.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if saved.DefaultLang != "go" {
		t.Errorf("reloaded default_lang = %q — the setting was not saved", saved.DefaultLang)
	}
}

// A bad command must say what is wrong and leave the line open to fix, not swallow the
// input and close.
func TestPaletteReportsABadCommand(t *testing.T) {
	m := boot(t, true, 120, 34)
	m = drive(t, m, key(":"))
	m.palette.input.SetValue("set nonsense 1")
	m = drive(t, m, key("enter"))

	if !m.paletteOpen {
		t.Error("the command line closed on an error, discarding what was typed")
	}
	if m.palette.err == "" {
		t.Error("an unknown setting produced no message")
	}
}

func TestPaletteCompletesKeys(t *testing.T) {
	// Completing to the longest COMMON prefix, not the first match: jumping to one
	// arbitrary candidate out of several loses your place.
	got, ok := completeCommand("set git.comm")
	if !ok {
		t.Fatal("no completion for a real prefix")
	}
	if !strings.HasPrefix(got, "set git.commit") {
		t.Errorf("completed to %q", got)
	}
}

// Tags name the technique, so they are hidden until asked for. The line still says how
// many there are and which key shows them — a blank space would read as "no tags".
func TestTagsAreHiddenButAnnounced(t *testing.T) {
	m := boot(t, true, 120, 34)
	m = drive(t, m, key("enter"))

	out := stripANSI(m.View())
	if strings.Contains(out, "array") || strings.Contains(out, "hash-table") {
		t.Errorf("a tag is on screen by default:\n%s", firstLines(out, 6))
	}
	if !strings.Contains(out, "tags hidden") {
		t.Errorf("nothing says the tags exist:\n%s", firstLines(out, 6))
	}

	m = drive(t, m, key("z"))
	if !m.cfg.UI.ShowTags {
		t.Fatal("z did not reveal the tags")
	}
	if out := stripANSI(m.View()); !strings.Contains(out, "array") {
		t.Errorf("tags did not appear after z:\n%s", firstLines(out, 6))
	}
}

func TestHintsAreHiddenButAnnounced(t *testing.T) {
	m := boot(t, true, 120, 34)
	m = drive(t, m, key("enter"))

	m.detail = &store.Detail{
		Row:   store.Row{Slug: m.rows[m.cursor].Slug},
		Hints: []string{"Use a map.", "Two passes are fine."},
	}
	m.detailMD = "A statement.\n"

	out := stripANSI(m.View())
	if strings.Contains(out, "Use a map") {
		t.Error("a hint is on screen by default")
	}
	if !strings.Contains(out, "hints hidden") {
		t.Errorf("nothing says the hints exist:\n%s", firstLines(out, 12))
	}

	m = drive(t, m, key("Z"))
	if out := stripANSI(m.View()); !strings.Contains(out, "Use a map") {
		t.Errorf("hints did not appear after Z:\n%s", firstLines(out, 12))
	}
}

func firstLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

// TestDocsIsReachableAndListsTheSurface. `?` is a keymap: it cannot list the command
// line, the subcommands, or the settings, because none of those are keys. Everything
// added in the last session was reachable and none of it was findable.
func TestDocsIsReachableAndListsTheSurface(t *testing.T) {
	for _, how := range []struct {
		name string
		open func(Model) Model
	}{
		{"D key", func(m Model) Model { return drive(t, m, key("D")) }},
		{":docs", func(m Model) Model {
			m = drive(t, m, key(":"))
			m.palette.input.SetValue("docs")
			return drive(t, m, key("enter"))
		}},
	} {
		t.Run(how.name, func(t *testing.T) {
			m := how.open(boot(t, true, 100, 30))
			if m.mode != modeDocs {
				t.Fatalf("mode = %v, want modeDocs", m.mode)
			}

			out := stripANSI(m.View())
			// The three surfaces that have no key and so cannot appear in the keymap.
			for _, want := range []string{":set", "leetui run", "leetui doctor", "z"} {
				if !strings.Contains(out, want) {
					t.Errorf("the reference does not mention %q", want)
				}
			}
			if m2 := drive(t, m, key("esc")); m2.mode != modeBoard {
				t.Error("esc did not leave the reference")
			}
		})
	}
}
