package config

import (
	"os"
	"strings"
)

// What this terminal can be trusted to draw.
//
// leetui leans on box-drawing and geometric glyphs. Most of that is safe, but the state
// column's `✓` and `⊘` are East-Asian Ambiguous — a terminal may draw them two cells wide,
// and in a grid ruled to the column that shears every row below it.
//
// So the environment is asked before assuming. The signals below are the ones that
// actually predict trouble; anything subtler is guesswork, and guessing wrong in the
// cautious direction only costs an uglier glyph.

// PreferASCII reports whether to fall back to plain ASCII glyphs.
//
// The explicit config flag always wins. Otherwise:
//
//   - a locale that is not UTF-8 — the strongest signal there is, since the terminal is
//     not being told to decode multi-byte characters at all
//   - TERM=linux, the kernel console, whose font is 256 glyphs and has none of these
//   - TERM=dumb or unset, which promises nothing
func PreferASCII(cfg Config) bool {
	if cfg.UI.ASCII {
		return true
	}

	switch strings.ToLower(os.Getenv("TERM")) {
	case "", "dumb", "linux", "vt100", "vt220":
		return true
	}

	// The first locale variable that is set decides, in POSIX precedence order.
	for _, key := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		v := os.Getenv(key)
		if v == "" {
			continue
		}
		low := strings.ToLower(v)
		return !strings.Contains(low, "utf-8") && !strings.Contains(low, "utf8")
	}

	// No locale set at all. macOS terminals often leave these unset while handling UTF-8
	// perfectly, so this is not treated as a failure — the config flag is there for
	// anyone it gets wrong.
	return false
}
