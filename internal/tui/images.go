package tui

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/Nano-AI/leetui/internal/config"
	"github.com/Nano-AI/leetui/internal/render"
)

// Drawing a figure inside the pane.
//
// The numbered marker is the floor and stays: it works in every terminal, it survives
// NO_COLOR and ASCII mode, and it is what a screen reader gets. This draws ON TOP of it
// when the terminal can, and falls back to exactly the marker when it cannot.
//
// WHY THIS WAS DEFERRED, AND WHAT CHANGED. An image escape is not text: the layout has
// no idea it is there, so a naive placement pushes the frame around and shears the
// bezel. The fix is that the image is placed AFTER the frame is fully composed, at a
// position found by locating the marker in the finished output, with C=1 so the cursor
// does not move. The frame is solved before an image is mentioned, so an image cannot
// disturb it — the same discipline the toast overlay already uses.
//
// Opt-in via ui.inline_images, which has been a field with nothing reading it since
// D-007. Off by default because a protocol a terminal mis-measures leaves debris on the
// screen that only a redraw clears, and the marker is never wrong.

// imageRows is how tall a drawn figure is, in cells.
//
// Fixed rather than derived from the image: the pane reserves this much space in the
// text, and a height that changed per figure would make the statement reflow as
// pictures arrived.
const imageRows = 12

// imageState is one figure's journey from a URL to pixels on screen.
type imageState struct {
	// id is kitty's handle for the transmitted pixels. Stable per URL so a figure is
	// uploaded once no matter how often it is drawn.
	id int
	// transmitted is the escape that uploads it, emitted once and then cleared.
	transmitted string
	// ready means the pixels are in the terminal and it can be placed by id alone.
	ready bool
	err   error
}

// imagesEnabled reports whether figures should be drawn rather than marked.
func (m Model) imagesEnabled() bool {
	return m.cfg.UI.InlineImages && m.graphics.Available()
}

type imageLoadedMsg struct {
	url string
	png []byte
	err error
}

// fetchPaneImages downloads any figure on screen that is not already in the terminal.
//
// Only what is VISIBLE: a long editorial can reference a dozen figures, and uploading
// all of them to read the first paragraph would stall the pane on a megabyte of PNG
// nobody has scrolled to.
func (m Model) fetchPaneImages() tea.Cmd {
	if !m.imagesEnabled() {
		return nil
	}

	var cmds []tea.Cmd
	for _, img := range m.paneImages() {
		if !img.IsDrawable() {
			continue
		}
		if _, seen := m.images[img.URL]; seen {
			continue
		}
		cmds = append(cmds, loadImage(img.URL))
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func loadImage(url string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return imageLoadedMsg{url: url, err: err}
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return imageLoadedMsg{url: url, err: err}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return imageLoadedMsg{url: url, err: fmt.Errorf("%s", resp.Status)}
		}
		// Capped: this is written to a terminal, and an unbounded read of something
		// that is not the image asked for would be sprayed across the screen.
		data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		if err != nil {
			return imageLoadedMsg{url: url, err: err}
		}
		png, err := render.ToPNG(data)
		return imageLoadedMsg{url: url, png: png, err: err}
	}
}

// handleImageLoaded stores a fetched figure and queues its upload.
func (m Model) handleImageLoaded(msg imageLoadedMsg) (tea.Model, tea.Cmd) {
	if m.images == nil {
		m.images = map[string]*imageState{}
	}
	if msg.err != nil {
		// Recorded rather than raised. A figure that will not load is a marker the
		// user can still press, and interrupting a statement to say so would be
		// louder than the problem.
		m.images[msg.url] = &imageState{err: msg.err}
		return m, nil
	}

	m.nextImageID++
	m.images[msg.url] = &imageState{
		id:          m.nextImageID,
		transmitted: render.KittyTransmit(m.nextImageID, msg.png),
	}
	return m, nil
}

// withImages draws the figures over a finished frame.
//
// Runs LAST, on composed output, for the same reason the toast does: the layout is
// already solved, so nothing here can move it.
func (m Model) withImages(view string) string {
	if !m.imagesEnabled() || len(m.images) == 0 {
		return view
	}

	markers := m.visibleMarkers(view)
	if len(markers) == 0 {
		return view
	}

	var b strings.Builder
	// Clear last frame's placements first. Without this every scroll leaves the old
	// picture behind while the text moves out from under it.
	b.WriteString(render.KittyClear())

	imgs := m.paneImages()
	for _, mk := range markers {
		if mk.n < 1 || mk.n > len(imgs) {
			continue
		}
		st := m.images[imgs[mk.n-1].URL]
		if st == nil || st.err != nil {
			continue
		}

		// Upload once, on the first frame this figure is drawn.
		if !st.ready {
			b.WriteString(st.transmitted)
			st.ready = true
		}

		// Save cursor, move, place, restore. The restore is what keeps this from
		// disturbing whatever Bubbletea writes next.
		fmt.Fprintf(&b, "\x1b7\x1b[%d;%dH", mk.row+1, mk.col+1)
		b.WriteString(render.KittyPlace(st.id, mk.cols, imageRows))
		b.WriteString("\x1b8")
	}

	seq := b.String()
	if m.graphics.InTmux {
		seq = render.Passthrough(seq)
	}
	return view + seq
}

// markerPos is where a figure's marker landed in the finished frame.
type markerPos struct {
	n        int
	row, col int
	cols     int
}

// visibleMarkers finds each numbered marker in the composed output.
//
// Reading the FINISHED frame rather than tracking positions through the layout: the
// pane wraps, scrolls, truncates and re-frames its content, and any position computed
// before all of that would be a guess that drifts. Where the marker actually is, is
// where the picture goes.
func (m Model) visibleMarkers(view string) []markerPos {
	var out []markerPos

	for row, line := range strings.Split(view, "\n") {
		plain := ansi.Strip(line)
		idx, n := findMarker(plain)
		if idx < 0 {
			continue
		}
		out = append(out, markerPos{
			n: n, row: row, col: idx,
			// Bounded by what is left of the pane, so a wide figure cannot spill
			// past the bezel into the column beside it.
			cols: maxInt(minInt(m.detailWidth()-4, 60), 10),
		})
	}
	return out
}

// findMarker locates a numbered marker in one plain line, returning its column and
// number, or -1.
//
// TWO FORMATS, because there are two renderers. The statement writes "[▸ img 3]" and
// the editorial writes "[▸ 3]" — a difference nobody noticed while both were only ever
// read by a human. A parser that knew one of them would have drawn figures on
// statements and silently nothing on editorials, or the reverse.
func findMarker(plain string) (col, n int) {
	idx := strings.Index(plain, markerOpen)
	if idx < 0 {
		return -1, 0
	}
	rest := plain[idx+len(markerOpen):]

	end := strings.IndexByte(rest, ']')
	if end < 0 {
		return -1, 0
	}
	body := strings.TrimSpace(rest[:end])
	body = strings.TrimPrefix(body, "img ")

	// The label after an em dash is prose and is not part of the number.
	if cut := strings.Index(body, " —"); cut >= 0 {
		body = body[:cut]
	}
	if _, err := fmt.Sscanf(strings.TrimSpace(body), "%d", &n); err != nil || n < 1 {
		return -1, 0
	}
	return idx, n
}

// detectGraphics is called once at startup; the result decides whether any of this
// runs at all.
func detectGraphics() config.Graphics { return config.DetectGraphics() }

// markerOpen is the start of a bracket marker, shared with the renderers so the two
// cannot drift apart — a marker a renderer writes and this cannot find is a figure that
// silently never draws.
const markerOpen = "[▸ "
