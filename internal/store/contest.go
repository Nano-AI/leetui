package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

// Contests (D-036)
//
// The registry mirrors study plans: a merge-not-replace upsert for the schedule, and a
// delete-then-reinsert for one contest's contents. What differs is that a contest's rows
// must survive without a matching problems row, because a live contest's problems are not
// in the problem set yet. Everything the solver needs — title, question id, credit — is
// therefore stored here rather than joined for.

// Contest is one contest as the board sees it.
type Contest struct {
	Slug  string
	Title string

	// StartTime and Duration are seconds, as the API sends them.
	StartTime int64
	Duration  int

	IsVirtual  bool
	HasPremium bool

	// Stored is how many of the contest's questions this database holds. Zero means the
	// contest has been seen in the schedule but never opened — which is every upcoming
	// contest, since the questions are not readable before the start.
	Stored int
	// Solved counts the questions marked solved for this contest.
	Solved int
}

// Brief converts back to the API shape, so the phase and countdown helpers are written
// once and used by the store, the CLI, and the TUI alike.
func (c Contest) Brief() leetcode.ContestBrief {
	return leetcode.ContestBrief{
		Title:     c.Title,
		Slug:      c.Slug,
		StartTime: c.StartTime,
		Duration:  c.Duration,
	}
}

// Start is when the contest opens.
func (c Contest) Start() time.Time { return c.Brief().Start() }

// PhaseAt reports whether the contest is upcoming, live, or ended at a given moment.
func (c Contest) PhaseAt(now time.Time) leetcode.Phase { return c.Brief().PhaseAt(now) }

// Remaining is the time until the contest's next transition.
func (c Contest) Remaining(now time.Time) time.Duration { return c.Brief().Remaining(now) }

// ContestQuestion is one problem in a contest, as stored.
type ContestQuestion struct {
	Slug  string
	Title string

	// QuestionID is LeetCode's internal id, kept here because a live contest's problem
	// has no row in problems to read it from and the submit endpoint cannot work without
	// it. This is the one field that makes submitting during a contest possible offline.
	QuestionID string

	// Credit is the scoring weight, which is also the running order.
	Credit int
	Solved bool
}

