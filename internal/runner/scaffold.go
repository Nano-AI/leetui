package runner

import (
	"fmt"
	"strings"

	"github.com/Nano-AI/leetui/internal/workspace"
)

// Solution-file scaffolding.
//
// A LeetCode starter snippet is not a compilable file. It has no imports, no package
// clause, and no definition of ListNode or TreeNode, because the judge supplies all of
// that around it. Written to disk verbatim, the result is a buffer your editor lights up
// red: clangd cannot resolve `vector<int>`, pyright cannot resolve `List[int]`.
//
// So the file leetui writes is two regions:
//
//	scaffolding   imports, package clause, driver types — for the editor and the
//	              local compiler. Never sent anywhere.
//	marked code   exactly what LeetCode gets, between @leetui markers.
//
// The markers are the same idea as vscode-leetcode's `@lc code=start`, and files carrying
// those are read too — see ExtractCode.
//
// RULE: scaffolding REFERENCES the driver's types, it never redefines them. Python's
// driver serializes with `isinstance(value, (ListNode, TreeNode))`, so a solution that
// declared its own ListNode would return an object the driver could not serialize, and
// the failure would look like a wrong answer rather than a wiring mistake. Including or
// importing the driver's definitions keeps one set of types in play.

// Marker text. The prefix is the comment token for the language, added by markers().
const (
	markStart = "@leetui code=start"
	markEnd   = "@leetui code=end"

	// vscode-leetcode's markers, accepted on read so a workspace made with that
	// extension works here without being rewritten.
	vscodeStart = "@lc code=start"
	vscodeEnd   = "@lc code=end"
)

// commentTokens maps a language slug to its line-comment prefix.
var commentTokens = map[string]string{
	"python3": "#", "python": "#", "ruby": "#", "elixir": "#", "bash": "#",
	"erlang": "%",
	"racket": ";;",
	"mysql":  "--", "mssql": "--", "oraclesql": "--", "postgresql": "--",
}

// comment returns the line-comment prefix for a language, defaulting to "//".
func comment(slug string) string {
	if c, ok := commentTokens[slug]; ok {
		return c
	}
	return "//"
}

// Scaffold describes one solution file to write.
type Scaffold struct {
	Lang Lang
	Meta Meta

	// ID, Title, Difficulty, and URL go in the header comment, so the file says what it
	// is when opened on its own — outside the problem folder, away from README.md.
	ID         int
	Title      string
	Difficulty string
	URL        string
}

// File renders the complete solution file for a starter snippet.
func (s Scaffold) File(snippet string) string {
	c := comment(s.Lang.Slug)

	var b strings.Builder
	b.WriteString(s.header(c))

	if pre := s.preamble(); pre != "" {
		b.WriteString(pre + "\n")
	}

	b.WriteString(c + " " + markStart + "\n")
	b.WriteString(strings.TrimRight(snippet, "\n") + "\n")
	b.WriteString(c + " " + markEnd + "\n")
	return b.String()
}

// header names the problem and states the file's one rule.
//
// The note is two lines and earns them: without it, a reader has no way to know that the
// region below is the only part that gets submitted, and would reasonably delete the
// markers as noise.
func (s Scaffold) header(c string) string {
	title := s.Title
	if s.ID > 0 {
		title = fmt.Sprintf("%d. %s", s.ID, s.Title)
	}
	if s.Difficulty != "" {
		title += " · " + s.Difficulty
	}

	lines := []string{title}
	if s.URL != "" {
		lines = append(lines, s.URL)
	}
	lines = append(lines, "",
		"Everything above the marker is local scaffolding, for your editor",
		"and the local runner. Only the marked region is submitted.")

	var b strings.Builder
	for _, l := range lines {
		if l == "" {
			b.WriteString(c + "\n")
			continue
		}
		b.WriteString(c + " " + l + "\n")
	}
	b.WriteString("\n")
	return b.String()
}

// preamble is the language-specific scaffolding above the marked region.
func (s Scaffold) preamble() string {
	switch s.Lang.Slug {
	case "golang":
		// The driver lives in this folder as package main, which is what puts ListNode
		// and TreeNode in scope for the solution — same package, no import.
		return "package main\n"

	case "cpp":
		// One include does all of it: the driver header carries the std headers,
		// `using namespace std`, and the node types. It is #pragma once, so the
		// generated main including both it and this file is fine.
		return fmt.Sprintf("#include %q\n", workspace.GlobalRef(cppHeaderFile))

	case "python3", "python":
		return s.pythonPreamble()

	case "java":
		return "import java.util.*;\n"

	case "csharp":
		return "using System;\nusing System.Collections.Generic;\nusing System.Linq;\n"

	default:
		// JavaScript, TypeScript, Rust, and the rest need nothing to be readable, and
		// inventing imports for a language with no local driver would be guessing.
		return ""
	}
}

// pythonPreamble imports the type names LeetCode's annotations use.
//
// The node types come from the generated driver rather than being declared here, so the
// object a solution returns is the one the driver knows how to serialize.
func (s Scaffold) pythonPreamble() string {
	pre := "from typing import Any, Dict, List, Optional, Set, Tuple\n"
	if nodes := s.nodeTypes(); len(nodes) > 0 {
		pre += fmt.Sprintf("\nfrom %s import %s\n",
			strings.TrimSuffix(pyDriverFile, ".py"), strings.Join(nodes, ", "))
	}
	return pre
}

// nodeTypes reports which of LeetCode's node types this problem mentions, so an import
// is only added when it is used. An unused import is noise a linter will flag.
func (s Scaffold) nodeTypes() []string {
	var seen []string
	all := append(s.Meta.ParamTypes(), s.Meta.Return.Type)
	if methods, err := s.Meta.DesignMethods(); err == nil {
		for _, m := range methods {
			for _, p := range m.Params {
				all = append(all, p.Type)
			}
			all = append(all, m.Return.Type)
		}
	}

	for _, name := range []string{"ListNode", "TreeNode"} {
		for _, t := range all {
			if strings.Contains(t, name) {
				seen = append(seen, name)
				break
			}
		}
	}
	return seen
}
