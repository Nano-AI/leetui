package leetcode

import (
	"context"
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

// Status returns the signed-in user's account state, including whether they hold
// Premium. A signed-out session is not an error — it returns IsSignedIn false.
func (c *Client) Status(ctx context.Context) (UserStatus, error) {
	var out userStatusResponse
	err := c.graphql(ctx, "globalData", qGlobalData, nil, &out)
	return out.UserStatus, err
}

// ProblemPage fetches one page of the problem set.
//
// Total is the full count, so a caller can size a progress bar before paging.
// An empty categorySlug means all categories.
func (c *Client) ProblemPage(ctx context.Context, skip, limit int) (problems []ProblemSummary, total int, err error) {
	vars := map[string]any{
		"categorySlug": "",
		"skip":         skip,
		"limit":        limit,
		"filters":      map[string]any{},
	}
	var out problemListResponse
	if err := c.graphql(ctx, "problemsetQuestionList", qProblemList, vars, &out); err != nil {
		return nil, 0, err
	}
	return out.ProblemsetQuestionList.Questions, out.ProblemsetQuestionList.Total, nil
}

// Question fetches full detail for one problem, including the HTML statement, starter
// snippets, and the metaData signature that drives local codegen.
//
// Premium-locked problems return ErrPremiumRequired for accounts without Premium.
func (c *Client) Question(ctx context.Context, slug string) (*Problem, error) {
	vars := map[string]any{"titleSlug": slug}
	var out questionResponse
	if err := c.graphql(ctx, "questionData", qQuestionData, vars, &out); err != nil {
		return nil, err
	}
	if out.Question == nil {
		return nil, fmt.Errorf("question %q: %w", slug, ErrNotFound)
	}
	// A paid-only problem with an empty statement means the session cannot see it.
	if out.Question.PaidOnly && strings.TrimSpace(out.Question.Content) == "" {
		return out.Question, ErrPremiumRequired
	}
	return out.Question, nil
}
