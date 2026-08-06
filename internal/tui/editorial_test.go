package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

func cacheEditorial(t *testing.T, m Model, slug string, sol *leetcode.Solution) {
	t.Helper()
	if err := m.store.SetEditorial(context.Background(), slug, sol); err != nil {
		t.Fatalf("cache editorial: %v", err)
	}
}

func TestEditorialTogglesTheDetailPane(t *testing.T) {
	m := boot(t, true, 120, 32)
	cacheEditorial(t, m, "two-sum", &leetcode.Solution{
		ID: "7", Title: "Two Sum", CanSeeDetail: true,
		Content: "## Approach 1: Hash Table\n\nStore each complement as you go.\n",
	})

	m = drive(t, m, key("d"))
	if !m.showEditorial {
		t.Fatal("d did not open the editorial")
	}

	out := m.View()
	if !strings.Contains(out, "EDITORIAL") {
		t.Error("the pane still calls itself the problem")
	}
	if !strings.Contains(out, "Approach 1") {
		t.Errorf("the editorial body is not on screen:\n%s", out)
	}

	// d again returns to the statement rather than opening a second pane.
	m = drive(t, m, key("d"))
	if m.showEditorial {
		t.Error("d did not toggle back to the statement")
	}
}

// TestEditorialLockNamesWhatIsWithheld is the D-006 contract: a gate must say what it is
// hiding and how to open it, never show a blank pane or a raw error.
func TestEditorialLockNamesWhatIsWithheld(t *testing.T) {
	m := boot(t, true, 120, 32)
	cacheEditorial(t, m, "two-sum", &leetcode.Solution{
		ID: "7", Title: "Two Sum", Content: "",
		PaidOnly: true, CanSeeDetail: false, HasVideoSolution: true,
	})

	m = drive(t, m, key("d"))
	out := m.View()

	for _, want := range []string{"is Premium", "video walkthrough", "sign in", "open it in your browser"} {
		if !strings.Contains(out, want) {
			t.Errorf("the lock state is missing %q:\n%s", want, out)
		}
	}
}

// TestEditorialFollowsTheCursor covers the pane staying open across a move: someone
// reading editorials down a list wants the next one, not the previous problem's.
func TestEditorialFollowsTheCursor(t *testing.T) {
	m := boot(t, true, 120, 32)
	cacheEditorial(t, m, "two-sum", &leetcode.Solution{
		ID: "7", Title: "Two Sum", CanSeeDetail: true,
		Content: "## Hash Table\n\nOne pass.\n",
	})

	m = drive(t, m, key("d"))
	if m.editorial == nil || m.editorial.Slug != "two-sum" {
		t.Fatalf("editorial is %+v, want two-sum", m.editorial)
	}

	m = drive(t, m, key("down"))
	if !m.showEditorial {
		t.Error("moving the cursor closed the editorial pane")
	}
	if m.editorial != nil {
		t.Errorf("the previous problem's editorial is still loaded: %+v", m.editorial)
	}
	if strings.Contains(m.View(), "One pass") {
		t.Error("one problem's editorial is showing under another problem's heading")
	}
}

// TestMarkersFollowThePane guards the number keys: the statement and the editorial have
// different marker lists, and 2 must open the one on screen.
func TestMarkersFollowThePane(t *testing.T) {
	m := boot(t, true, 120, 32)
	cacheEditorial(t, m, "two-sum", &leetcode.Solution{
		ID: "7", Title: "Two Sum", CanSeeDetail: true,
		Content: `Read this.

<iframe src="https://leetcode.com/playground/abc/shared"></iframe>
`,
	})

	m = drive(t, m, key("d"))
	if got := len(m.paneImages()); got != 1 {
		t.Fatalf("editorial has %d markers, want 1", got)
	}
	if url := m.paneImages()[0].URL; !strings.Contains(url, "playground") {
		t.Errorf("marker 1 points at %q, want the playground embed", url)
	}

	m = drive(t, m, key("d"))
	if got := len(m.paneImages()); got != 0 {
		t.Errorf("back on the statement, %d editorial markers are still live", got)
	}
}
