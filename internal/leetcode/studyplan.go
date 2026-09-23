package leetcode

import (
	"context"
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Study plans (D-031)
// ---------------------------------------------------------------------------
//
// A study plan is a curated, ORDERED list — Top Interview 150, LeetCode 75, Graph
// Theory. Where a company pack answers "what does Google ask", a plan answers "what
// should I work through next", and the answer is a curriculum: its subgroups run
// Array/String, then Two Pointers, then Sliding Window, and that sequence is the point.
//
// Two things make plans cheaper than packs, and both come from the wire format:
//
//	ONE REQUEST      studyPlanV2Detail returns every question and every subgroup at once.
//	                 No paging, no page size, no rate-limit dance. Google's pack is 24
//	                 requests; Top Interview 150 is one.
//	SIGNED OUT       the contents are readable without a session. Company packs invert
//	                 this — their registry is free and their contents are premium.
//
// So plans are synced whole, on demand, and a free account gets real content out of them.

// PlanQuestion is one problem inside a study plan.
//
// Unlike PackQuestion there is no frequency: a plan is ordered by its author, not by how
// often anyone asks it. Position within the plan is the ranking, which is why the store
// records a rank rather than a score.
type PlanQuestion struct {
	Slug       string     `json:"titleSlug"`
	Title      string     `json:"title"`
	FrontendID string     `json:"questionFrontendId"`
	Difficulty Difficulty `json:"difficulty"`
	Status     Status     `json:"status"`
	PaidOnly   bool       `json:"paidOnly"`
	Tags       []Tag      `json:"topicTags"`
}

// PlanGroup is one chapter of a plan, e.g. "Array / String".
type PlanGroup struct {
	Slug        string         `json:"slug"`
	Name        string         `json:"name"`
	QuestionNum int            `json:"questionNum"`
	Questions   []PlanQuestion `json:"questions"`
}

// PlanBrief is a plan as it appears in a list, without its questions.
type PlanBrief struct {
	Slug string `json:"slug"`
	Name string `json:"name"`

	// Highlight is LeetCode's own one-line pitch, e.g. "Must-do List for Interview Prep".
	// It is shown in the picker because "LeetCode 75" and "Top 100 Liked" do not, on their
	// own, say how they differ.
	Highlight string `json:"highlight"`

	QuestionNum int  `json:"questionNum"`
	PremiumOnly bool `json:"premiumOnly"`
}

// Plan is a full study plan: its metadata plus every chapter.
type Plan struct {
	PlanBrief
	Groups []PlanGroup `json:"planSubGroups"`
}

// Questions flattens the plan into curriculum order.
//
// Flattening here rather than at each call site keeps the plan's ORDER in one place. The
// index of a question in this slice is its rank, and rank is what makes the board read as
// a curriculum instead of as a numbered list.
func (p Plan) Questions() []PlanQuestion {
	out := make([]PlanQuestion, 0, p.QuestionNum)
	for _, g := range p.Groups {
		out = append(out, g.Questions...)
	}
	return out
}

// GroupOf maps each problem slug to the chapter it sits in, so the board can show which
// part of the curriculum a row belongs to.
func (p Plan) GroupOf() map[string]string {
	out := make(map[string]string, p.QuestionNum)
	for _, g := range p.Groups {
		for _, q := range g.Questions {
			out[q.Slug] = g.Name
		}
	}
	return out
}

// SeedPlans are the plan slugs leetui ships knowing about.
//
// Discovery by tag exists (StudyPlansByTag) but is incomplete — LeetCode's tag vocabulary
// is a short curated set that omits Top 100 Liked, Binary Search, Graph Theory and every
// premium plan. A seed list is what makes the picker useful on a first launch, and the
// registry sync unions the two.
//
// Verified live on 2026-08-09; TestLiveStudyPlanSlugs is what catches LeetCode retiring or
// renaming one. Note "top-sql-50", not "sql-50": the website's URL and its plan slug
// disagree, which is exactly the kind of thing that test is for.
func SeedPlans() []string {
	return []string{
		"top-interview-150",
		"leetcode-75",
		"top-100-liked",
		"top-sql-50",
		"binary-search",
		"dynamic-programming",
		"programming-skills",
		"graph-theory",
		"30-days-of-pandas",
		"30-days-of-javascript",
		"introduction-to-pandas",
		"amazon-spring-23-high-frequency",
		"google-spring-23-high-frequency",
		"premium-algo-100",
		"dynamic-programming-grandmaster",
		"tiktok-spring-23-high-frequency",
		"apple-spring-23-high-frequency",
	}
}

// DiscoveryTags are the tags swept to find plans that are not in SeedPlans.
//
// This is LeetCode's own vocabulary for study plans and it is NOT the topic-tag list:
// "array" and "graph" return nothing, while "beginner" and "interview" return plenty.
// Sweeping it is five requests and picks up plans added after this release.
func DiscoveryTags() []string {
	return []string{"interview", "beginner", "intermediate", "database", "dynamic-programming"}
}

// StudyPlan fetches a plan and all of its questions.
//
// An unknown slug returns ErrNotFound: LeetCode answers it with null, and a caller that
// could not tell that apart from an empty plan would report a typo as "this plan has no
// problems".
//
// A premium plan returns the plan and ErrPremiumRequired, with its groups empty. The plan
// itself is returned rather than dropped so the caller can still name what is gated —
// same bargain as CompanyPage and Editorial (D-006).
func (c *Client) StudyPlan(ctx context.Context, slug string) (Plan, error) {
	var out struct {
		Detail *Plan `json:"studyPlanV2Detail"`
	}
	vars := map[string]any{"planSlug": slug}
	if err := c.graphql(ctx, "studyPlanV2Detail", qStudyPlanDetail, vars, &out); err != nil {
		return Plan{}, err
	}
	if out.Detail == nil {
		return Plan{}, fmt.Errorf("study plan %q: %w", slug, ErrNotFound)
	}
	p := *out.Detail
	p.normalize()
	// A gate may expose chapter metadata while withholding their questions.
	// Counting chapters would accept that response and wipe a cached curriculum.
	count := 0
	for _, g := range p.Groups {
		count += len(g.Questions)
	}
	if count == 0 && p.QuestionNum > 0 {
		return p, ErrPremiumRequired
	}
	return p, nil
}

// normalize rewrites the plan endpoint's enums into the wire strings the rest of leetui
// stores.
//
// THIS ENDPOINT DISAGREES WITH EVERY OTHER ONE. problemsetQuestionList answers "Easy" and
// "ac"; studyPlanV2Detail answers "EASY" and "SOLVED". Difficulty and Status are declared
// as the former (models.go), the board's difficultyOf compares against "Hard"/"Medium",
// and Row.Solved tests for "ac" — so storing the plan's own spelling would produce rows
// that render as Easy whatever they are and never count as solved.
//
// Normalising here, once, is why nothing downstream branches on where a question came
// from. An unrecognised value is left alone rather than guessed at: a new enum member
// should surface as an odd-looking row, not as a silent Easy.
func (p *Plan) normalize() {
	for gi := range p.Groups {
		for qi := range p.Groups[gi].Questions {
			q := &p.Groups[gi].Questions[qi]
			q.Difficulty = normalizeDifficulty(q.Difficulty)
			q.Status = normalizeStatus(q.Status)
		}
	}
}

func normalizeDifficulty(d Difficulty) Difficulty {
	switch strings.ToUpper(string(d)) {
	case "EASY":
		return Easy
	case "MEDIUM":
		return Medium
	case "HARD":
		return Hard
	default:
		return d
	}
}

// normalizeStatus maps the plan endpoint's progress enum onto leetui's.
//
// TO_DO is "not started", which leetui spells as the empty string — the same thing the
// problem list returns for an untouched problem, and what the board's status filter
// tests for.
func normalizeStatus(s Status) Status {
	switch strings.ToUpper(string(s)) {
	case "SOLVED", "AC":
		return StatusAccepted
	case "ATTEMPTED", "TRIED", "NOTAC":
		return StatusAttempted
	case "TO_DO", "TODO", "NOT_STARTED":
		return StatusNone
	default:
		return s
	}
}

// StudyPlansByTag lists the plans carrying one tag. Works signed out.
//
// An unknown tag is an empty list, not an error, so this cannot be used to validate a tag.
func (c *Client) StudyPlansByTag(ctx context.Context, tag string) ([]PlanBrief, error) {
	var out struct {
		List struct {
			Total   int         `json:"total"`
			HasMore bool        `json:"hasMore"`
			Plans   []PlanBrief `json:"studyPlans"`
		} `json:"studyPlansV2ByTag"`
	}
	vars := map[string]any{"tagSlug": tag, "limit": 60}
	if err := c.graphql(ctx, "studyPlansV2ByTag", qStudyPlansByTag, vars, &out); err != nil {
		return nil, err
	}
	return out.List.Plans, nil
}

// Brief reduces a fetched plan to its list entry, so the registry can be filled from a
// detail fetch without a second request.
func (p Plan) Brief() PlanBrief { return p.PlanBrief }

// Summary converts a plan entry into a problem-list row, so a plan can seed problems the
// list sync has not reached yet without a second write path in the store.
//
// AcRate is absent from the plan payload and is left at zero rather than guessed: the
// board reads it from the problems table, where a real sync will have filled it in.
func (q PlanQuestion) Summary() ProblemSummary {
	return ProblemSummary{
		FrontendID: strings.TrimSpace(q.FrontendID),
		Title:      q.Title,
		Slug:       q.Slug,
		Difficulty: q.Difficulty,
		PaidOnly:   q.PaidOnly,
		Status:     q.Status,
		Tags:       q.Tags,
	}
}
