package runner

import "testing"

func TestCompareExact(t *testing.T) {
	r := DefaultRule()
	cases := []struct {
		actual, expected string
		want             bool
	}{
		{"[0,1]", "[0,1]", true},
		{"[0,1]", "[1,0]", false},
		{" [0,1] ", "[0,1]", true}, // whitespace is not meaningful
		{"[0, 1]", "[0,1]", true},  // JSON equivalence, not string equality
		{"true", "true", true},
		{"3", "3", true},
		{"3", "4", false},
		{`"abc"`, `"abc"`, true},
		{"null", "null", true},
		{"[[1,2],[3]]", "[[1,2],[3]]", true},
		{"[[1,2],[3]]", "[[2,1],[3]]", false}, // order matters without an override
		// Non-JSON on both sides falls back to whitespace-insensitive text.
		{"hello  world", "hello world", true},
	}
	for _, tc := range cases {
		if got := Compare(tc.actual, tc.expected, r); got != tc.want {
			t.Errorf("Compare(%q, %q) = %v, want %v", tc.actual, tc.expected, got, tc.want)
		}
	}
}

// TestCompareUnordered covers problems that accept any permutation. Without this a
// correct solution to subsets or permutations fails locally for no reason.
func TestCompareUnordered(t *testing.T) {
	r := Rule{MutatesArg: -1, Unordered: true}
	cases := []struct {
		actual, expected string
		want             bool
	}{
		{"[[1,2],[3]]", "[[3],[1,2]]", true}, // outer order
		{"[[1,2],[3]]", "[[3],[2,1]]", true}, // inner order too
		{"[3,1,2]", "[1,2,3]", true},
		{"[1,2]", "[1,2,3]", false},           // length still matters
		{"[[1,2],[3]]", "[[1,2],[4]]", false}, // contents still matter
		{`[["eat","tea"],["bat"]]`, `[["bat"],["tea","eat"]]`, true},
	}
	for _, tc := range cases {
		if got := Compare(tc.actual, tc.expected, r); got != tc.want {
			t.Errorf("unordered Compare(%q, %q) = %v, want %v", tc.actual, tc.expected, got, tc.want)
		}
	}
}

func TestCompareFloatTolerance(t *testing.T) {
	r := Rule{MutatesArg: -1, FloatTolerance: 1e-5}
	cases := []struct {
		actual, expected string
		want             bool
	}{
		{"2.00000", "2.0", true},
		{"2.000001", "2.0", true}, // inside tolerance
		{"2.1", "2.0", false},     // outside it
		{"[1.000001,2.0]", "[1.0,2.0]", true},
		{"[1.5,2.0]", "[1.0,2.0]", false},
	}
	for _, tc := range cases {
		if got := Compare(tc.actual, tc.expected, r); got != tc.want {
			t.Errorf("float Compare(%q, %q) = %v, want %v", tc.actual, tc.expected, got, tc.want)
		}
	}
}

func TestRuleFor(t *testing.T) {
	if r := RuleFor("remove-duplicates-from-sorted-array"); r.MutatesArg != 0 {
		t.Errorf("in-place problem MutatesArg = %d, want 0", r.MutatesArg)
	}
	if r := RuleFor("subsets"); !r.Unordered {
		t.Error("subsets should accept any order")
	}
	if r := RuleFor("median-of-two-sorted-arrays"); r.FloatTolerance == 0 {
		t.Error("median should be judged to a tolerance")
	}
	// An unknown problem gets the strict default, and is reported as uncurated so the
	// UI can say a mismatch might be ours rather than the user's.
	r := RuleFor("some-problem-nobody-has-curated")
	if r.MutatesArg != -1 || r.Unordered || r.FloatTolerance != 0 {
		t.Errorf("unknown problem got a non-default rule: %+v", r)
	}
	if HasOverride("some-problem-nobody-has-curated") {
		t.Error("unknown problem reported as curated")
	}
	if !HasOverride("subsets") {
		t.Error("curated problem reported as uncurated")
	}
}

