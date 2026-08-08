package config

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// The settable surface, declared once.
//
// Everything that can be changed at runtime is described here — its name, what it means,
// how to read it, and how to write it. The command palette and the settings screen are
// both views onto this table, which is the same discipline D-013 applies to keys: a
// setting edited at one call site cannot be listed anywhere, and a setting listed but not
// wired is worse.
//
// Adding an option means adding a row here. Nothing else needs to know.

// Kind is how a value is entered and shown.
type Kind int

const (
	KindString Kind = iota
	KindBool
	KindInt
	KindFloat
	// KindChoice is a string from a fixed set, so a picker can offer them.
	KindChoice
)

// Setting is one editable option.
type Setting struct {
	// Key is what the user types: `default_lang`, `ui.ascii`, `git.branch`.
	Key  string
	Help string
	Kind Kind

	// Choices lists the accepted values for KindChoice.
	Choices []string

	Get func(*Config) string
	Set func(*Config, string) error
}

// settings is the registry. Grouped in the order a person would look for them.
var settings = []Setting{
	{
		Key: "workspace", Help: "where problem folders are written", Kind: KindString,
		Get: func(c *Config) string { return c.Workspace },
		Set: func(c *Config, v string) error { c.Workspace = v; return nil },
	},
	{
		Key: "default_lang", Help: "language for a problem you have not opened before", Kind: KindString,
		Get: func(c *Config) string { return c.DefaultLang },
		Set: func(c *Config, v string) error { c.DefaultLang = v; return nil },
	},
	{
		Key: "editor", Help: "overrides $EDITOR", Kind: KindString,
		Get: func(c *Config) string { return c.Editor },
		Set: func(c *Config, v string) error { c.Editor = v; return nil },
	},
	{
		Key: "watch_solution", Help: "saving the solution re-runs the tests", Kind: KindBool,
		Get: func(c *Config) string { return boolStr(c.WatchSolution) },
		Set: setBool(func(c *Config, v bool) { c.WatchSolution = v }),
	},
	{
		Key: "editor_pane", Help: "open the editor in a new tmux pane beside leetui", Kind: KindBool,
		Get: func(c *Config) string { return boolStr(c.EditorPane) },
		Set: setBool(func(c *Config, v bool) { c.EditorPane = v }),
	},
	{
		Key: "run_after_edit", Help: "run the tests when the editor exits", Kind: KindBool,
		Get: func(c *Config) string { return boolStr(c.RunAfterEdit) },
		Set: setBool(func(c *Config, v bool) { c.RunAfterEdit = v }),
	},
	{
		Key: "open_statement", Help: "open README.md alongside the solution", Kind: KindBool,
		Get: func(c *Config) string { return boolStr(c.OpenStatement) },
		Set: setBool(func(c *Config, v bool) { c.OpenStatement = v }),
	},

	{
		Key: "ui.show_tags", Help: "show a problem's tags — they give the approach away", Kind: KindBool,
		Get: func(c *Config) string { return boolStr(c.UI.ShowTags) },
		Set: setBool(func(c *Config, v bool) { c.UI.ShowTags = v }),
	},
	{
		Key: "ui.show_hints", Help: "show the problem's hints", Kind: KindBool,
		Get: func(c *Config) string { return boolStr(c.UI.ShowHints) },
		Set: setBool(func(c *Config, v bool) { c.UI.ShowHints = v }),
	},
	{
		Key: "ui.inline_images", Help: "draw figures in the pane, not just a marker", Kind: KindBool,
		Get: func(c *Config) string { return boolStr(c.UI.InlineImages) },
		Set: setBool(func(c *Config, v bool) { c.UI.InlineImages = v }),
	},
	{
		Key: "ui.ascii", Help: "draw with ASCII instead of box-drawing characters", Kind: KindBool,
		Get: func(c *Config) string { return boolStr(c.UI.ASCII) },
		Set: setBool(func(c *Config, v bool) { c.UI.ASCII = v }),
	},
	{
		Key: "ui.reduce_motion", Help: "settle the flip instantly", Kind: KindBool,
		Get: func(c *Config) string { return boolStr(c.UI.ReduceMotion) },
		Set: setBool(func(c *Config, v bool) { c.UI.ReduceMotion = v }),
	},
	{
		Key: "ui.mouse", Help: "respond to the mouse", Kind: KindBool,
		Get: func(c *Config) string { return boolStr(c.UI.Mouse) },
		Set: setBool(func(c *Config, v bool) { c.UI.Mouse = v }),
	},

	{
		Key: "sync.requests_per_second", Help: "how hard to hit leetcode.com", Kind: KindFloat,
		Get: func(c *Config) string { return strconv.FormatFloat(c.Sync.RequestsPerSecond, 'g', -1, 64) },
		Set: func(c *Config, v string) error {
			f, err := strconv.ParseFloat(v, 64)
			if err != nil || f <= 0 {
				return fmt.Errorf("requests_per_second wants a positive number, got %q", v)
			}
			c.Sync.RequestsPerSecond = f
			return nil
		},
	},

	{
		Key: "git.commit_on_accepted", Help: "an accepted verdict commits itself", Kind: KindBool,
		Get: func(c *Config) string { return boolStr(c.Git.CommitOnAccepted) },
		Set: setBool(func(c *Config, v bool) { c.Git.CommitOnAccepted = v }),
	},
	{
		Key: "git.commit_notes", Help: "commit notes.md alongside the solution", Kind: KindBool,
		Get: func(c *Config) string { return boolStr(c.Git.CommitNotes) },
		Set: setBool(func(c *Config, v bool) { c.Git.CommitNotes = v }),
	},
	{
		Key: "git.branch", Help: "refuse to commit anywhere else; empty means whatever is checked out", Kind: KindString,
		Get: func(c *Config) string { return c.Git.Branch },
		Set: func(c *Config, v string) error { c.Git.Branch = v; return nil },
	},
	{
		Key: "git.remote", Help: "where a push goes; empty means the branch's own upstream", Kind: KindString,
		Get: func(c *Config) string { return c.Git.Remote },
		Set: func(c *Config, v string) error { c.Git.Remote = v; return nil },
	},
}

