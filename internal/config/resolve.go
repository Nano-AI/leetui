package config

import "fmt"

// Keymap returns the effective bindings: defaults with the user's overrides applied.
func (c Config) Keymap() map[string]string {
	km := make(map[string]string, len(DefaultKeymap))
	for action, key := range DefaultKeymap {
		km[action] = key
	}
	for action, key := range c.Keys {
		if _, known := DefaultKeymap[action]; known && key != "" {
			km[action] = key
		}
	}
	return km
}

// Actions returns the reverse mapping, key -> action, for dispatch in the TUI.
//
// If the user binds two actions to the same key, the result is that only one of them is
// reachable. Conflicts are reported by Validate rather than silently resolved.
func (c Config) Actions() map[string]string {
	out := make(map[string]string, len(DefaultKeymap))
	for action, key := range c.Keymap() {
		out[key] = action
	}
	return out
}

// Validate reports configuration problems worth surfacing to the user. It returns nil
// when the config is usable — these are warnings, not load failures.
func (c Config) Validate() []string {
	var problems []string

	for action := range c.Keys {
		if _, known := DefaultKeymap[action]; !known {
			problems = append(problems, fmt.Sprintf("keys.%s: unknown action", action))
		}
	}

	seen := map[string]string{}
	for action, key := range c.Keymap() {
		if other, dup := seen[key]; dup {
			problems = append(problems,
				fmt.Sprintf("key %q is bound to both %q and %q", key, other, action))
			continue
		}
		seen[key] = action
	}

	if c.Sync.RequestsPerSecond > 10 {
		problems = append(problems,
			"sync.requests_per_second above 10 risks a rate limit or an account flag")
	}
	if c.Git.AutoPush {
		problems = append(problems,
			"git.auto_push is on — accepted solutions will be published without confirmation")
	}
	return problems
}
