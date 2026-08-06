package runner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// goDriverFile deliberately has NO leading underscore.
//
// The go tool ignores files beginning with "_" or ".", so an underscore-prefixed driver
// is silently excluded from the build and the package fails with "function main is
// undeclared" — pointing nowhere near the real cause.
const goDriverFile = "leetui_driver.go"

// goTypes maps LeetCode's metaData type names onto Go types.
//
// LeetCode spells the same shape several ways across problems ("integer[]" and
// "list<integer>"), so both forms are accepted rather than assuming one.
var goTypes = map[string]string{
	"integer": "int", "int": "int", "long": "int64",
	"double": "float64", "float": "float64",
	"boolean": "bool", "string": "string", "character": "byte",
	"void": "",

	"integer[]": "[]int", "list<integer>": "[]int",
	"integer[][]": "[][]int", "list<list<integer>>": "[][]int",
	"string[]": "[]string", "list<string>": "[]string",
	"string[][]": "[][]string", "list<list<string>>": "[][]string",
	"double[]": "[]float64", "list<double>": "[]float64",
	"boolean[]": "[]bool", "list<boolean>": "[]bool",
	"character[]": "[]byte", "list<character>": "[]byte",
	"character[][]": "[][]byte", "list<list<character>>": "[][]byte",

	"ListNode": "*ListNode", "TreeNode": "*TreeNode",
	"ListNode[]": "[]*ListNode", "TreeNode[]": "[]*TreeNode",
	"list<ListNode>": "[]*ListNode", "list<TreeNode>": "[]*TreeNode",
}

// goType resolves a metaData type, reporting unknown ones rather than guessing.
//
// A wrong guess compiles into a driver that decodes garbage; an error here sends the
// problem to the judge instead, which is always correct if slower.
func goType(t string) (string, error) {
	if g, ok := goTypes[strings.TrimSpace(t)]; ok {
		return g, nil
	}
	return "", fmt.Errorf("no Go type for metaData type %q", t)
}

// generateGo writes the driver and the run() that calls the solution.
func (l *Local) generateGo(p Problem, meta Meta, dir string) error {
	tmpl, err := drivers.ReadFile("drivers/golang/driver.go.tmpl")
	if err != nil {
		return fmt.Errorf("read embedded go driver: %w", err)
	}

	var decls, args strings.Builder
	for i, param := range meta.Params {
		gt, err := goType(param.Type)
		if err != nil {
			return err
		}
		switch gt {
		case "*ListNode":
			fmt.Fprintf(&decls, "\tvar raw%d []int\n\tdecode(lines[%d], &raw%d)\n\ta%d := buildList(raw%d)\n", i, i, i, i, i)
		case "*TreeNode":
			fmt.Fprintf(&decls, "\tvar raw%d []*int\n\tdecode(lines[%d], &raw%d)\n\ta%d := buildTree(raw%d)\n", i, i, i, i, i)
		default:
			fmt.Fprintf(&decls, "\tvar a%d %s\n\tdecode(lines[%d], &a%d)\n", i, gt, i, i)
		}
		if i > 0 {
			args.WriteString(", ")
		}
		fmt.Fprintf(&args, "a%d", i)
	}

	rule := RuleFor(p.Slug)
	var call string
	switch {
	case rule.MutatesArg >= 0:
		if rule.MutatesArg >= len(meta.Params) {
			return fmt.Errorf("override for %s names argument %d but there are %d",
				p.Slug, rule.MutatesArg, len(meta.Params))
		}
		// In-place problems answer through a mutated argument; the return value is
		// usually the length of the meaningful prefix.
		call = fmt.Sprintf("\tn := %s(%s)\n\t_ = n\n\tfmt.Println(serialize(prefix(a%d, n)))\n",
			meta.Name, args.String(), rule.MutatesArg)
	case meta.Return.Type == "void":
		call = fmt.Sprintf("\t%s(%s)\n\tfmt.Println(\"null\")\n", meta.Name, args.String())
	default:
		call = fmt.Sprintf("\tfmt.Println(serialize(%s(%s)))\n", meta.Name, args.String())
	}

	body := fmt.Sprintf(`
func run() {
	lines := readLines(%d)
	_ = lines
%s%s}

// prefix trims an in-place answer to the length the solution reported. Types other than
// []int pass through, since only slice problems use this shape.
func prefix(v any, n int) any {
	if s, ok := v.([]int); ok && n >= 0 && n <= len(s) {
		return s[:n]
	}
	return v
}
`, len(meta.Params), decls.String(), call)

	out := append(bytes.TrimRight(tmpl, "\n"), []byte("\n"+body)...)
	if err := os.WriteFile(filepath.Join(dir, goDriverFile), out, 0o644); err != nil {
		return fmt.Errorf("write go driver: %w", err)
	}
	return writeGoMod(dir)
}

// writeGoMod gives the problem folder a module, since `go build` refuses to work
// outside one. Written only when absent, so a user who added dependencies keeps them.
func writeGoMod(dir string) error {
	path := filepath.Join(dir, "go.mod")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	mod := fmt.Sprintf("module %s\n\ngo 1.22\n", filepath.Base(dir))
	if err := os.WriteFile(path, []byte(mod), 0o644); err != nil {
		return fmt.Errorf("write go.mod: %w", err)
	}
	return nil
}

// runGo compiles once, then runs each case against the built binary.
//
// Compiling per case would pay the build cost every time; compiling once and reusing
// the binary keeps a five-case run near the cost of one.
func (l *Local) runGo(ctx context.Context, dir string, cases []TestCase, rule Rule) (Result, error) {
	bin := filepath.Join(dir, "_leetui_bin")

	build := exec.CommandContext(ctx, "go", "build", "-o", bin, ".")
	build.Dir = dir
	var buildErr bytes.Buffer
	build.Stderr = &buildErr

	if err := build.Run(); err != nil {
		// A compiler writes a better message than we could summarise, so it is passed
		// through verbatim.
		msg := strings.TrimSpace(buildErr.String())
		if msg == "" {
			msg = err.Error()
		}
		return Result{CompileErr: msg}, nil
	}
	defer os.Remove(bin)

	started := time.Now()
	result := Result{Cases: make([]CaseResult, 0, len(cases))}
	for _, tc := range cases {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		result.Cases = append(result.Cases, l.runBinaryCase(ctx, dir, bin, nil, tc, rule))
	}
	result.Elapsed = time.Since(started)
	return result, nil
}