// Settings returns the registry.
func Settings() []Setting { return settings }

// FindSetting looks one up by key.
//
// Also accepts a `keys.<action>` key, which is not in the table because the actions are
// declared in the keymap and duplicating them here would be a second list to keep in
// step (D-013).
func FindSetting(key string) (Setting, bool) {
	key = strings.TrimSpace(key)
	for _, s := range settings {
		if s.Key == key {
			return s, true
		}
	}
	if action, ok := strings.CutPrefix(key, "keys."); ok {
		if _, known := DefaultKeymap[action]; known {
			return keySetting(action), true
		}
	}
	return Setting{}, false
}

// keySetting builds a Setting for one keybinding on demand.
func keySetting(action string) Setting {
	return Setting{
		Key:  "keys." + action,
		Help: "the key bound to " + action,
		Kind: KindString,
		Get: func(c *Config) string {
			if k, ok := c.Keys[action]; ok {
				return k
			}
			return DefaultKeymap[action]
		},
		Set: func(c *Config, v string) error {
			v = strings.TrimSpace(v)
			if v == "" {
				return fmt.Errorf("a key cannot be empty; use the default with :set keys.%s %s",
					action, DefaultKeymap[action])
			}
			// A duplicate binding silently shadows one of the two actions, and which
			// one wins depends on map order — so it is refused rather than resolved.
			//
			// Actions() is keyed by KEY, not by action: it is the lookup the event loop
			// does on every keypress.
			if owner, taken := c.Actions()[v]; taken && owner != action {
				return fmt.Errorf("%q is already bound to %s", v, owner)
			}
			if c.Keys == nil {
				c.Keys = map[string]string{}
			}
			c.Keys[action] = v
			return nil
		},
	}
}

// SettingKeys lists every settable key, including the per-action bindings, sorted.
//
// Used for completion and for the settings screen, so a key that can be set is always a
// key that can be found.
func SettingKeys() []string {
	out := make([]string, 0, len(settings)+len(DefaultKeymap))
	for _, s := range settings {
		out = append(out, s.Key)
	}
	for action := range DefaultKeymap {
		out = append(out, "keys."+action)
	}
	sort.Strings(out)
	return out
}

func boolStr(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

// setBool accepts the spellings people actually type.
func setBool(assign func(*Config, bool)) func(*Config, string) error {
	return func(c *Config, raw string) error {
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "true", "yes", "on", "1":
			assign(c, true)
		case "false", "no", "off", "0":
			assign(c, false)
		case "":
			// `:set watch_solution` with no value toggles, which is what a bare flag
			// reads as and saves typing the value you are trying to change.
			return errToggle
		default:
			return fmt.Errorf("wants true or false, got %q", raw)
		}
		return nil
	}
}

// errToggle asks the caller to flip the current value. Returned rather than handled here
// because the setter cannot read.
var errToggle = fmt.Errorf("toggle")

// Apply sets one key, handling the bare-name toggle for booleans.
func (c *Config) Apply(key, value string) error {
	s, ok := FindSetting(key)
	if !ok {
		return fmt.Errorf("no setting called %q", key)
	}
	err := s.Set(c, value)
	if err == errToggle {
		return s.Set(c, boolStr(s.Get(c) != "true"))
	}
	if err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	return nil
}