func TestParseMeta(t *testing.T) {
	const twoSum = `{"name":"twoSum","params":[{"name":"nums","type":"integer[]"},
		{"name":"target","type":"integer"}],"return":{"type":"integer[]","size":2}}`

	m, err := ParseMeta(twoSum)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if m.Name != "twoSum" {
		t.Errorf("name = %q", m.Name)
	}
	if got := m.ParamTypes(); len(got) != 2 || got[0] != "integer[]" || got[1] != "integer" {
		t.Errorf("param types = %v", got)
	}
	if got := m.ParamNames(); len(got) != 2 || got[0] != "nums" {
		t.Errorf("param names = %v", got)
	}
	if m.IsDesign() {
		t.Error("two-sum is not a design problem")
	}

	// Design problems must be recognised, because the driver cannot handle them and
	// guessing would produce confidently wrong answers.
	design := `{"classname":"LRUCache","constructor":{"params":[{"name":"capacity","type":"integer"}]},
		"methods":[{"name":"get","params":[{"name":"key","type":"integer"}],"return":{"type":"integer"}}]}`
	dm, err := ParseMeta(design)
	if err != nil {
		t.Fatalf("parse design: %v", err)
	}
	if !dm.IsDesign() {
		t.Error("LRUCache should be recognised as a design problem")
	}

	for _, bad := range []string{"", "{", "{}"} {
		if _, err := ParseMeta(bad); err == nil {
			t.Errorf("ParseMeta(%q) accepted invalid metaData", bad)
		}
	}
}

func TestAnswerArg(t *testing.T) {
	meta := func(t *testing.T, raw string) Meta {
		t.Helper()
		m, err := ParseMeta(raw)
		if err != nil {
			t.Fatalf("parse meta: %v", err)
		}
		return m
	}
	voidOneArg := `{"name":"moveZeroes","params":[{"name":"nums","type":"integer[]"}],
		"return":{"type":"void"}}`

	cases := []struct {
		name, slug, meta string
		want             int
	}{
		{
			name: "a curated argument wins over anything inferred",
			slug: "merge-sorted-array", meta: voidOneArg, want: 0,
		},
		{
			// squares-of-a-sorted-array is in the table at -1 on purpose. A curated
			// silence is a decision and must not be second-guessed.
			name: "a curated opt-out is respected",
			slug: "squares-of-a-sorted-array", meta: voidOneArg, want: -1,
		},
		{
			name: "an uncurated void problem answers through its first argument",
			slug: "no-such-problem", meta: voidOneArg, want: 0,
		},
		{
			name: "an uncurated problem that returns something does not",
			slug: "no-such-problem", meta: twoSumMeta, want: -1,
		},
		{
			// LeetCode writes these judge drivers by hand and prints something we may
			// not hold — delete-node-in-a-linked-list gets the node, not the head.
			name: "a manual problem is left alone",
			slug: "no-such-problem",
			meta: `{"name":"deleteNode","params":[{"name":"node","type":"ListNode"}],
				"return":{"type":"void"},"manual":true}`,
			want: -1,
		},
		{
			name: "a design problem answers through its operations",
			slug: "no-such-problem",
			meta: `{"classname":"LRUCache","constructor":{"params":[]},
				"methods":[{"name":"get","params":[],"return":{"type":"integer"}}],
				"return":{"type":"void"}}`,
			want: -1,
		},
		{
			name: "a void problem with no arguments has nothing to mutate",
			slug: "no-such-problem",
			meta: `{"name":"noop","params":[],"return":{"type":"void"}}`,
			want: -1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := AnswerArg(tc.slug, meta(t, tc.meta)); got != tc.want {
				t.Errorf("AnswerArg(%q) = %d, want %d", tc.slug, got, tc.want)
			}
		})
	}

	// Inference must not borrow a curated rule's authority: the UI phrases an uncurated
	// mismatch as "check on the judge" rather than a wrong answer (D-003).
	if HasOverride("no-such-problem") {
		t.Error("an inferred problem reported itself as curated")
	}
}

// TestInPlaceOverridesNameTheFirstArgument guards the shape both compiled generators
// assume. They index the mutated argument directly, so an entry naming a later one would
// generate a harness that answers with the wrong variable.
func TestInPlaceOverridesNameTheFirstArgument(t *testing.T) {
	for slug, rule := range overrides {
		if rule.MutatesArg > 0 {
			t.Errorf("%s names argument %d; the generators assume 0", slug, rule.MutatesArg)
		}
	}
}
