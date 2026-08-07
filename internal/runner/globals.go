package runner

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Nano-AI/leetui/internal/workspace"
)

// The shared half of a driver.
//
// Each language's driver splits in two: a part generated from THIS problem's metaData —
// the argument decoding, the call, the return shape — and a part that is byte-identical
// for every problem. The second half used to be written beside every solution, which put
// ~19KB of scaffolding into each folder and buried the four files you edit under four you
// never open (D-029).
//
// It now lives once, in the workspace's globals/ directory, referenced RELATIVELY:
//
//	0001-two-sum/leetui_main.cpp   #include "../globals/leetui_driver.h"
//	globals/leetui_driver.h
//
// Relative is the whole trick. A quoted include resolves against the including file's own
// directory, so this needs no -I, no compile_commands.json, and no PYTHONPATH — and
// clangd follows it with zero configuration, which was the objection to sharing them.

// writeGlobalDriver copies an embedded driver into the workspace's globals/ directory.
//
// dir is a PROBLEM folder; globals/ is its sibling. Overwrites unconditionally: this file
// is ours, holds nothing a user typed, and must match the binary that generated the entry
// point beside it — a stale driver from an older leetui is a confusing failure at compile
// time rather than an honest one.
func writeGlobalDriver(dir, name, embedded string) error {
	data, err := drivers.ReadFile(embedded)
	if err != nil {
		return fmt.Errorf("read embedded driver %s: %w", embedded, err)
	}

	globals := filepath.Join(filepath.Dir(dir), workspace.GlobalsDir)
	if err := os.MkdirAll(globals, 0o755); err != nil {
		return fmt.Errorf("create globals dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(globals, name), data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

// migrateInclude rewrites a solution's driver include from the old per-folder path to
// the shared one, and removes the copy that used to sit beside it.
//
// leetui WROTE that line, above the `@leetui code=start` marker, in the region docs and
// scaffold.go both call "local scaffolding, not your code". Rewriting exactly it is the
// only way a folder created before D-029 keeps compiling; doing nothing would greet
// every existing problem with `fatal error: 'leetui_driver.h' file not found`.
//
// Narrow on purpose: one exact string, and only when it is present. Nothing below the
// marker is read, let alone touched.
func migrateInclude(dir, name string) error {
	solution := filepath.Join(dir, "solution.cpp")
	body, err := os.ReadFile(solution)
	if err != nil {
		// No solution yet is the ordinary case on a fresh pull.
		return nil
	}

	old := fmt.Sprintf("#include %q", name)
	if !bytes.Contains(body, []byte(old)) {
		return nil
	}
	updated := bytes.Replace(body, []byte(old),
		[]byte(fmt.Sprintf("#include %q", workspace.GlobalRef(name))), 1)

	if err := os.WriteFile(solution, updated, 0o644); err != nil {
		return fmt.Errorf("migrate include in %s: %w", solution, err)
	}
	// The stale copy is dead weight now, and leaving it means the old include would
	// still resolve — so the migration would look optional until the day it was not.
	_ = os.Remove(filepath.Join(dir, name))
	return nil
}
