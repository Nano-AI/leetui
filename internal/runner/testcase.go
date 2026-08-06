package runner

import (
	"regexp"
	"strings"
)

// Test cases: where inputs and expected outputs come from.
//
// LeetCode's API gives INPUTS but not EXPECTED OUTPUTS. `exampleTestcases` is
// newline-separated input, one line per parameter, with no answers attached. The
// expected outputs exist only in the statement prose, inside the "Example N:" blocks.
//
// So inputs are read from the API and expected values are scraped from the rendered
// statement. When scraping fails, the case still runs — it just reports output instead
// of judging it, which is more useful than refusing to run.

// ParseCases pairs LeetCode's example inputs with expected outputs scraped from the
// statement.
//
// paramCount comes from metaData and determines how many lines make up one case;
// getting it wrong silently misgroups every case, which is why it is required rather
// than guessed.
func ParseCases(exampleTestcases, statementMarkdown string, paramCount int) []TestCase {
	inputs := groupInputs(exampleTestcases, paramCount)
	expected := scrapeOutputs(statementMarkdown)

	cases := make([]TestCase, 0, len(inputs))
	for i, in := range inputs {
		c := TestCase{Input: in}
		if i < len(expected) {
			c.Expected = expected[i]
		}
		cases = append(cases, c)
	}
	return cases
}

// groupInputs splits the flat input block into one entry per case.
func groupInputs(raw string, paramCount int) []string {
	if paramCount < 1 {
		paramCount = 1
	}
	lines := splitLines(raw)

	var out []string
	for i := 0; i+paramCount <= len(lines); i += paramCount {
		out = append(out, strings.Join(lines[i:i+paramCount], "\n"))
	}
	return out
}

func splitLines(raw string) []string {
	var out []string
	for _, l := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, strings.TrimSpace(l))
		}
	}
	return out
}

// reOutput matches the "Output: ..." lines LeetCode puts in every example block. The
// label survives markdown conversion because the converter strips the <strong> around
// it but keeps the text.
var reOutput = regexp.MustCompile(`(?m)^\s*\*{0,2}Output:?\*{0,2}\s*(.+?)\s*$`)

// scrapeOutputs pulls expected values out of a statement, in order.
func scrapeOutputs(markdown string) []string {
	var out []string
	for _, m := range reOutput.FindAllStringSubmatch(markdown, -1) {
		v := strings.TrimSpace(m[1])
		// Trailing prose sometimes rides along on the same line.
		if i := strings.Index(v, "  "); i > 0 {
			v = strings.TrimSpace(v[:i])
		}
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

// HasExpected reports whether any case carries an answer to check against.
//
// A set with none can still be run — it prints output instead of judging it — but it
// cannot pass or fail, which is worth telling the caller apart from an empty set.
func HasExpected(cases []TestCase) bool {
	for _, c := range cases {
		if strings.TrimSpace(c.Expected) != "" {
			return true
		}
	}
	return false
}

// FormatCases renders cases back into testcases.txt.
//
// The format is deliberately plain so it can be hand-edited: input lines, then a line
// holding only "output:", then the expected value, then a blank line between cases.
func FormatCases(cases []TestCase) string {
	var b strings.Builder
	for i, c := range cases {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(strings.TrimRight(c.Input, "\n"))
		b.WriteString("\noutput:\n")
		b.WriteString(c.Expected)
		b.WriteString("\n")
	}
	return b.String()
}

// LoadCases reads back what FormatCases wrote, so a user's hand-added cases are honoured.
func LoadCases(text string) []TestCase {
	var (
		cases   []TestCase
		current TestCase
		inOut   bool
		input   []string
	)

	flush := func() {
		if len(input) == 0 && current.Expected == "" {
			return
		}
		current.Input = strings.Join(input, "\n")
		cases = append(cases, current)
		current, input, inOut = TestCase{}, nil, false
	}

	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			if inOut {
				flush()
			}
		case strings.EqualFold(trimmed, "output:"):
			inOut = true
		case inOut:
			current.Expected = trimmed
		default:
			input = append(input, trimmed)
		}
	}
	flush()
	return cases
}
