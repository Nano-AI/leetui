package runner

import "sort"

// Lang is one of LeetCode's submission languages.
//
// Slug is the identifier LeetCode uses on the wire (`codeSnippets[].langSlug` and the
// `lang` field on a submission). It is the only value that may be sent to the API —
// Display and Ext exist for humans and filenames.
type Lang struct {
	Slug    string
	Display string
	Ext     string

	// Local reports whether leetui can execute this language on the machine (D-004).
	// False means edit and submit still work; only `run` goes to the judge.
	Local bool
}

// langs is the full submission language set LeetCode offers.
//
// Local is true only for the four leetgo implements drivers for. Adding a language here
// with Local: true without a driver behind it would promise a local run that cannot
// happen — see D-004 before changing any of these flags.
var langs = []Lang{
	{Slug: "python3", Display: "Python3", Ext: ".py", Local: true},
	{Slug: "golang", Display: "Go", Ext: ".go", Local: true},
	{Slug: "cpp", Display: "C++", Ext: ".cpp", Local: true},
	{Slug: "rust", Display: "Rust", Ext: ".rs", Local: true},

	{Slug: "java", Display: "Java", Ext: ".java"},
	{Slug: "javascript", Display: "JavaScript", Ext: ".js"},
	{Slug: "typescript", Display: "TypeScript", Ext: ".ts"},
	{Slug: "python", Display: "Python", Ext: ".py"},
	{Slug: "c", Display: "C", Ext: ".c"},
	{Slug: "csharp", Display: "C#", Ext: ".cs"},
	{Slug: "kotlin", Display: "Kotlin", Ext: ".kt"},
	{Slug: "swift", Display: "Swift", Ext: ".swift"},
	{Slug: "ruby", Display: "Ruby", Ext: ".rb"},
	{Slug: "scala", Display: "Scala", Ext: ".scala"},
	{Slug: "php", Display: "PHP", Ext: ".php"},
	{Slug: "dart", Display: "Dart", Ext: ".dart"},
	{Slug: "elixir", Display: "Elixir", Ext: ".ex"},
	{Slug: "erlang", Display: "Erlang", Ext: ".erl"},
	{Slug: "racket", Display: "Racket", Ext: ".rkt"},
}

var bySlug = func() map[string]Lang {
	m := make(map[string]Lang, len(langs))
	for _, l := range langs {
		m[l.Slug] = l
	}
	return m
}()

// Lookup returns a language by its LeetCode slug.
func Lookup(slug string) (Lang, bool) {
	l, ok := bySlug[slug]
	return l, ok
}

// Langs returns every known language, locally-runnable ones first, then alphabetical.
//
// The ordering is the point: when the picker lists languages, the ones that give a fast
// offline loop should be the ones the cursor lands near.
func Langs() []Lang {
	out := append([]Lang(nil), langs...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Local != out[j].Local {
			return out[i].Local
		}
		return out[i].Display < out[j].Display
	})
	return out
}

// Available filters to the languages a specific problem offers, preserving Langs order.
//
// LeetCode does not offer every language on every problem — database and shell problems
// offer only SQL or Bash — so the picker must be built from the problem's own snippet
// table rather than from this list.
func Available(slugs []string) []Lang {
	have := make(map[string]bool, len(slugs))
	for _, s := range slugs {
		have[s] = true
	}
	var out []Lang
	for _, l := range Langs() {
		if have[l.Slug] {
			out = append(out, l)
		}
	}
	return out
}

// Filename is the solution file for this language, e.g. "solution.go".
//
// One file per language per problem folder, so a problem solved twice in two languages
// keeps both (D-010).
func (l Lang) Filename() string { return "solution" + l.Ext }
