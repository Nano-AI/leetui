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

	// 3: premium content — the company registry and cached editorials (D-006).
	//
	// question_count is LeetCode's own all-time figure and arrives with the registry,
	// which is readable signed out. It is stored separately from the count of rows in
	// problem_companies, because the two answer different questions: how big the pack is
	// on the site, versus how much of it has been pulled down here.
	`
	ALTER TABLE companies ADD COLUMN question_count INTEGER NOT NULL DEFAULT 0;

	CREATE TABLE IF NOT EXISTS editorials (
		problem_slug TEXT PRIMARY KEY REFERENCES problems(slug) ON DELETE CASCADE,
		solution_id  TEXT NOT NULL DEFAULT '',
		title        TEXT NOT NULL DEFAULT '',
		content      TEXT NOT NULL DEFAULT '',
		paid_only    INTEGER NOT NULL DEFAULT 0,
		can_see      INTEGER NOT NULL DEFAULT 0,
		has_video    INTEGER NOT NULL DEFAULT 0,
		synced_at    INTEGER NOT NULL DEFAULT 0
	);
	`,

	// 4: the todo list — problems the user means to get to.
	//
	// Deliberately NOT a column on problems. A todo is the user's own data, and the
	// problems table is a cache of LeetCode's: a re-sync rewrites it wholesale, and a
	// list someone curated must never be collateral damage of a refresh.
	`
	CREATE TABLE IF NOT EXISTS todo (
		problem_slug TEXT PRIMARY KEY,
		note         TEXT NOT NULL DEFAULT '',
		added_at     INTEGER NOT NULL DEFAULT 0
	);
	`,

	// 5: study plans (D-031) — curated, ordered lists like Top Interview 150.
	//
	// Deliberately its own pair of tables rather than a reuse of companies /
	// problem_companies. The two look alike and are not: a pack is keyed by a timeframe
	// and ranked by frequency, a plan has no timeframe and is ranked by its author's
	// running order. Folding them together would mean a timeframe column that is always
	// empty for plans and a frequency column that is always zero.
	//
	// plan_rank, not rank: RANK() is a window function, and a bare `rank` in an ORDER BY
	// is the kind of thing that parses today and stops parsing later. group_name, not
	// group, because GROUP is reserved outright.
	`
	CREATE TABLE IF NOT EXISTS study_plans (
		slug         TEXT PRIMARY KEY,
		name         TEXT NOT NULL DEFAULT '',
		highlight    TEXT NOT NULL DEFAULT '',
		question_num INTEGER NOT NULL DEFAULT 0,
		premium_only INTEGER NOT NULL DEFAULT 0
	);

	-- No timeframe in the primary key: a problem appears at most once in a plan, which is
	-- what lets plan_rank be a plain position rather than a per-window score.
	CREATE TABLE IF NOT EXISTS problem_plans (
		problem_slug TEXT NOT NULL REFERENCES problems(slug) ON DELETE CASCADE,
		plan_slug    TEXT NOT NULL,
		plan_rank    INTEGER NOT NULL DEFAULT 0,
		group_name   TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (problem_slug, plan_slug)
	);
	CREATE INDEX IF NOT EXISTS idx_problem_plans_plan ON problem_plans(plan_slug);
	`,

	// 6: importance marks — the user's verdict on whether a problem was worth doing
	// (D-032).
	//
	// Its own table, and NOT a column on todo, because the two say different things. A
	// todo is a queue you drain: done means off the list. A mark is a judgment that
	// SURVIVES solving — "solved, and worth doing again" is the whole point, and it is a
	// state the todo list cannot express.
	//
	// No foreign key, for the same reason todo has none (D-022): an agent may mark a
	// problem this machine has not synced yet, and the mark has to survive until it does.
	`
	CREATE TABLE IF NOT EXISTS marks (
		problem_slug TEXT PRIMARY KEY,
		mark         TEXT NOT NULL DEFAULT '',
		note         TEXT NOT NULL DEFAULT '',
		marked_at    INTEGER NOT NULL DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_marks_mark ON marks(mark);
	`,

	// 7: contests (D-036) — the schedule, and the problems in each contest.
	//
	// Its own pair of tables rather than a reuse of study_plans / problem_plans, for the
	// reason migration 5 gives about packs: the two look alike and are not. A plan is a
	// curriculum with no clock; a contest is defined by one. start_time and duration are
	// the whole feature, and a plan has nowhere to put them.
	//
	// start_time and duration are SECONDS, matching what the API sends. Storing
	// milliseconds here and seconds there is how a countdown ends up 1000x wrong.
	//
	// No foreign key from problem_contests to problems, unlike problem_plans. A live
	// contest's problems do not exist in the problems table yet — they are not in the
	// problem set until the contest ends — so a foreign key would reject the exact rows
	// this feature exists to store. This is the todo/marks bargain (D-022) for the same
	// reason: the row has to survive until a sync catches up.
	//
	// credit, not rank: the contest's own scoring weight is also its running order, so
	// one column carries both and there is no second number to keep in step.
	`
	CREATE TABLE IF NOT EXISTS contests (
		slug        TEXT PRIMARY KEY,
		title       TEXT NOT NULL DEFAULT '',
		start_time  INTEGER NOT NULL DEFAULT 0,
		duration    INTEGER NOT NULL DEFAULT 0,
		is_virtual  INTEGER NOT NULL DEFAULT 0,
		has_premium INTEGER NOT NULL DEFAULT 0,
		synced_at   INTEGER NOT NULL DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_contests_start ON contests(start_time);

	CREATE TABLE IF NOT EXISTS problem_contests (
		problem_slug  TEXT NOT NULL,
		contest_slug  TEXT NOT NULL,
		question_id   TEXT NOT NULL DEFAULT '',
		title         TEXT NOT NULL DEFAULT '',
		credit        INTEGER NOT NULL DEFAULT 0,
		solved        INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (problem_slug, contest_slug)
	);
	CREATE INDEX IF NOT EXISTS idx_problem_contests_contest ON problem_contests(contest_slug);
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
