package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Default returns the configuration used when no file exists.
func Default() Config {
	home, _ := os.UserHomeDir()
	return Config{
		Workspace:     filepath.Join(home, "leetcode"),
		DefaultLang:   "python3",
		WatchSolution: true,
		UI: UI{
			Mouse: true,
		},
		Sync: Sync{
			// Deliberately conservative. LeetCode publishes no rate limit, so this is
			// a guess tuned to be boring rather than fast; see the open item in
			// docs/DECISIONS.md about determining it empirically.
			RequestsPerSecond:  2,
			PageSize:           100,
			CompanyRefreshDays: 7,
		},
		Git: Git{
			CommitOnAccepted: true,
			AutoPush:         false,
			CommitNotes:      true,
		},
		Keys: map[string]string{},
	}
}

// Dir returns leetui's config directory, creating it if needed.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return "", fmt.Errorf("locate config dir: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	dir := filepath.Join(base, "leetui")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	return dir, nil
}

// DataDir returns where the SQLite store lives, creating it if needed.
func DataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home dir: %w", err)
	}
	dir := filepath.Join(home, ".local", "share", "leetui")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create data dir: %w", err)
	}
	return dir, nil
}
