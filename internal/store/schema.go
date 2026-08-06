package store

import (
	"context"
	"fmt"
)

// ---------------------------------------------------------------------------
// Schema
// ---------------------------------------------------------------------------

// migrations are applied in order and recorded in schema_version.
//
// APPEND ONLY. Never edit a migration that has shipped — an existing database has
// already run it, and editing it silently diverges old installs from new ones. To
// change something, add a new migration.
var migrations = []string{
	// 1: core problem data.
	`
	CREATE TABLE IF NOT EXISTS problems (
		slug              TEXT PRIMARY KEY,
		frontend_id       TEXT NOT NULL DEFAULT '',
		numeric_id        INTEGER NOT NULL DEFAULT 0,
		question_id       TEXT NOT NULL DEFAULT '',
		title             TEXT NOT NULL DEFAULT '',
		difficulty        TEXT NOT NULL DEFAULT '',
		ac_rate           REAL NOT NULL DEFAULT 0,
		paid_only         INTEGER NOT NULL DEFAULT 0,
		status            TEXT NOT NULL DEFAULT '',
		has_solution      INTEGER NOT NULL DEFAULT 0,
		has_video         INTEGER NOT NULL DEFAULT 0,

		-- Detail columns, filled lazily on first open (D-009: metadata syncs fast,
		-- bodies arrive as you browse).
		content           TEXT NOT NULL DEFAULT '',
		meta_data         TEXT NOT NULL DEFAULT '',
		sample_testcase   TEXT NOT NULL DEFAULT '',
		example_testcases TEXT NOT NULL DEFAULT '',
		hints             TEXT NOT NULL DEFAULT '',
		detail_synced_at  INTEGER NOT NULL DEFAULT 0,

		synced_at         INTEGER NOT NULL DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_problems_numeric ON problems(numeric_id);
	CREATE INDEX IF NOT EXISTS idx_problems_difficulty ON problems(difficulty);
	CREATE INDEX IF NOT EXISTS idx_problems_status ON problems(status);

	CREATE TABLE IF NOT EXISTS tags (
		slug TEXT PRIMARY KEY,
		name TEXT NOT NULL DEFAULT ''
	);
	CREATE TABLE IF NOT EXISTS problem_tags (
		problem_slug TEXT NOT NULL REFERENCES problems(slug) ON DELETE CASCADE,
		tag_slug     TEXT NOT NULL,
		PRIMARY KEY (problem_slug, tag_slug)
	);
	CREATE INDEX IF NOT EXISTS idx_problem_tags_tag ON problem_tags(tag_slug);

	-- Premium company data (D-006, D-008). Populated by inverting company -> problem.
	CREATE TABLE IF NOT EXISTS companies (
		slug TEXT PRIMARY KEY,
		name TEXT NOT NULL DEFAULT ''
	);
	CREATE TABLE IF NOT EXISTS problem_companies (
		problem_slug TEXT NOT NULL REFERENCES problems(slug) ON DELETE CASCADE,
		company_slug TEXT NOT NULL,
		frequency    REAL NOT NULL DEFAULT 0,
		timeframe    TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (problem_slug, company_slug, timeframe)
	);
	CREATE INDEX IF NOT EXISTS idx_problem_companies_company ON problem_companies(company_slug);

	CREATE TABLE IF NOT EXISTS snippets (
		problem_slug TEXT NOT NULL REFERENCES problems(slug) ON DELETE CASCADE,
		lang_slug    TEXT NOT NULL,
		lang         TEXT NOT NULL DEFAULT '',
		code         TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (problem_slug, lang_slug)
	);

	CREATE TABLE IF NOT EXISTS submissions (
		id           TEXT PRIMARY KEY,
		problem_slug TEXT NOT NULL,
		lang         TEXT NOT NULL DEFAULT '',
		verdict      TEXT NOT NULL DEFAULT '',
		runtime      TEXT NOT NULL DEFAULT '',
		memory       TEXT NOT NULL DEFAULT '',
		created_at   INTEGER NOT NULL DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_submissions_problem ON submissions(problem_slug);

	-- Checkpoints so an interrupted sync resumes instead of restarting (D-008).
	CREATE TABLE IF NOT EXISTS sync_state (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL DEFAULT ''
	);
	`,

	// 2: full-text search.
	//
	// Standalone (not external-content) FTS5: the corpus is ~4k rows, so the duplicated
	// text is a few MB, and keeping it standalone avoids the contentless-table rebuild
	// dance on every detail fetch.
	`
	CREATE VIRTUAL TABLE IF NOT EXISTS problems_fts USING fts5(
		slug UNINDEXED,
		title,
		body,
		tokenize = 'porter unicode61'
	);
	`,
}

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`); err != nil {
		return fmt.Errorf("create schema_version: %w", err)
	}

	var current int
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&current)
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	for i := current; i < len(migrations); i++ {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", i+1, err)
		}
		if _, err := tx.ExecContext(ctx, migrations[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_version (version) VALUES (?)`, i+1); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %d: %w", i+1, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", i+1, err)
		}
	}
	return nil
}
