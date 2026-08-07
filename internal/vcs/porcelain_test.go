package vcs

import "testing"

// Parser tests run on captured output rather than a live repository, so the awkward
// states — detached HEAD, a rename, a branch behind its upstream — are reachable without
// building a repository that is genuinely in each of them.

func nul(lines ...string) string {
	var out string
	for _, l := range lines {
		out += l + "\x00"
	}
	return out
}

func TestParsePorcelainReadsTheBranchHeader(t *testing.T) {
	st := parsePorcelain(nul(
		"# branch.oid 1a2b3c4",
		"# branch.head main",
		"# branch.upstream origin/main",
		"# branch.ab +3 -1",
	))

	if st.Branch != "main" || st.Detached {
		t.Errorf("branch = %q detached=%v, want main", st.Branch, st.Detached)
	}
	if st.Upstream != "origin/main" {
		t.Errorf("upstream = %q", st.Upstream)
	}
	// Behind arrives as "-1"; it is a count, not a signed offset.
	if st.Ahead != 3 || st.Behind != 1 {
		t.Errorf("ahead/behind = %d/%d, want 3/1", st.Ahead, st.Behind)
	}
	if !st.Unpushed() {
		t.Error("3 commits ahead should report unpushed work")
	}
}

func TestParsePorcelainDetachedHead(t *testing.T) {
	st := parsePorcelain(nul("# branch.head (detached)"))
	if !st.Detached {
		t.Fatal("(detached) was not recognised")
	}
	if st.Branch != "" {
		t.Errorf("branch = %q, want empty when detached", st.Branch)
	}
	// Nothing to push from a detached HEAD, and claiming otherwise would send the user
	// looking for a branch that does not exist.
	if st.Unpushed() {
		t.Error("a detached HEAD should not report unpushed work")
	}
}

func TestParsePorcelainClassifiesEntries(t *testing.T) {
	st := parsePorcelain(nul(
		"# branch.head main",
		"1 M. N... 100644 100644 100644 aaa bbb 0001-two-sum/solution.go",
		"1 .M N... 100644 100644 100644 ccc ddd 0002-add-two/solution.py",
		"1 MM N... 100644 100644 100644 eee fff 0003-x/notes.md",
		"? 0004-y/solution.rs",
	))

	if len(st.Changes) != 4 {
		t.Fatalf("got %d changes, want 4", len(st.Changes))
	}
	for _, tc := range []struct {
		path                        string
		staged, unstaged, untracked bool
	}{
		{"0001-two-sum/solution.go", true, false, false},
		{"0002-add-two/solution.py", false, true, false},
		{"0003-x/notes.md", true, true, false}, // edited after being added
		{"0004-y/solution.rs", false, false, true},
	} {
		var got Change
		for _, c := range st.Changes {
			if c.Path == tc.path {
				got = c
			}
		}
		if got.Path == "" {
			t.Errorf("%s missing from the status", tc.path)
			continue
		}
		if got.Staged != tc.staged || got.Unstaged != tc.unstaged || got.Untracked != tc.untracked {
			t.Errorf("%s: %+v, want staged=%v unstaged=%v untracked=%v",
				tc.path, got, tc.staged, tc.unstaged, tc.untracked)
		}
	}
}

// TestParsePorcelainSkipsARenamesOriginalPath covers the one record that is two fields
// wide. Miscounting it makes the original path look like a separate change, and the
// pane then reports a file the user does not have.
func TestParsePorcelainSkipsARenamesOriginalPath(t *testing.T) {
	st := parsePorcelain(nul(
		"# branch.head main",
		"2 R. N... 100644 100644 100644 aaa bbb R100 0002-new/solution.go",
		"0001-old/solution.go",
		"? untracked.txt",
	))

	if len(st.Changes) != 2 {
		t.Fatalf("got %d changes, want 2: %+v", len(st.Changes), st.Changes)
	}
	if st.Changes[0].Path != "0002-new/solution.go" {
		t.Errorf("rename recorded as %q, want the new path", st.Changes[0].Path)
	}
	if st.Changes[1].Path != "untracked.txt" {
		t.Errorf("the record after a rename was misread as %q", st.Changes[1].Path)
	}
}

func TestParsePorcelainCleanTree(t *testing.T) {
	st := parsePorcelain(nul("# branch.head main", "# branch.upstream origin/main", "# branch.ab +0 -0"))
	if !st.Clean() {
		t.Errorf("clean tree reported %d changes", st.Dirty())
	}
	if st.Unpushed() {
		t.Error("in sync with the upstream should not report unpushed work")
	}
}

func TestParseLog(t *testing.T) {
	got := parseLog(nul(
		"1a2b3c4", "solve(0146): lru cache — go, 58ms", "3 minutes ago",
		"9f8e7d6", "solve(0001): two sum — python3", "2 hours ago",
	))
	if len(got) != 2 {
		t.Fatalf("got %d commits, want 2", len(got))
	}
	if got[0].Short != "1a2b3c4" || got[0].When != "3 minutes ago" {
		t.Errorf("first entry = %+v", got[0])
	}
	if got[1].Subject != "solve(0001): two sum — python3" {
		t.Errorf("second subject = %q", got[1].Subject)
	}
}

// TestParseLogIgnoresATruncatedRecord guards the flat-triple walk: a partial trailing
// record must be dropped rather than producing a commit with empty fields.
func TestParseLogIgnoresATruncatedRecord(t *testing.T) {
	got := parseLog(nul("1a2b3c4", "subject", "now", "9f8e7d6"))
	if len(got) != 1 {
		t.Fatalf("got %d commits, want 1", len(got))
	}
}
