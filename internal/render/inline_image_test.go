package render

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

// A protocol a terminal does not understand prints as garbage across the statement, and
// the user has to quit to fix it. So the encoding is pinned rather than eyeballed.

func TestKittyChunksAndTerminates(t *testing.T) {
	// Two chunks' worth, so the continuation path is exercised rather than assumed.
	png := make([]byte, KittyChunk*2)
	for i := range png {
		png[i] = byte(i)
	}

	out := KittyImage(png, 40, 20)

	if !strings.HasPrefix(out, "\x1b_Gf=100,a=T,c=40,r=20,m=1;") {
		t.Errorf("first chunk header is wrong: %q", out[:60])
	}
	if strings.Count(out, "\x1b_G") < 2 {
		t.Error("a payload over one chunk was not split")
	}
	// The LAST chunk must say m=0 or the terminal waits forever for more and draws
	// nothing at all.
	if !strings.Contains(out, "\x1b_Gm=0;") {
		t.Error("no final chunk marker; the terminal would never draw")
	}
	if !strings.HasSuffix(out, "\x1b\\") {
		t.Error("unterminated escape — the rest of the statement would be swallowed")
	}
}

func TestKittySingleChunk(t *testing.T) {
	out := KittyImage([]byte("small"), 10, 5)
	if strings.Count(out, "\x1b_G") != 1 {
		t.Errorf("a small payload was split: %q", out)
	}
	if !strings.Contains(out, "m=0;") {
		t.Error("a single chunk must still be marked final")
	}
	if !strings.Contains(out, base64.StdEncoding.EncodeToString([]byte("small"))) {
		t.Error("payload missing")
	}
}

func TestITermCarriesTheSize(t *testing.T) {
	png := []byte("0123456789")
	out := ITermImage(png, 30, 15)

	// iTerm needs the byte count up front; without it the image is dropped silently.
	if !strings.Contains(out, "size=10:") {
		t.Errorf("no size in %q", out)
	}
	if !strings.Contains(out, "inline=1") {
		t.Error("without inline=1 iTerm offers a download instead of drawing")
	}
	if !strings.HasSuffix(out, "\a") {
		t.Error("unterminated OSC")
	}
}

// tmux terminates its wrapper at the first raw ESC, so every ESC inside has to be
// doubled. Getting this wrong prints the rest of the payload as text across the pane.
func TestPassthroughDoublesEscapes(t *testing.T) {
	out := Passthrough("\x1b_Gm=0;abc\x1b\\")

	if !strings.HasPrefix(out, "\x1bPtmux;") {
		t.Errorf("not wrapped: %q", out)
	}
	if !strings.Contains(out, "\x1b\x1b_G") {
		t.Error("inner escapes were not doubled; tmux would cut the payload short")
	}
	if !strings.HasSuffix(out, "\x1b\\") {
		t.Error("unterminated DCS")
	}
}

// TestPassthroughWrapsEachChunk is the bug that made images silently fail in tmux.
//
// The first version put the whole image in ONE DCS. That works for a thumbnail and
// fails for anything real: a 100 KB PNG is about 33 chunks, tmux caps what a single DCS
// may carry, and the overflow is dropped with no error and no picture.
func TestPassthroughWrapsEachChunk(t *testing.T) {
	// Three chunks' worth, so the sequence genuinely has several escapes.
	png := make([]byte, KittyChunk*3)
	seq := KittyImage(png, 40, 20)

	chunks := strings.Count(seq, "\x1b_G")
	if chunks < 3 {
		t.Fatalf("fixture produced %d chunks, want at least 3", chunks)
	}

	out := Passthrough(seq)
	wrappers := strings.Count(out, "\x1bPtmux;")
	if wrappers != chunks {
		t.Errorf("%d chunks got %d tmux wrappers — tmux would drop the overflow",
			chunks, wrappers)
	}

	// No wrapper may contain a raw (undoubled) ESC, or tmux ends it early.
	for _, part := range strings.Split(out, "\x1bPtmux;")[1:] {
		body := strings.TrimSuffix(part, "\x1b\\")
		if strings.Contains(strings.ReplaceAll(body, "\x1b\x1b", ""), "\x1b") {
			t.Error("a wrapper carries an undoubled ESC")
		}
	}
}

// iTerm sends ONE sequence terminated by BEL rather than ST, and it must still be
// wrapped exactly once — not split, not dropped.
func TestPassthroughHandlesITerm(t *testing.T) {
	out := Passthrough(ITermImage([]byte("data"), 10, 5))
	if n := strings.Count(out, "\x1bPtmux;"); n != 1 {
		t.Errorf("iTerm sequence got %d wrappers, want 1", n)
	}
	if !strings.HasSuffix(out, "\x1b\\") {
		t.Error("unterminated DCS")
	}
}

// A sequence with no recognised terminator is passed through whole rather than
// discarded — silently dropping one would be the worst possible failure here.
func TestSplitEscapesKeepsAnUnterminatedTail(t *testing.T) {
	parts := splitEscapes("\x1b_Gm=1;a\x1b\\\x1b_Gm=0;b")
	if len(parts) != 2 {
		t.Fatalf("got %d parts, want 2: %q", len(parts), parts)
	}
	if parts[1] != "\x1b_Gm=0;b" {
		t.Errorf("tail was mangled: %q", parts[1])
	}
}

// TestToPNGConvertsJPEG is the bug running it found: LeetCode serves plenty of JPEGs,
// and kitty's f=100 means PNG. Handing over a JPEG draws nothing and reports nothing —
// exactly the silent failure this path exists to avoid.
func TestToPNGConvertsJPEG(t *testing.T) {
	var jpg bytes.Buffer
	if err := jpeg.Encode(&jpg, image.NewRGBA(image.Rect(0, 0, 8, 8)), nil); err != nil {
		t.Fatalf("build fixture: %v", err)
	}
	if bytes.HasPrefix(jpg.Bytes(), pngMagic) {
		t.Fatal("fixture is not a JPEG")
	}

	out, err := ToPNG(jpg.Bytes())
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if !bytes.HasPrefix(out, pngMagic) {
		t.Errorf("output is not a PNG: %x", out[:8])
	}
}

// A PNG passes through untouched — re-encoding costs time and gains nothing.
func TestToPNGLeavesAPNGAlone(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	in := buf.Bytes()

	out, err := ToPNG(in)
	if err != nil {
		t.Fatalf("convert: %v", err)
	}
	if !bytes.Equal(in, out) {
		t.Error("a PNG was re-encoded")
	}
}

func TestToPNGRejectsGarbage(t *testing.T) {
	if _, err := ToPNG([]byte("this is an HTML error page")); err == nil {
		t.Error("non-image data was accepted; it would be sprayed at the terminal")
	}
}

var pngMagic = []byte("\x89PNG\r\n\x1a\n")
