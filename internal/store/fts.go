package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// reindexTx replaces a problem's FTS row from its cached data.
func reindexTx(ctx context.Context, tx *sql.Tx, slug string) error {
	// Partial summary and contest payloads must not erase the statement or tags.
	var title, content, tags string
	if err := tx.QueryRowContext(ctx, `SELECT p.title, p.content,
		COALESCE((SELECT group_concat(t.name, ' ') FROM problem_tags pt
		JOIN tags t ON t.slug = pt.tag_slug WHERE pt.problem_slug = p.slug), '')
		FROM problems p WHERE p.slug = ?`, slug).Scan(&title, &content, &tags); err != nil {
		return fmt.Errorf("read index source for %s: %w", slug, err)
	}
	body := stripHTML(content) + " " + tags
	if _, err := tx.ExecContext(ctx, `DELETE FROM problems_fts WHERE slug = ?`, slug); err != nil {
		return fmt.Errorf("clear fts for %s: %w", slug, err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO problems_fts (slug, title, body) VALUES (?,?,?)`, slug, title, body); err != nil {
		return fmt.Errorf("index %s: %w", slug, err)
	}
	return nil
}

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
