package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Default returns the configuration used when no file exists.
func Default() Config {
	home, _ := os.UserHomeDir()
	return Config{
		Workspace:     filepath.Join(home, "leetcode"),
		DefaultLang:   "python3",
		WatchSolution: true,
		EditorPane:    true,
		RunAfterEdit:  true,
		OpenStatement: true,
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

// DirEnv overrides where the config lives.
//
// It exists because a test that calls Save() writes to the REAL config otherwise, and one
// did: the suite persisted a t.TempDir() workspace into a developer's own config.toml and
// blanked their editor with it. A process that can be told where its state lives is one
// that can be tested without touching anybody's.
//
// Also fair for keeping more than one profile.
const DirEnv = "LEETUI_CONFIG_DIR"

// Dir returns leetui's config directory, creating it if needed.
func Dir() (string, error) {
	if override := strings.TrimSpace(os.Getenv(DirEnv)); override != "" {
		if err := os.MkdirAll(override, 0o755); err != nil {
			return "", fmt.Errorf("create %s: %w", DirEnv, err)
		}
		return override, nil
	}

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
