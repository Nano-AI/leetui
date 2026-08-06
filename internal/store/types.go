package store

import (
	"errors"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

// ErrNotFound means no problem matched.
var ErrNotFound = errors.New("problem not found")

// Row is a board row: everything needed to render the list without a second query.
type Row struct {
	Slug        string
	FrontendID  string
	NumericID   int
	Title       string
	Difficulty  string
	AcRate      float64
	PaidOnly    bool
	Status      string // "", "ac", "notac"
	Tags        []string
	Companies   []string
	HasSolution bool
	HasVideo    bool

	// HasDetail reports whether the statement has been fetched. The detail pane shows
	// a fetching state rather than an empty one when this is false.
	HasDetail bool

	// Todo reports whether this problem is on the user's list. Filled by the caller from
	// TodoSlugs rather than by the row query, so browsing pays one lookup instead of a
	// correlated subquery per row.
	Todo bool
}

// Solved reports whether the user has an accepted submission.
func (r Row) Solved() bool { return r.Status == string(leetcode.StatusAccepted) }

// Detail is a problem's full record, including the statement.
type Detail struct {
	Row
	QuestionID       string
	Content          string // HTML as LeetCode serves it
	MetaData         string
	SampleTestCase   string
	ExampleTestcases string
	Hints            []string
	Snippets         map[string]string // langSlug -> starter code
}
