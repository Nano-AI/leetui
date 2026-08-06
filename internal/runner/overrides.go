package runner

// Comparator overrides — what metaData cannot tell us.
//
// LeetCode's `metaData` gives parameter names and types and the return type. It is
// silent on three things that decide whether an answer is correct:
//
//  1. IN-PLACE problems answer through a mutated argument. The function returns a
//     length, and the real answer is the first k elements of the input.
//  2. UNORDERED problems accept any permutation. Comparing literally fails a correct
//     solution.
//  3. FLOAT problems are judged to a tolerance, not exactly.
//
// This table supplies those three facts per problem. It is seeded with the well-known
// cases and grows as they are found.
//
// A MISSING ENTRY IS NOT A BUG THAT FAILS THE USER. An unknown mismatch is reported as
// "local check disagreed — verify on the judge", never as a wrong answer (D-003). That
// is what makes an incomplete table safe to ship.

// Rule is how one problem's output should be judged.
type Rule struct {
	// MutatesArg is the index of the parameter holding the answer, or -1 when the
	// return value is the answer.
	MutatesArg int

	// Unordered accepts any permutation of a list answer.
	Unordered bool

	// FloatTolerance judges numbers to within this absolute error. Zero means exact.
	FloatTolerance float64
}

// DefaultRule judges the return value exactly.
func DefaultRule() Rule { return Rule{MutatesArg: -1} }

// overrides maps a problem slug to its judging rule.
var overrides = map[string]Rule{
	// --- answer is a mutated argument ---
	"remove-duplicates-from-sorted-array":    {MutatesArg: 0},
	"remove-duplicates-from-sorted-array-ii": {MutatesArg: 0},
	"remove-element":                         {MutatesArg: 0},
	"move-zeroes":                            {MutatesArg: 0},
	"sort-colors":                            {MutatesArg: 0},
	"rotate-array":                           {MutatesArg: 0},
	"rotate-image":                           {MutatesArg: 0},
	"set-matrix-zeroes":                      {MutatesArg: 0},
	"next-permutation":                       {MutatesArg: 0},
	"merge-sorted-array":                     {MutatesArg: 0},
	"reverse-string":                         {MutatesArg: 0},
	"squares-of-a-sorted-array":              {MutatesArg: -1},
	"flatten-binary-tree-to-linked-list":     {MutatesArg: 0},
	"remove-nth-node-from-end-of-list":       {MutatesArg: -1},

	// --- any order accepted ---
	"subsets":                               {MutatesArg: -1, Unordered: true},
	"subsets-ii":                            {MutatesArg: -1, Unordered: true},
	"permutations":                          {MutatesArg: -1, Unordered: true},
	"permutations-ii":                       {MutatesArg: -1, Unordered: true},
	"combination-sum":                       {MutatesArg: -1, Unordered: true},
	"combination-sum-ii":                    {MutatesArg: -1, Unordered: true},
	"combinations":                          {MutatesArg: -1, Unordered: true},
	"group-anagrams":                        {MutatesArg: -1, Unordered: true},
	"3sum":                                  {MutatesArg: -1, Unordered: true},
	"4sum":                                  {MutatesArg: -1, Unordered: true},
	"palindrome-partitioning":               {MutatesArg: -1, Unordered: true},
	"letter-combinations-of-a-phone-number": {MutatesArg: -1, Unordered: true},
	"generate-parentheses":                  {MutatesArg: -1, Unordered: true},
	"word-break-ii":                         {MutatesArg: -1, Unordered: true},
	"find-all-anagrams-in-a-string":         {MutatesArg: -1, Unordered: true},

	// --- judged to a tolerance ---
	"median-of-two-sorted-arrays":      {MutatesArg: -1, FloatTolerance: 1e-5},
	"average-of-levels-in-binary-tree": {MutatesArg: -1, FloatTolerance: 1e-5},
	"maximum-average-subarray-i":       {MutatesArg: -1, FloatTolerance: 1e-5},
	"minimum-time-to-repair-cars":      {MutatesArg: -1, FloatTolerance: 1e-5},
}

// RuleFor returns how a problem should be judged.
func RuleFor(slug string) Rule {
	if r, ok := overrides[slug]; ok {
		return r
	}
	return DefaultRule()
}

// HasOverride reports whether a problem has a curated rule.
//
// The UI uses this to phrase a failure honestly: without an override, a mismatch might
// be the comparator's fault rather than the solution's.
func HasOverride(slug string) bool {
	_, ok := overrides[slug]
	return ok
}
