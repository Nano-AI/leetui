package config

import "testing"

func TestApplySetsAndToggles(t *testing.T) {
	c := Default()

	if err := c.Apply("default_lang", "go"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if c.DefaultLang != "go" {
		t.Errorf("default_lang = %q", c.DefaultLang)
	}

	// A bare name toggles a boolean — a flag with no value reads as "flip this", and it
	// saves typing the value you are trying to change.
	before := c.WatchSolution
	if err := c.Apply("watch_solution", ""); err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if c.WatchSolution == before {
		t.Error("a bare boolean did not toggle")
	}

	if err := c.Apply("ui.show_tags", "on"); err != nil {
		t.Fatalf("set bool: %v", err)
	}
	if !c.UI.ShowTags {
		t.Error(`"on" was not accepted as true`)
	}
}

// Tags name the technique. Printing them beside the problem answers the question the
// problem is asking, and it is not a choice — you cannot un-see it.
func TestTagsAndHintsAreHiddenByDefault(t *testing.T) {
	c := Default()
	if c.UI.ShowTags {
		t.Error("tags are shown by default; they give the approach away")
	}
	if c.UI.ShowHints {
		t.Error("hints are shown by default")
	}
}

func TestApplyRejectsUnknownAndBadValues(t *testing.T) {
	c := Default()
	if err := c.Apply("nonsense", "1"); err == nil {
		t.Error("an unknown key was accepted")
	}
	if err := c.Apply("watch_solution", "maybe"); err == nil {
		t.Error(`"maybe" was accepted as a boolean`)
	}
	if err := c.Apply("sync.requests_per_second", "-2"); err == nil {
		t.Error("a negative rate was accepted")
	}
}

// A binding can be changed, but not onto a key another action already owns: a duplicate
// silently shadows one of the two, and which one wins depends on map order.
func TestKeyBindingsAreSettableAndUnique(t *testing.T) {
	c := Default()

	if err := c.Apply("keys.run", "x"); err != nil {
		t.Fatalf("rebind: %v", err)
	}
	if c.Actions()["x"] != "run" {
		t.Errorf("x is bound to %q, want run", c.Actions()["x"])
	}

	if err := c.Apply("keys.submit", "x"); err == nil {
		t.Error("two actions were allowed onto the same key")
	}
}

// Every key that can be set must be findable, or the palette can offer something it
// cannot then apply.
func TestEverySettableKeyResolves(t *testing.T) {
	for _, key := range SettingKeys() {
		if _, ok := FindSetting(key); !ok {
			t.Errorf("SettingKeys lists %q but FindSetting cannot resolve it", key)
		}
	}
}