// UpsertContests merges schedule entries.
//
// Merges rather than replaces, for the reason UpsertPlans does: upcomingContests is a
// short window onto a schedule this table also remembers the past of, and a replace would
// delete every contest already recorded the moment the next sync ran.
//
// A zero start_time is not written over a known one — a brief from a list is allowed to be
// less complete than a detail fetch, and a contest whose start is forgotten is one whose
// countdown breaks.
func (s *Store) UpsertContests(ctx context.Context, items []leetcode.ContestBrief) error {
	if len(items) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin contest upsert: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO contests (slug, title, start_time, duration)
		VALUES (?,?,?,?)
		ON CONFLICT(slug) DO UPDATE SET
			title      = excluded.title,
			start_time = CASE WHEN excluded.start_time > 0 THEN excluded.start_time ELSE contests.start_time END,
			duration   = CASE WHEN excluded.duration > 0 THEN excluded.duration ELSE contests.duration END`)
	if err != nil {
		return fmt.Errorf("prepare contest upsert: %w", err)
	}
	defer stmt.Close()

	for _, c := range items {
		if c.Slug == "" {
			continue
		}
		if _, err := stmt.ExecContext(ctx, c.Slug, c.Title, c.StartTime, c.Duration); err != nil {
			return fmt.Errorf("upsert contest %s: %w", c.Slug, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit contest upsert: %w", err)
	}
	return nil
}

// SetContest replaces one contest's questions.
//
// Delete-then-reinsert inside one transaction, the same shape as SetPlan. A contest's
// question list is small and arrives whole, so there is no half-written state to defend
// against: the contest lands or it does not.
//
// EMPTY IS A NO-OP, NOT A WIPE. Before the start LeetCode answers with an empty list
// rather than an error, and treating that as truth would erase a contest already pulled —
// which is exactly what a stray refresh during a live contest would do.
func (s *Store) SetContest(ctx context.Context, c *leetcode.Contest) error {
	if c == nil || c.Slug == "" {
		return fmt.Errorf("set contest: no contest given")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin set contest: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO contests (slug, title, start_time, duration, is_virtual, has_premium, synced_at)
		VALUES (?,?,?,?,?,?,?)
		ON CONFLICT(slug) DO UPDATE SET
			title       = excluded.title,
			start_time  = CASE WHEN excluded.start_time > 0 THEN excluded.start_time ELSE contests.start_time END,
			duration    = CASE WHEN excluded.duration > 0 THEN excluded.duration ELSE contests.duration END,
			is_virtual  = excluded.is_virtual,
			has_premium = excluded.has_premium,
			synced_at   = excluded.synced_at`,
		c.Slug, c.Title, c.StartTime, c.Duration,
		boolInt(c.IsVirtual), boolInt(c.ContainsPremium), time.Now().Unix()); err != nil {
		return fmt.Errorf("upsert contest %s: %w", c.Slug, err)
	}

	if len(c.Questions) == 0 {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit set contest: %w", err)
		}
		return nil
	}

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM problem_contests WHERE contest_slug = ?`, c.Slug); err != nil {
		return fmt.Errorf("clear contest %s: %w", c.Slug, err)
	}

	// Seed a bare problems row so the board can show a contest problem the list sync has
	// not reached. INSERT OR IGNORE leaves a real synced row untouched — the same trick
	// SetPlan and SetPack use.
	seed, err := tx.PrepareContext(ctx,
		`INSERT OR IGNORE INTO problems (slug, title) VALUES (?,?)`)
	if err != nil {
		return fmt.Errorf("prepare contest seed: %w", err)
	}
	defer seed.Close()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO problem_contests
			(problem_slug, contest_slug, question_id, title, credit, solved)
		VALUES (?,?,?,?,?,?)`)
	if err != nil {
		return fmt.Errorf("prepare contest questions: %w", err)
	}
	defer stmt.Close()

	for _, q := range c.Questions {
		if q.Slug == "" {
			continue
		}
		if _, err := seed.ExecContext(ctx, q.Slug, q.Title); err != nil {
			return fmt.Errorf("seed contest problem %s: %w", q.Slug, err)
		}
		// Index it, or search cannot see it. A live contest's problems exist ONLY as
		// these seeded rows, and the FTS table is a standalone one that a plain INSERT
		// into problems does not touch — so without this, typing a contest problem's own
		// name on a contest-filtered board returns nothing.
		if err := reindexTx(ctx, tx, q.Slug); err != nil {
			return err
		}
		if _, err := stmt.ExecContext(ctx, q.Slug, c.Slug, q.QuestionID,
			q.Title, q.Credit, boolInt(q.IsAC)); err != nil {
			return fmt.Errorf("insert contest question %s: %w", q.Slug, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit set contest: %w", err)
	}
	return nil
}

// Contests returns the schedule, newest first, optionally narrowed by a substring.
//
// Newest first rather than "started first" as Plans does: a contest's relevance is its
// date, and the one you want is almost always the next one or the last one. Both sit at
// the top of a descending sort, which is why it beats sorting by progress here.
func (s *Store) Contests(ctx context.Context, query string) ([]Contest, error) {
	sqlText := `
		SELECT c.slug, c.title, c.start_time, c.duration, c.is_virtual, c.has_premium,
		       COALESCE((SELECT COUNT(*) FROM problem_contests pc
		                 WHERE pc.contest_slug = c.slug), 0) AS stored,
		       COALESCE((SELECT COUNT(*) FROM problem_contests pc
		                 WHERE pc.contest_slug = c.slug AND pc.solved = 1), 0) AS solved
		FROM contests c`
	var args []any

	if q := strings.TrimSpace(strings.ToLower(query)); q != "" {
		sqlText += ` WHERE LOWER(c.title) LIKE ? OR LOWER(c.slug) LIKE ?`
		args = append(args, "%"+q+"%", "%"+q+"%")
	}
	sqlText += ` ORDER BY c.start_time DESC, c.slug`

	rows, err := s.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("query contests: %w", err)
	}
	defer rows.Close()

	var out []Contest
	for rows.Next() {
		var c Contest
		if err := rows.Scan(&c.Slug, &c.Title, &c.StartTime, &c.Duration,
			&c.IsVirtual, &c.HasPremium, &c.Stored, &c.Solved); err != nil {
			return nil, fmt.Errorf("scan contest: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ContestOf returns one contest, or ErrNotFound.
func (s *Store) ContestOf(ctx context.Context, slug string) (Contest, error) {
	var c Contest
	err := s.db.QueryRowContext(ctx, `
		SELECT c.slug, c.title, c.start_time, c.duration, c.is_virtual, c.has_premium,
		       COALESCE((SELECT COUNT(*) FROM problem_contests pc
		                 WHERE pc.contest_slug = c.slug), 0),
		       COALESCE((SELECT COUNT(*) FROM problem_contests pc
		                 WHERE pc.contest_slug = c.slug AND pc.solved = 1), 0)
		FROM contests c WHERE c.slug = ?`, slug).
		Scan(&c.Slug, &c.Title, &c.StartTime, &c.Duration,
			&c.IsVirtual, &c.HasPremium, &c.Stored, &c.Solved)
	if err == sql.ErrNoRows {
		return Contest{}, ErrNotFound
	}
	if err != nil {
		return Contest{}, fmt.Errorf("get contest %s: %w", slug, err)
	}
	return c, nil
}

// ContestQuestions returns one contest's questions in its running order.
func (s *Store) ContestQuestions(ctx context.Context, slug string) ([]ContestQuestion, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT problem_slug, title, question_id, credit, solved
		FROM problem_contests WHERE contest_slug = ?
		ORDER BY credit, problem_slug`, slug)
	if err != nil {
		return nil, fmt.Errorf("query contest questions: %w", err)
	}
	defer rows.Close()

	var out []ContestQuestion
	for rows.Next() {
		var q ContestQuestion
		if err := rows.Scan(&q.Slug, &q.Title, &q.QuestionID, &q.Credit, &q.Solved); err != nil {
			return nil, fmt.Errorf("scan contest question: %w", err)
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

// SetContestSolved records that a contest question was accepted.
//
// Separate from SetStatus on problems, and both are written when a contest submission is
// accepted. The contest table is the one that must be right during the contest: a live
// contest's problem may have no meaningful row in problems yet, and the scoreboard the
// user is watching is this one.
func (s *Store) SetContestSolved(ctx context.Context, contest, problem string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE problem_contests SET solved = 1
		 WHERE contest_slug = ? AND problem_slug = ?`, contest, problem)
	if err != nil {
		return fmt.Errorf("mark contest solved %s/%s: %w", contest, problem, err)
	}
	return nil
}

// boolInt stores a bool the way every other table here does.
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
