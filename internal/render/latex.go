package render

import (
	"regexp"
	"strings"
)

// LaTeX handling (D-007).
//
// LeetCode statements and editorials lean on MathJax, mostly for constraints
// ("1 <= nums.length <= 10^4") and complexity ("O(n \log n)"). A terminal cannot render
// MathJax, and dropping it loses the constraint bounds that matter most when reading a
// problem. So it is approximated in Unicode.
//
// This is deliberately shallow. It covers the operators LeetCode actually uses and
// leaves anything exotic as-is — a visible "\frac{a}{b}" is more useful than a silently
// mangled one.

var (
	// $...$ and \(...\) inline math, and \[...\] display math.
	reInlineMath  = regexp.MustCompile(`\$([^$]{1,200})\$`)
	reParenMath   = regexp.MustCompile(`\\\(([^)]{1,200}?)\\\)`)
	reBracketMath = regexp.MustCompile(`\\\[([^\]]{1,400}?)\\\]`)

	// ^{...} and _{...} groups, plus bare ^2 / _i.
	reSupGroup = regexp.MustCompile(`\^\{([^}]{1,20})\}`)
	reSubGroup = regexp.MustCompile(`_\{([^}]{1,20})\}`)
	reSupChar  = regexp.MustCompile(`\^([0-9a-zA-Z])`)
	reSubChar  = regexp.MustCompile(`_([0-9a-zA-Z])`)
)

// symbols maps LaTeX commands to Unicode. Longest keys must be replaced first, which
// replaceSymbols handles by iterating a sorted list.
var symbols = map[string]string{
	`\leqslant`: "≤", `\geqslant`: "≥",
	`\times`: "×", `\cdot`: "·", `\div`: "÷", `\pm`: "±",
	`\leq`: "≤", `\geq`: "≥", `\neq`: "≠", `\approx`: "≈", `\equiv`: "≡",
	`\le`: "≤", `\ge`: "≥", `\ne`: "≠",
	`\infty`: "∞", `\sum`: "Σ", `\prod`: "Π", `\sqrt`: "√",
	`\alpha`: "α", `\beta`: "β", `\gamma`: "γ", `\delta`: "δ",
	`\epsilon`: "ε", `\theta`: "θ", `\lambda`: "λ", `\mu`: "μ",
	`\pi`: "π", `\sigma`: "σ", `\phi`: "φ", `\omega`: "ω",
	`\rightarrow`: "→", `\leftarrow`: "←", `\Rightarrow`: "⇒", `\to`: "→",
	`\in`: "∈", `\notin`: "∉", `\subset`: "⊂", `\subseteq`: "⊆",
	`\cup`: "∪", `\cap`: "∩", `\emptyset`: "∅", `\forall`: "∀", `\exists`: "∃",
	`\lfloor`: "⌊", `\rfloor`: "⌋", `\lceil`: "⌈", `\rceil`: "⌉",
	`\ldots`: "…", `\dots`: "…", `\cdots`: "⋯",
	`\log`: "log", `\ln`: "ln", `\max`: "max", `\min`: "min", `\mod`: "mod",
	`\text`: "", `\mathit`: "", `\mathrm`: "", `\left`: "", `\right`: "",
	`\,`: " ", `\;`: " ", `\!`: "", `\ `: " ",
}

// sortedSymbols is symbols' keys, longest first, so `\leq` never matches inside
// `\leqslant`.
var sortedSymbols = func() []string {
	keys := make([]string, 0, len(symbols))
	for k := range symbols {
		keys = append(keys, k)
	}
	// Insertion sort by descending length; the list is small and this runs once.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && len(keys[j]) > len(keys[j-1]); j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}()

// approximateLaTeX converts math notation in a text run to Unicode.
func approximateLaTeX(s string) string {
	if !strings.ContainsAny(s, `$\^_`) {
		return s // fast path: most text has no math at all
	}

	s = reBracketMath.ReplaceAllString(s, "$1")
	s = reParenMath.ReplaceAllString(s, "$1")
	s = reInlineMath.ReplaceAllString(s, "$1")

	s = replaceSymbols(s)

	s = reSupGroup.ReplaceAllStringFunc(s, func(m string) string {
		return toSuperscript(reSupGroup.FindStringSubmatch(m)[1])
	})
	s = reSubGroup.ReplaceAllStringFunc(s, func(m string) string {
		return toSubscript(reSubGroup.FindStringSubmatch(m)[1])
	})
	s = reSupChar.ReplaceAllStringFunc(s, func(m string) string {
		return toSuperscript(m[1:])
	})
	s = reSubChar.ReplaceAllStringFunc(s, func(m string) string {
		return toSubscript(m[1:])
	})

	// Braces left over from \text{...} and friends carry no meaning once the command
	// is gone.
	s = strings.NewReplacer("{", "", "}", "").Replace(s)
	return s
}

func replaceSymbols(s string) string {
	for _, k := range sortedSymbols {
		if strings.Contains(s, k) {
			s = strings.ReplaceAll(s, k, symbols[k])
		}
	}
	return s
}

var (
	superscripts = map[rune]rune{
		'0': '⁰', '1': '¹', '2': '²', '3': '³', '4': '⁴',
		'5': '⁵', '6': '⁶', '7': '⁷', '8': '⁸', '9': '⁹',
		'+': '⁺', '-': '⁻', '=': '⁼', '(': '⁽', ')': '⁾',
		'n': 'ⁿ', 'i': 'ⁱ',
	}
	subscripts = map[rune]rune{
		'0': '₀', '1': '₁', '2': '₂', '3': '₃', '4': '₄',
		'5': '₅', '6': '₆', '7': '₇', '8': '₈', '9': '₉',
		'+': '₊', '-': '₋', '=': '₌', '(': '₍', ')': '₎',
		'a': 'ₐ', 'e': 'ₑ', 'i': 'ᵢ', 'j': 'ⱼ', 'k': 'ₖ',
		'm': 'ₘ', 'n': 'ₙ', 'o': 'ₒ', 'p': 'ₚ', 'r': 'ᵣ',
		's': 'ₛ', 't': 'ₜ', 'u': 'ᵤ', 'v': 'ᵥ', 'x': 'ₓ',
	}
)

// toSuperscript renders text as superscript where Unicode allows.
//
// Falls back to caret notation for characters with no superscript form: "10^k" reads
// correctly, whereas silently dropping the k would change the meaning.
func toSuperscript(s string) string {
	var b strings.Builder
	for _, r := range s {
		if sup, ok := superscripts[r]; ok {
			b.WriteRune(sup)
			continue
		}
		return "^" + s
	}
	return b.String()
}

// toSubscript renders text as subscript where Unicode allows, falling back to
// underscore notation.
func toSubscript(s string) string {
	var b strings.Builder
	for _, r := range s {
		if sub, ok := subscripts[r]; ok {
			b.WriteRune(sub)
			continue
		}
		return "_" + s
	}
	return b.String()
}
