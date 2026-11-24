package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReportEditor(t *testing.T) {
	dir := t.TempDir()
	for _, b := range []string{"nvim", "my-ed"} {
		if err := os.WriteFile(filepath.Join(dir, b), nil, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)

	for _, tc := range []struct {
		configured, want string
		problems         int
	}{
		{"/opt/homebrew/bin/nvim", "Neovim", 0},
		{"my-ed", "my-ed", 0},
		{"/gone/my-ed", "/gone/my-ed not found", 1},
	} {
		var out strings.Builder
		if n := reportEditor(&out, tc.configured, "config.toml"); n != tc.problems ||
			!strings.Contains(out.String(), tc.want) {
			t.Errorf("reportEditor(%q) = %d, %q; want %d, %q",
				tc.configured, n, out.String(), tc.problems, tc.want)
		}
	}
}
