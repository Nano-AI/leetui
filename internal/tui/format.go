package tui

import (
	"fmt"
	"time"

	"github.com/Nano-AI/leetui/internal/tui/theme"
)

// ---------------------------------------------------------------------------
// Small helpers
// ---------------------------------------------------------------------------

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ---------------------------------------------------------------------------
// Formatting helpers
// ---------------------------------------------------------------------------

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	mi := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, mi, s)
}

func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	return string(r[:w-1]) + "…"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// sep1 and sep2 are the inline separators, spaced for their two uses: sep1 between items
// in a list, sep2 between key hints where the extra air keeps pairs from merging.
//
// Functions rather than constants because theme.ASCII is decided at startup and a
// package-level var here would be initialised before it (D-026).
func sep1() string { return " " + theme.Chars().Bullet + " " }
func sep2() string { return "  " + theme.Chars().Bullet + "  " }
