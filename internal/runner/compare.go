package runner

import (
	"encoding/json"
	"math"
	"reflect"
	"sort"
	"strings"
)

// Comparing output to expectation.
//
// This is where the local runner is most likely to disagree with the judge, and the
// disagreement is never the user's fault. `metaData` describes shapes, not semantics: it
// cannot say that a problem accepts any ordering, or tolerates float error, or answers
// through a mutated argument. Overrides supply what it omits (overrides.go).
//
// When comparison is uncertain the answer is "verify remotely", never "wrong" (D-003).

// Compare reports whether actual satisfies expected under the given rules.
func Compare(actual, expected string, rule Rule) bool {
	a, e := strings.TrimSpace(actual), strings.TrimSpace(expected)
	if a == e {
		return true
	}

	av, aok := parseJSON(a)
	ev, eok := parseJSON(e)
	if !aok || !eok {
		// Not JSON on one side; fall back to whitespace-insensitive text.
		return normalizeSpace(a) == normalizeSpace(e)
	}

	if rule.FloatTolerance > 0 {
		if ok, applied := compareFloats(av, ev, rule.FloatTolerance); applied {
			return ok
		}
	}
	if rule.Unordered {
		return compareUnordered(av, ev)
	}
	return reflect.DeepEqual(av, ev)
}

func parseJSON(s string) (any, bool) {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil, false
	}
	return v, true
}

func normalizeSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// compareFloats handles answers judged to a tolerance, e.g. median or average problems.
// applied is false when neither side is numeric, so the caller can fall through.
func compareFloats(a, e any, eps float64) (ok, applied bool) {
	af, aIsNum := a.(float64)
	ef, eIsNum := e.(float64)
	if aIsNum && eIsNum {
		return math.Abs(af-ef) <= eps, true
	}

	al, aIsList := a.([]any)
	el, eIsList := e.([]any)
	if aIsList && eIsList {
		if len(al) != len(el) {
			return false, true
		}
		for i := range al {
			sub, subApplied := compareFloats(al[i], el[i], eps)
			if !subApplied {
				return reflect.DeepEqual(al, el), true
			}
			if !sub {
				return false, true
			}
		}
		return true, true
	}
	return false, false
}

// compareUnordered treats top-level lists as multisets, recursively.
//
// Used for problems that say "in any order" — permutations, subsets, group anagrams.
func compareUnordered(a, e any) bool {
	al, aok := a.([]any)
	el, eok := e.([]any)
	if !aok || !eok {
		return reflect.DeepEqual(a, e)
	}
	if len(al) != len(el) {
		return false
	}

	ak := canonicalKeys(al)
	ek := canonicalKeys(el)
	sort.Strings(ak)
	sort.Strings(ek)
	return reflect.DeepEqual(ak, ek)
}

// canonicalKeys renders each element to a stable string, sorting nested lists first so
// [[1,2],[3]] and [[2,1],[3]] compare equal.
func canonicalKeys(items []any) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = canonical(item)
	}
	return out
}

func canonical(v any) string {
	list, ok := v.([]any)
	if !ok {
		b, _ := json.Marshal(v)
		return string(b)
	}
	parts := make([]string, len(list))
	for i, item := range list {
		parts[i] = canonical(item)
	}
	sort.Strings(parts)
	return "[" + strings.Join(parts, ",") + "]"
}
