package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Nano-AI/leetui/internal/config"
	"github.com/Nano-AI/leetui/internal/render"
	"github.com/Nano-AI/leetui/internal/solve"
)

// `leetui image <problem> [n]` — draw a figure in the terminal.
//
// The statement's figures are numbered markers ("[▸ img 1 — tree]"), and pressing the
// number in the app opens the browser. That is the floor and it works everywhere. This is
// the ceiling for a terminal that can actually show the picture.
//
// DELIBERATELY A COMMAND, NOT A PANE. An image escape occupies cells that the layout has
// to have reserved, and a graphics protocol the terminal mis-measures shears the frame —
// the same failure mode as an unwrapped long line, but harder to see coming and impossible
// to recover from without quitting. Here there is no frame to shear: the picture is drawn
// into a plain terminal, at a size the caller chose.
//
// Which means it composes the way everything else does — put it in the pane beside the
// app, the way `leetui run --watch` goes in one.

const imageMaxBytes = 8 << 20 // 8 MiB; LeetCode's diagrams are tens of kilobytes

func runImage(a *app, args []string) (int, error) {
	fs, _ := flags("image")
	cols := fs.Int("cols", 60, "cell width to draw within")
	rows := fs.Int("rows", 20, "cell height to draw within")
	editorial := fs.Bool("editorial", false, "take the figure from the editorial (premium)")
	rest, err := parseFlags(fs, args)
	if err != nil {
		return exitProblem, err
	}

	arg := first(rest)
	which := 1
	if len(rest) > 1 {
		if n, err := strconv.Atoi(rest[len(rest)-1]); err == nil {
			which = n
			arg = rest[0]
		}
	}

	g := config.DetectGraphics()
	if g.Protocol == config.ProtocolNone {
		return exitProblem, fmt.Errorf(
			"%s cannot draw images; open the figure with `leetui open` instead", g.Terminal)
	}
	if g.Blocked {
		// The silent failure this exists to prevent: tmux drops the escape and nothing
		// anywhere reports it, so the user concludes the feature is broken.
		return exitProblem, fmt.Errorf("%s", g.Fix)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	d, err := a.problem(ctx, arg)
	if err != nil {
		return exitProblem, err
	}

	// The statement first, then the editorial. Most figures are in the EDITORIAL —
	// the statement usually has none, and asking for a figure only to be told the
	// problem has none, while looking at three of them, is the wrong answer.
	images := solve.StatementImages(d)
	where := "statement"
	if *editorial || len(images) == 0 {
		if ed, err := a.sync.Editorial(ctx, d.Slug, false); err == nil && ed != nil {
			if figs := render.EditorialToMarkdown(ed.Content).Images; len(figs) > 0 {
				images, where = figs, "editorial"
			}
		}
	}
	if len(images) == 0 {
		return exitProblem, fmt.Errorf("%s has no figures in its statement or editorial", d.Slug)
	}
	if which < 1 || which > len(images) {
		return exitProblem, fmt.Errorf("%s has %d figures; asked for %d",
			d.Slug, len(images), which)
	}

	img := images[which-1]
	fmt.Fprintf(os.Stderr, "%d. %s — %s figure %d of %d%s\n",
		d.NumericID, d.Title, where, which, len(images), altNote(img.Alt))

	// A playground or a video is a real marker with a real number — pressing it in the
	// app opens a browser, which is right. It is just not a picture, and trying to
	// draw one fails as a bare 403 that explains nothing.
	if !img.IsDrawable() {
		return exitProblem, fmt.Errorf("figure %d of %s is a %s, not an image — open it:\n    %s",
			which, d.Slug, mediaKind(img.URL), img.URL)
	}

	a.log.Printf("image fetch %s", img.URL)
	data, err := fetchImage(ctx, img.URL)
	if err != nil {
		// Name the URL: a 403 or a 404 is about THAT asset, and without it the user
		// cannot tell a broken resolver from a figure LeetCode does not serve.
		return exitProblem, fmt.Errorf("%s: %w", img.URL, err)
	}
	a.log.Printf("image ok %d bytes", len(data))

	var seq string
	if g.Protocol == config.ProtocolITerm {
		// iTerm2 takes whatever the OS can decode, so a JPEG goes straight through.
		seq = render.ITermImage(data, *cols, *rows)
	} else {
		// Kitty takes PNG or raw pixels. LeetCode serves JPEGs, and handing one over
		// draws nothing and reports nothing.
		png, err := render.ToPNG(data)
		if err != nil {
			// Name the URL. "unknown format" alone leaves the user with no way to
			// tell a broken decoder from a figure LeetCode serves as something else,
			// or from an HTML error page arriving with a 200.
			return exitProblem, fmt.Errorf("%s: %w", img.URL, err)
		}
		seq = render.KittyImage(png, *cols, *rows)
	}
	if g.InTmux {
		seq = render.Passthrough(seq)
	}

	fmt.Print(seq + "\n")
	return exitOK, nil
}

// fetchImage downloads one figure.
//
// Capped, because the response is written straight to a terminal: an unbounded read of
// something that is not the image we asked for would spray it across the screen.
// altNote appends a figure's description when it has one worth printing.
// mediaKind names what a non-drawable marker is, for the message that says so.
func mediaKind(url string) string {
	switch {
	case strings.Contains(url, "/playground/"):
		return "code playground"
	case strings.Contains(url, "vimeo"), strings.Contains(url, "youtube"):
		return "video"
	default:
		return "web embed"
	}
}

func altNote(alt string) string {
	if alt == "" || alt == "figure" {
		return ""
	}
	return ": " + alt
}

func fetchImage(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch figure: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch figure: %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, imageMaxBytes))
	if err != nil {
		return nil, fmt.Errorf("read figure: %w", err)
	}
	return data, nil
}
