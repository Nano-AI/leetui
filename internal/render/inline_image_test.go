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
	if strings.Contains(out[7:len(out)-2], "\x1b") && !strings.Contains(out, "\x1b\x1b") {
		t.Error("inner escapes were not doubled; tmux would cut the payload short")
	}
	if !strings.HasSuffix(out, "\x1b\\") {
		t.Error("unterminated DCS")
	}
}

// TestToPNGConvertsJPEG is the bug running it found: LeetCode serves plenty of JPEGs,
// and kitty's f=100 means PNG. Handing over a JPEG draws nothing and reports nothing —
// exactly the silent failure this path exists to avoid.
func TestToPNGConvertsJPEG(t *testing.T) {
	var jpg bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	if err := jpeg.Encode(&jpg, img, nil); err != nil {
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
