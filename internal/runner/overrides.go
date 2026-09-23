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

	// OrderedRows preserves the order within each row of an unordered answer.
	// Permutations and partitions are sequences, unlike subsets or anagram groups.
	OrderedRows bool

	// FloatTolerance judges numbers to within this absolute error. Zero means exact.
	FloatTolerance float64

	// NoLocal names the reason a problem cannot run locally at all, or is empty when
	// it can.
	//
	// A handful of problems carry a `metaData` that does not describe the function
	// LeetCode actually asks you to write. Clone Graph declares
	// `cloneGraph(integer[][]) -> boolean` and hands you `Node* cloneGraph(Node*)`.
	// Linked List Cycle declares two parameters and the snippet takes one, because the
	// judge uses the second to wire the cycle before calling. Nothing generated from
	// that description can compile, so the driver must not be written at all.
	//
	// This is deliberately a list of named problems rather than a rule over
	// `metaData.manual`. Number of Islands is also marked manual and its description is
	// exact — gating on the flag would break a problem that works today.
	NoLocal string
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
	"permutations":                          {MutatesArg: -1, Unordered: true, OrderedRows: true},
	"permutations-ii":                       {MutatesArg: -1, Unordered: true, OrderedRows: true},
	"combination-sum":                       {MutatesArg: -1, Unordered: true},
	"combination-sum-ii":                    {MutatesArg: -1, Unordered: true},
	"combinations":                          {MutatesArg: -1, Unordered: true},
	"group-anagrams":                        {MutatesArg: -1, Unordered: true},
	"3sum":                                  {MutatesArg: -1, Unordered: true},
	"4sum":                                  {MutatesArg: -1, Unordered: true},
	"palindrome-partitioning":               {MutatesArg: -1, Unordered: true, OrderedRows: true},
	"letter-combinations-of-a-phone-number": {MutatesArg: -1, Unordered: true},
	"generate-parentheses":                  {MutatesArg: -1, Unordered: true},
	"word-break-ii":                         {MutatesArg: -1, Unordered: true},
	"find-all-anagrams-in-a-string":         {MutatesArg: -1, Unordered: true},

	// --- judged to a tolerance ---
	"median-of-two-sorted-arrays":      {MutatesArg: -1, FloatTolerance: 1e-5},
	"average-of-levels-in-binary-tree": {MutatesArg: -1, FloatTolerance: 1e-5},
	"maximum-average-subarray-i":       {MutatesArg: -1, FloatTolerance: 1e-5},
	"minimum-time-to-repair-cars":      {MutatesArg: -1, FloatTolerance: 1e-5},

	// --- metaData does not describe the real function; the judge writes the driver ---
	//
	// Verified against each problem's own C++ snippet. Number of Islands is marked
	// manual too and is deliberately absent: its description is exact and it runs.
	"clone-graph": {MutatesArg: -1,
		NoLocal: "the judge builds the graph and passes a Node*, which metaData describes as integer[][]"},
	"construct-quad-tree": {MutatesArg: -1,
		NoLocal: "the answer is a quad tree of Node*, which metaData describes as list<list<integer>>"},
	"encode-n-ary-tree-to-binary-tree": {MutatesArg: -1,
		NoLocal: "both sides are an n-ary Node* the judge constructs"},
	"linked-list-cycle": {MutatesArg: -1,
		NoLocal: "the judge uses the second parameter to wire the cycle before calling; the snippet takes one"},
	"inorder-successor-in-bst": {MutatesArg: -1,
		NoLocal: "the second parameter is a pointer into the tree, not the integer metaData claims"},
	"read-n-characters-given-read4": {MutatesArg: -1,
		NoLocal: "read4 is supplied by the judge and exists nowhere locally"},
	"read-n-characters-given-read4-ii-call-multiple-times": {MutatesArg: -1,
		NoLocal: "read4 is supplied by the judge and exists nowhere locally"},
}

// RuleFor returns how a problem should be judged.
func RuleFor(slug string) Rule {
	if r, ok := overrides[slug]; ok {
		return r
	}
	return DefaultRule()
}

// AnswerArg reports which argument holds the answer, or -1 when the return value does.
//
// The override table is the first authority. Where it is silent, a void return is itself
// evidence: a function that returns nothing and is not a design class has no channel to
// answer through except a mutated argument, and the first argument is that argument in
// every case LeetCode has shipped. Printing "null" instead — which is what a void problem
// with no entry used to do — is a guaranteed mismatch with nothing in it for the user.
//
// Two exclusions. Design problems answer through an operation sequence. Problems marked
// `manual` have a judge driver LeetCode wrote by hand, and its output need not be any
// argument we hold: delete-node-in-a-linked-list is handed the node to delete and judged
// on a list head it never gives us. Guessing there is still no worse than "null", but the
// table is the place to say so deliberately.
//
// A mismatch on an inferred rule is still reported as "check on the judge" (D-003):
// HasOverride stays false, so a guess never speaks with a curated rule's confidence.
func AnswerArg(slug string, meta Meta) int {
	// A curated -1 is a decision, not a silence — squares-of-a-sorted-array and
	// remove-nth-node-from-end-of-list are in the table precisely to opt out.
	if HasOverride(slug) {
		return RuleFor(slug).MutatesArg
	}
	if meta.Return.Type == "void" && !meta.IsDesign() && !meta.Manual && len(meta.Params) > 0 {
		return 0
	}
	return -1
}

// HasOverride reports whether a problem has a curated rule.
//
// The UI uses this to phrase a failure honestly: without an override, a mismatch might
// be the comparator's fault rather than the solution's.
func HasOverride(slug string) bool {
	_, ok := overrides[slug]
	return ok
}
