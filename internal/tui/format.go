package tui

import (
	"fmt"
	"time"
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
