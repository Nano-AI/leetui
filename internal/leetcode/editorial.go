package leetcode

import (
	"context"
	"fmt"
	"strings"
)

// Solution is LeetCode's official editorial for a problem.
//
// Content is MARKDOWN, not HTML — unlike a problem statement, which is HTML. It carries
// embedded HTML for figures and playground embeds, plus `$$…$$` MathJax. internal/render
// has a separate entry point for it; do not feed it to render.HTML.
type Solution struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	PaidOnly bool   `json:"paidOnly"`

	// CanSeeDetail is the real gate. A premium editorial still returns a node with a
	// title, so the pane can say what it is withholding rather than showing a blank.
	CanSeeDetail bool `json:"canSeeDetail"`

	HasVideoSolution bool `json:"hasVideoSolution"`
}

// Editorial fetches a problem's official solution.
//
// Three distinct outcomes, and the caller needs to tell them apart:
//
//	nil, ErrNotFound        no editorial exists for this problem
//	sol, ErrPremiumRequired one exists but this account cannot read it
//	sol, nil                readable
//
// The gated case returns the Solution as well, because its title and video flag are
// public and worth showing next to the lock.
func (c *Client) Editorial(ctx context.Context, slug string) (*Solution, error) {
	var out struct {
		Question *struct {
			Solution *Solution `json:"solution"`
		} `json:"question"`
	}
	vars := map[string]any{"titleSlug": slug}
	if err := c.graphql(ctx, "questionSolution", qSolution, vars, &out); err != nil {
		return nil, err
	}
	if out.Question == nil || out.Question.Solution == nil {
		return nil, fmt.Errorf("editorial for %q: %w", slug, ErrNotFound)
	}

	sol := out.Question.Solution
	if !sol.CanSeeDetail || strings.TrimSpace(sol.Content) == "" {
		return sol, ErrPremiumRequired
	}
	return sol, nil
}
