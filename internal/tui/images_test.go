package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/Nano-AI/leetui/internal/config"
	"github.com/Nano-AI/leetui/internal/render"
	"github.com/Nano-AI/leetui/internal/store"
)

// TWO renderers write markers in TWO formats — the statement says "[▸ img 3]" and the
// editorial says "[▸ 3]". A parser that knew one would draw figures on statements and
// silently nothing on editorials, and nothing about either would look broken.
func TestFindMarkerReadsBothFormats(t *testing.T) {
	for _, tc := range []struct {
		line    string
		wantN   int
		wantCol int
	}{
		{"  [▸ img 1]", 1, 2},
		{"  [▸ img 12 — dp table]", 12, 2},
		{"  [▸ 1]", 1, 2},
		{"  [▸ 3 — figure]", 3, 2},
		{"   [▸ 7 — a label with — a dash]", 7, 3},

		{"no marker here", -1, -1},
		{"[▸ notanumber]", -1, -1},
		{"[▸ 1 unterminated", -1, -1},
	} {
		col, n := findMarker(tc.line)
		if tc.wantN < 0 {
			if col >= 0 {
				t.Errorf("%q: found a marker where there is none (col %d, n %d)", tc.line, col, n)
			}
			continue
		}
		if n != tc.wantN || col != tc.wantCol {
			t.Errorf("%q: got col %d n %d, want col %d n %d",
				tc.line, col, n, tc.wantCol, tc.wantN)
		}
	}
}

// TestPlacementCannotDisturbTheFrame is the reason this was deferred, made into a test.
//
// An image escape is not text. The objection was that the layout has no idea it is
// there, so a placement would push the frame around and shear the bezel. The answer is
// that placement happens AFTER the frame is composed and writes nothing into it — every
// byte appended is zero-width, so the frame is byte-identical with images on and off.
func TestPlacementCannotDisturbTheFrame(t *testing.T) {
	m := boot(t, true, 120, 34)
	m = drive(t, m, key("enter"))

	m.detail = &store.Detail{Row: store.Row{Slug: m.rows[m.cursor].Slug}}
	m.detailMD = "Some text.\n\n`[▸ img 1 — a tree]`\n\nMore text.\n"
	m.detailImages = []render.Image{{URL: "https://example.com/1.png", Alt: "a tree"}}

	// With drawing off — the marker, and nothing else.
	m.cfg.UI.InlineImages = false
	plain := m.View()

	// With drawing on, and a figure already uploaded.
	m.cfg.UI.InlineImages = true
	m.graphics = config.Graphics{Protocol: config.ProtocolKitty, Terminal: "kitty"}
	m.images = map[string]*imageState{
		"https://example.com/1.png": {id: 1, transmitted: "\x1b_Ga=t;PAYLOAD\x1b\\"},
	}
	drawn := m.View()

	if drawn == plain {
		t.Fatal("nothing was emitted; the rest of this test would pass vacuously")
	}
	// EVERY line must be unchanged in width, and the frame must be the same height.
	pl := strings.Split(plain, "\n")
	dl := strings.Split(drawn, "\n")
	if len(pl) != len(dl) {
		t.Fatalf("frame is %d lines with images, %d without", len(dl), len(pl))
	}
	for i := range pl {
		if ansi.StringWidth(pl[i]) != ansi.StringWidth(dl[i]) {
			t.Errorf("line %d is %d cells with images and %d without — the frame moved",
				i, ansi.StringWidth(dl[i]), ansi.StringWidth(pl[i]))
		}
	}
	// And the visible text is identical: the marker stays as the fallback.
	if ansi.Strip(plain) != ansi.Strip(strings.Split(drawn, "\x1b_G")[0])[:len(ansi.Strip(plain))] {
		t.Error("the visible text changed")
	}
}

// A terminal that cannot draw gets exactly the marker, with no escape appended.
func TestNoPlacementWithoutAGraphicsProtocol(t *testing.T) {
	m := boot(t, true, 120, 34)
	m = drive(t, m, key("enter"))
	m.cfg.UI.InlineImages = true
	m.graphics = config.Graphics{Protocol: config.ProtocolNone, Terminal: "xterm"}
	m.images = map[string]*imageState{"x": {id: 1}}

	if strings.Contains(m.View(), "\x1b_G") {
		t.Error("a graphics escape reached a terminal that cannot draw one")
	}
}

// tmux blocking is not the same as tmux being absent: with passthrough off, drawing is
// off, because the escape would be eaten and the marker is the honest answer.
func TestNoPlacementWhenTmuxBlocks(t *testing.T) {
	m := boot(t, true, 120, 34)
	m.cfg.UI.InlineImages = true
	m.graphics = config.Graphics{Protocol: config.ProtocolKitty, InTmux: true, Blocked: true}

	if m.imagesEnabled() {
		t.Error("drawing stayed on while tmux was swallowing the escape")
	}
}
