package render

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"strings"

	// Registered for their decoders only. LeetCode serves .jpg and .gif alongside
	// .png, and kitty's protocol takes PNG or raw pixels and nothing else.
	_ "image/gif"
	_ "image/jpeg"
)

// ToPNG re-encodes an image so kitty can draw it.
//
// Kitty's f=100 means PNG. LeetCode serves plenty of JPEGs, and handing one over with
// f=100 produces no picture and no error — the terminal simply fails to decode and
// nothing appears, which is the silent failure this whole path is trying to avoid.
//
// A PNG passes through untouched: re-encoding one would cost time and lose nothing.
func ToPNG(data []byte) ([]byte, error) {
	if bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		return data, nil
	}
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, fmt.Errorf("re-encode %s as png: %w", format, err)
	}
	return out.Bytes(), nil
}

// Drawing a figure in the terminal.
//
// Statements are full of diagrams — a tree, a grid, the array after each step — and the
// numbered marker ("[▸ img 1 — tree]") is the floor, not the ceiling. It works everywhere,
// which is why it stays the default and the fallback, but a terminal that can show the
// picture should.
//
// Two protocols cover everything in practice:
//
//   - Kitty's, which WezTerm and Ghostty also speak. Chunked base64 in an APC sequence.
//   - iTerm2's, a single OSC 1337 with the file inline.
//
// Both are opt-in (D-007). A protocol a terminal does not understand prints as garbage
// across the statement, and the cost of guessing wrong is a screen the user has to fix by
// quitting — far worse than a marker they can press a number on.

// KittyChunk is the payload size per escape.
//
// Kitty's protocol requires the base64 be split, and 4096 is the documented chunk size.
// Larger works on kitty itself and breaks through tmux, which has its own buffer limits.
const KittyChunk = 4096

// KittyImage encodes a PNG for kitty's graphics protocol.
//
// cols and rows bound the drawing to a cell box, so a diagram cannot push the statement
// off the screen — the terminal scales to fit.
func KittyImage(png []byte, cols, rows int) string {
	payload := base64.StdEncoding.EncodeToString(png)

	var b strings.Builder
	for i := 0; i < len(payload); i += KittyChunk {
		end := min(i+KittyChunk, len(payload))
		chunk := payload[i:end]

		more := 1
		if end == len(payload) {
			more = 0
		}

		if i == 0 {
			// f=100: the payload is a PNG, so the terminal does the decoding.
			// a=T: transmit and display in one go.
			// c/r: the cell box to fit within.
			fmt.Fprintf(&b, "\x1b_Gf=100,a=T,c=%d,r=%d,m=%d;%s\x1b\\", cols, rows, more, chunk)
			continue
		}
		fmt.Fprintf(&b, "\x1b_Gm=%d;%s\x1b\\", more, chunk)
	}
	return b.String()
}

// ITermImage encodes a PNG for iTerm2's protocol.
//
// One sequence, no chunking. `inline=1` draws it rather than offering a download, and
// preserveAspectRatio keeps a tree diagram from being squashed into its box.
func ITermImage(png []byte, cols, rows int) string {
	return fmt.Sprintf("\x1b]1337;File=inline=1;width=%d;height=%d;preserveAspectRatio=1;size=%d:%s\a",
		cols, rows, len(png), base64.StdEncoding.EncodeToString(png))
}

// Passthrough wraps an escape so it survives tmux.
//
// tmux eats sequences it does not recognise unless `allow-passthrough` is on, and even
// then the payload must be wrapped in its own DCS and every ESC inside it doubled —
// otherwise tmux terminates the wrapper at the first one and prints the rest as text.
//
// `leetui doctor` reports when passthrough is off, because the failure is otherwise
// completely silent: no error anywhere, just a figure that never appears.
func Passthrough(seq string) string {
	return "\x1bPtmux;" + strings.ReplaceAll(seq, "\x1b", "\x1b\x1b") + "\x1b\\"
}
