package tui

import (
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Renders real screens to /tmp for the website. Not a mockup: this is the same View()
// the app draws, so a screenshot cannot drift from the product.
func TestCaptureShots(t *testing.T) {
	if os.Getenv("LEETUI_SHOTS") == "" {
		t.Skip("set LEETUI_SHOTS=1 to capture")
	}
	for name, keys := range map[string][]tea.Msg{
		"board":    nil,
		"settings": {key("V")},
		"help":     {key("?")},
	} {
		m := boot(t, true, 100, 26)
		m = drive(t, m, keys...)
		if err := os.WriteFile("/tmp/shot_"+name+".ans", []byte(m.View()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
