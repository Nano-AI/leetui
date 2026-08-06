package leetcode

import (
	"context"
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Company packs (D-006)
// ---------------------------------------------------------------------------
//
// LeetCode's premium company loop is: pick a company, pick how recently it was asked,
// work the list. leetui models that directly — a company plus a Timeframe is a PACK, and
// a pack is one favoriteSlug on the wire.
//
// Packs are synced ONE AT A TIME, on demand. Pulling all 984 companies × 5 timeframes
// would be ~5,000 requests and forty minutes at the rate limit, to answer a question the
// user asked about one company. Google's largest pack is 2,335 problems — 24 requests,
// a few seconds.

// Timeframe is how recently a company asked a problem. These five are the complete set
// LeetCode offers; anything else returns a null pack.
type Timeframe string

const (
	Thirty       Timeframe = "thirty-days"
	ThreeMonths  Timeframe = "three-months"
	SixMonths    Timeframe = "six-months"
	OlderThanSix Timeframe = "more-than-six-months"
	AllTime      Timeframe = "all"
)

// Timeframes lists them newest-window first, which is the order the website shows and
// the order a user preparing for an interview wants.
func Timeframes() []Timeframe {
	return []Timeframe{Thirty, ThreeMonths, SixMonths, OlderThanSix, AllTime}
}

// Label is the timeframe in prose, for a picker row or a pane label.
func (t Timeframe) Label() string {
	switch t {
	case Thirty:
		return "last 30 days"
	case ThreeMonths:
		return "last 3 months"
	case SixMonths:
		return "last 6 months"
	case OlderThanSix:
		return "over 6 months ago"
	case AllTime:
		return "all time"
	default:
		return string(t)
	}
}

// Valid reports whether t is one of LeetCode's five windows.
func (t Timeframe) Valid() bool {
	for _, k := range Timeframes() {
		if k == t {
			return true
		}
	}
	return false
}

// PackSlug is the favoriteSlug for a company and timeframe, e.g. "google-three-months".
func PackSlug(company string, t Timeframe) string {
	return company + "-" + string(t)
}

// Company is one entry in the company registry.
type Company struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	// QuestionCount is the all-time pack size, as LeetCode reports it. It matches the
	// "-all" pack and is available without Premium, so the browse list can show real
	// sizes before anything is synced.
	QuestionCount int `json:"questionCount"`
}

// PackQuestion is one problem inside a company pack.
type PackQuestion struct {
	Slug       string     `json:"titleSlug"`
	Title      string     `json:"title"`
	FrontendID string     `json:"questionFrontendId"`
	Difficulty Difficulty `json:"difficulty"`
	Status     Status     `json:"status"`
	PaidOnly   bool       `json:"paidOnly"`
	AcRate     float64    `json:"acRate"`
	Tags       []Tag      `json:"topicTags"`

	// Frequency is how often this company asks it, on LeetCode's own scale. It is
	// relative WITHIN a pack — a 0.8 at one company means nothing next to a 0.8 at
	// another — so it is only ever rendered as a rank or a bar, never as a number.
	Frequency float64 `json:"frequency"`
}

// Pack is one page of a company pack.
type Pack struct {
	Total     int            `json:"totalLength"`
	HasMore   bool           `json:"hasMore"`
	Questions []PackQuestion `json:"questions"`
}

// CompanyTags returns the full company registry — 984 entries as of 2026-08-06.
//
// This works SIGNED OUT, so the browse list is populated before the user authenticates
// and a free account can still see which companies exist and how large each list is.
func (c *Client) CompanyTags(ctx context.Context) ([]Company, error) {
	var out struct {
		CompanyTags []Company `json:"companyTags"`
	}
	if err := c.graphql(ctx, "companyTags", qCompanyTags, nil, &out); err != nil {
		return nil, err
	}
	return out.CompanyTags, nil
}

// PackSize returns a pack's problem count, and confirms the pack exists.
//
// An unknown company or timeframe returns ErrNotFound rather than an empty pack, because
// LeetCode answers both with null and a caller that could not tell them apart would
// report a typo as "this company has no problems".
func (c *Client) PackSize(ctx context.Context, company string, t Timeframe) (int, error) {
	var out struct {
		Detail *struct {
			QuestionNumber int `json:"questionNumber"`
		} `json:"favoriteDetailV2"`
	}
	vars := map[string]any{"favoriteSlug": PackSlug(company, t)}
	if err := c.graphql(ctx, "favoriteDetailV2", qFavoriteDetail, vars, &out); err != nil {
		return 0, err
	}
	if out.Detail == nil {
		return 0, fmt.Errorf("company pack %q: %w", PackSlug(company, t), ErrNotFound)
	}
	return out.Detail.QuestionNumber, nil
}

// CompanyPage fetches one page of a company pack.
//
// Without Premium LeetCode returns an EMPTY list rather than an error, so an empty first
// page for a pack that PackSize said was non-empty is the gate. Callers get
// ErrPremiumRequired for that case; deciding it here keeps the "did it work" test in one
// place rather than in every caller.
func (c *Client) CompanyPage(ctx context.Context, company string, t Timeframe, skip, limit int) (Pack, error) {
	var out struct {
		List Pack `json:"favoriteQuestionList"`
	}
	vars := map[string]any{
		"favoriteSlug": PackSlug(company, t),
		"skip":         skip,
		"limit":        limit,
		"version":      "v2",
	}
	if err := c.graphql(ctx, "favoriteQuestionList", qFavoriteQuestionList, vars, &out); err != nil {
		return Pack{}, err
	}
	if len(out.List.Questions) == 0 && out.List.Total > 0 {
		return out.List, ErrPremiumRequired
	}
	return out.List, nil
}

// Summary converts a pack entry into a problem-list row, so a pack can seed problems the
// list sync has not reached yet without a second write path in the store.
func (q PackQuestion) Summary() ProblemSummary {
	return ProblemSummary{
		FrontendID: strings.TrimSpace(q.FrontendID),
		Title:      q.Title,
		Slug:       q.Slug,
		Difficulty: q.Difficulty,
		AcRate:     q.AcRate,
		PaidOnly:   q.PaidOnly,
		Status:     q.Status,
		Tags:       q.Tags,
	}
}
