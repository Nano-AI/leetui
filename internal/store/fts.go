package store

import "strings"

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// ftsQuery converts user input into an FTS5 MATCH expression.
//
// User text is never interpolated raw: FTS5 has its own operator syntax (AND, OR, NOT,
// NEAR, quotes, colons) and a stray quote or hyphen from a problem title would be a
// syntax error, not a search. Each term is quoted and given a prefix wildcard so search
// narrows as you type.
func ftsQuery(text string) string {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return ""
	}
	terms := make([]string, 0, len(fields))
	for _, f := range fields {
		// Strip FTS5-significant characters, then quote what remains.
		f = strings.Map(func(r rune) rune {
			switch r {
			case '"', '\'', '*', '(', ')', ':', '^', '-':
				return -1
			}
			return r
		}, f)
		if f == "" {
			continue
		}
		terms = append(terms, `"`+f+`"*`)
	}
	if len(terms) == 0 {
		return ""
	}
	return strings.Join(terms, " AND ")
}
