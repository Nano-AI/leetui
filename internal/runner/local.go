package runner

import (
	"context"
	"embed"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// drivers holds the vendored per-language runtimes.
//
// These are adapted from leetgo (MIT, Copyright (c) 2022 Jo). They are embedded rather
// than installed: a driver that needs `pip install` before the first run is a setup step
// the user did not ask for. See D-005 for why they are vendored instead of imported.
//
//go:embed drivers
var drivers embed.FS

// DefaultTimeout bounds one test case. LeetCode's own limit is a few seconds; this is
// generous enough for a slow debug build and short enough that an accidental infinite
// loop fails instead of hanging the UI.
const DefaultTimeout = 10 * time.Second

// Local runs solutions on this machine.
//
// It implements Engine, and it is the only implementation that matters right now — the
// interface exists to keep the TUI from caring how execution happens, and to make the
// remote fallback in D-004 a routing decision instead of a special case.
type Local struct {
	// Timeout bounds a single test case. Zero uses DefaultTimeout.
	Timeout time.Duration

	once      sync.Once
	toolchain map[string]bool
}

// NewLocal returns a ready Engine.
func NewLocal() *Local { return &Local{Timeout: DefaultTimeout} }

// toolchains maps a language to the executable that must be on PATH to run it.
//
// Only languages with a vendored driver belong here. A language listed with a toolchain
// but no driver would report "needs rustc" when the real answer is "no driver yet".
// designSupported lists languages whose driver handles class-with-operations problems.
var designSupported = map[string]bool{
	"python3": true,
	// JavaScript gets these because a design problem in JS is just a constructor
	// function and its prototype — no type declarations to reconstruct, which is what
	// stops Go and C++ from taking them.
	"javascript": true,
	"typescript": true,
}

var toolchains = map[string]string{
	"python3": "python3",
	"golang":  "go",
	"cpp":     "c++",
	// One binary for both. Node strips TypeScript's types itself from v23, so
	// solution.ts needs no transpiler, no tsconfig, and no second toolchain.
	"javascript": "node",
	"typescript": "node",
}

// detect probes PATH once, so a run does not pay for it per case.
func (l *Local) detect() {
	l.once.Do(func() {
		l.toolchain = make(map[string]bool, len(toolchains))
		for lang, bin := range toolchains {
			_, err := exec.LookPath(bin)
			l.toolchain[lang] = err == nil
		}
	})
}

// Supports reports whether this language can actually run here.
//
// Both halves matter: the language needs a vendored driver (Lang.Local) AND the machine
// needs the toolchain. Rust is a real example — D-004 lists it as supported, but without
// rustup installed a local run is not possible, and saying so beats failing later.
func (l *Local) Supports(lang Lang) bool {
	if !lang.Local {
		return false
	}
	l.detect()
	return l.toolchain[lang.Slug]
}

// MissingToolchain names the executable a language needs but does not have, or "" when
// it is available. Lets the UI say "install go" rather than "unsupported".
func (l *Local) MissingToolchain(lang Lang) string {
	if !lang.Local {
		return ""
	}
	l.detect()
	if l.toolchain[lang.Slug] {
		return ""
	}
	return toolchains[lang.Slug]
}

// Generate writes the driver and entry point for a problem.
func (l *Local) Generate(ctx context.Context, p Problem, lang Lang, dir string) error {
	if !lang.Local {
		return fmt.Errorf("%s: %w", lang.Display, ErrLangNotLocal)
	}

	meta, err := ParseMeta(p.MetaData)
	if err != nil {
		return err
	}
	// Design problems need a different driver shape — a class plus an operation
	// sequence. Languages whose driver does not implement it decline rather than guess,
	// because guessing produces confidently wrong answers.
	if meta.IsDesign() && !designSupported[lang.Slug] {
		return fmt.Errorf("%s is a design problem and %s has no driver for those: %w",
			p.Slug, lang.Display, ErrLangNotLocal)
	}

	switch lang.Slug {
	case "python3":
		return l.generatePython(p, meta, dir)
	case "golang":
		return l.generateGo(p, meta, dir)
	case "cpp":
		return l.generateCpp(p, meta, dir)
	case "javascript", "typescript":
		return l.generateJS(p, meta, lang, dir)
	default:
		return fmt.Errorf("%s: %w", lang.Display, ErrLangNotLocal)
	}
}

// Run executes every case and collects the results.
//
// A failing case is data, not an error: the error return is reserved for the run not
// happening at all, so the caller can tell "your answer is wrong" apart from "nothing
// ran".
func (l *Local) Run(ctx context.Context, dir string, lang Lang, cases []TestCase, rule Rule) (Result, error) {
	if !l.Supports(lang) {
		if bin := l.MissingToolchain(lang); bin != "" {
			return Result{}, fmt.Errorf("%s needs %q on PATH: %w", lang.Display, bin, ErrNoToolchain)
		}
		return Result{}, fmt.Errorf("%s: %w", lang.Display, ErrLangNotLocal)
	}

	switch lang.Slug {
	case "python3":
		return l.runPython(ctx, dir, cases, rule)
	case "golang":
		return l.runGo(ctx, dir, cases, rule)
	case "cpp":
		return l.runCpp(ctx, dir, cases, rule)
	case "javascript", "typescript":
		return l.runJS(ctx, dir, lang, cases, rule)
	default:
		return Result{}, fmt.Errorf("%s: %w", lang.Display, ErrLangNotLocal)
	}
}

func (l *Local) timeout() time.Duration {
	if l.Timeout > 0 {
		return l.Timeout
	}
	return DefaultTimeout
}
