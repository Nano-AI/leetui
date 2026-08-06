// Package runner executes solutions locally.
//
// The interfaces here were written as a firewall against leetgo's unstable API
// (D-005). That import was then measured and dropped — it compiled 144 packages for
// four language drivers — and the drivers were vendored into `drivers/` instead.
//
// The firewall still earns its place. Because the TUI only ever saw these interfaces,
// swapping the implementation underneath was a substitution rather than a rewrite, and
// the same seam now separates "how a language runs" from everything that calls it.
//
// File map:
//
//	local.go      the Engine implementation: toolchain detection, dispatch
//	python.go     Python code generation and execution
//	drivers/      vendored per-language runtimes, embedded into the binary
//	meta.go       parsing LeetCode's metaData
//	testcase.go   pairing example inputs with outputs scraped from the statement
//	compare.go    judging output against expectation
//	overrides.go  the per-problem semantics metaData cannot express
//	lang.go       the language registry
//
// Local execution covers the languages with a vendored driver AND a toolchain on this
// machine. Everything else edits and submits normally; only `run` goes to the judge,
// which is a routing decision rather than a failure (D-004).
package runner

import (
	"context"
	"errors"
	"time"
)

// ErrLangNotLocal means this language has no local driver and must use the judge.
//
// Callers are expected to fall back to a remote run rather than surfacing this as a
// failure — it is a routing decision, not an error the user caused.
var ErrLangNotLocal = errors.New("no local runner for this language")

// ErrNoToolchain means the language is supported but its compiler or interpreter is not
// installed. Distinct from ErrLangNotLocal because the fix is different: install a
// toolchain, rather than pick another language.
var ErrNoToolchain = errors.New("toolchain not installed")

// TestCase is one input/expected pair.
//
// Input is newline-separated with one line per parameter, matching the format LeetCode
// uses in `exampleTestcases`. Expected may be empty for cases the user added by hand,
// in which case the runner reports output without judging it.
type TestCase struct {
	Input    string
	Expected string
}

// CaseResult is the outcome of one test case.
type CaseResult struct {
	Case   TestCase
	Actual string

	// Passed is meaningful only when the case had an Expected value.
	Passed bool
	Judged bool

	// Err is set when the case failed to run at all — a panic, a timeout, a crash —
	// as opposed to running and producing the wrong answer.
	Err error

	Elapsed time.Duration
}

// Result is a whole local run.
type Result struct {
	Cases []CaseResult

	// CompileErr is set when the solution never ran. Its presence means Cases is empty
	// and the output should be shown verbatim: a compiler already writes a better
	// message than we could summarise.
	CompileErr string

	Elapsed time.Duration
}

// Passed reports whether every judged case passed and nothing failed to run.
func (r Result) Passed() bool {
	if r.CompileErr != "" || len(r.Cases) == 0 {
		return false
	}
	for _, c := range r.Cases {
		if c.Err != nil || (c.Judged && !c.Passed) {
			return false
		}
	}
	return true
}

// Summary counts outcomes for a one-line status.
func (r Result) Summary() (passed, failed, errored int) {
	for _, c := range r.Cases {
		switch {
		case c.Err != nil:
			errored++
		case !c.Judged:
		case c.Passed:
			passed++
		default:
			failed++
		}
	}
	return
}

// Problem is the subset of a problem a runner needs.
//
// Deliberately not store.Detail: the runner should not depend on the storage layer, and
// keeping this narrow documents exactly what code generation actually consumes.
type Problem struct {
	Slug  string
	Title string

	// MetaData is LeetCode's JSON function signature. It drives driver generation and
	// is the only structured description of the solution's shape that exists.
	MetaData string

	// Snippet is the starter code for the chosen language.
	Snippet string
}

// Generator writes the files needed to work on a problem: the solution stub and
// whatever driver or harness the language needs to execute it.
type Generator interface {
	// Generate writes into dir, which already exists. It must not overwrite a solution
	// file that already has the user's work in it.
	Generate(ctx context.Context, p Problem, lang Lang, dir string) error
}

// Runner executes a generated solution against test cases.
type Runner interface {
	// Supports reports whether this language can run locally here. False sends the
	// caller to the judge instead (D-004).
	Supports(lang Lang) bool

	// Run executes the solution in dir. A failing test case is reported inside Result,
	// not as an error; the error return is for the run not happening at all.
	//
	// rule carries the per-problem judging semantics metaData cannot express. Passing
	// it explicitly rather than looking it up inside keeps the comparison testable and
	// makes it obvious at the call site that the choice was made.
	Run(ctx context.Context, dir string, lang Lang, cases []TestCase, rule Rule) (Result, error)
}

// Engine is the whole local-execution capability.
type Engine interface {
	Generator
	Runner
}
