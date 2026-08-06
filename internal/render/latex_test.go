package render

import (
	"strings"
	"testing"
)

func TestLaTeXApproximation(t *testing.T) {
	cases := []struct{ in, want string }{
		{`$O(n \log n)$`, "O(n log n)"},
		{`\(1 \leq n \leq 10^5\)`, "1 ≤ n ≤ 10⁵"},
		{`x \ge 0`, "x ≥ 0"},
		{`a \times b`, "a × b"},
		{`\sum_{i=1}^{n}`, "Σ"},
		{`n_1`, "n₁"},
		{`\infty`, "∞"},
		{`10^{12}`, "10¹²"},
		{`\text{count}`, "count"},
		{"no math here", "no math here"},
	}
	for _, c := range cases {
		if got := approximateLaTeX(c.in); !strings.Contains(got, c.want) {
			t.Errorf("approximateLaTeX(%q) = %q, want it to contain %q", c.in, got, c.want)
		}
	}
}

// TestSuperscriptFallback asserts we never silently drop an exponent we cannot render.
func TestSuperscriptFallback(t *testing.T) {
	if got := toSuperscript("k"); got != "^k" {
		t.Errorf("toSuperscript(\"k\") = %q, want %q — dropping it would change the meaning", got, "^k")
	}
	if got := toSuperscript("5"); got != "⁵" {
		t.Errorf("toSuperscript(\"5\") = %q, want ⁵", got)
	}
}
