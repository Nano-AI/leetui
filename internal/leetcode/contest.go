package leetcode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Contests (D-036)
//
// A contest is the one LeetCode surface where the clock is part of the problem. Everything
// here is shaped by that: times are read from the API rather than assumed, and the phase a
// contest is in is computed from a caller-supplied `now` so the TUI can tick it and a test
// can pin it.

// ContestBrief is a contest's schedule entry — what a list needs and nothing more.
type ContestBrief struct {
	Title string `json:"title"`
	Slug  string `json:"titleSlug"`

	// StartTime is a Unix SECOND, not a millisecond. LeetCode sends seconds here and
	// milliseconds elsewhere in its API; mixing them up puts the next contest in 1970.
	StartTime int64 `json:"startTime"`

	// Duration is in seconds. 5400 for both weeklies and biweeklies today. Read, never
	// assumed — a hardcoded 90 minutes is a countdown that lies the first time LeetCode
	// runs a contest of a different length.
	Duration int `json:"duration"`
}

// Start is when the contest opens.
func (b ContestBrief) Start() time.Time { return time.Unix(b.StartTime, 0) }

// End is when the contest closes.
func (b ContestBrief) End() time.Time { return b.Start().Add(time.Duration(b.Duration) * time.Second) }

// Phase is where a contest sits relative to the clock.
type Phase int

// The three phases a contest can be in.
const (
	// PhaseUpcoming is before the start. The questions are not readable yet.
	PhaseUpcoming Phase = iota
	// PhaseLive is between start and end. This is the only phase in which a submission
	// scores.
	PhaseLive
	// PhaseEnded is after the end. The questions stay readable and become ordinary
	// problems, so upsolving needs nothing special.
	PhaseEnded
)

// String names the phase for display.
func (p Phase) String() string {
	switch p {
	case PhaseUpcoming:
		return "upcoming"
	case PhaseLive:
		return "live"
	case PhaseEnded:
		return "ended"
	}
	return "unknown"
}

// PhaseAt reports the contest's phase at a given moment.
//
// Takes `now` rather than calling time.Now: the countdown is redrawn on a tick and the
// tests pin a fixed clock. A function that reads the wall clock itself cannot do either.
func (b ContestBrief) PhaseAt(now time.Time) Phase {
	switch {
	case now.Before(b.Start()):
		return PhaseUpcoming
	case now.Before(b.End()):
		return PhaseLive
	default:
		return PhaseEnded
	}
}

// Remaining is the time until the next transition — until the start while upcoming, until
// the end while live, and zero once ended.
//
// Never negative. A negative duration formats as "-1m23s", which on a countdown reads as a
// bug rather than as "it is over".
func (b ContestBrief) Remaining(now time.Time) time.Duration {
	var d time.Duration
	switch b.PhaseAt(now) {
	case PhaseUpcoming:
		d = b.Start().Sub(now)
	case PhaseLive:
		d = b.End().Sub(now)
	default:
		return 0
	}
	if d < 0 {
		return 0
	}
	return d
}

// ContestQuestion is one problem in a contest.
type ContestQuestion struct {
	// QuestionID is LeetCode's INTERNAL id, which is what the submit endpoint wants.
	// The contest API gives no frontend id at all — during a live contest the problem
	// has not been assigned a number yet, which is why the board shows credit instead.
	QuestionID string `json:"questionId"`
	Title      string `json:"title"`
	Slug       string `json:"titleSlug"`

	// Credit is the contest's scoring weight: 3/4/5/6 across a weekly's four problems.
	// It is the running order and the closest thing to a difficulty the contest API
	// offers, since the questions carry no difficulty field.
	Credit int `json:"credit"`

	// IsAC is the SIGNED-IN user's own progress. Signed out it is false for every
	// question, which is the truth for an anonymous caller rather than a missing value.
	IsAC bool `json:"isAc"`

	// Difficulty is filled in from the REST endpoint when signed in, and is EMPTY
	// otherwise — contestQuestionList does not carry one. Empty renders as unknown.
	Difficulty Difficulty `json:"-"`
}

// Summary converts a contest question into a problem-list row, so a contest can seed
// problems the list sync has not reached yet without a second write path in the store.
//
// Difficulty is passed through rather than derived from credit. Deriving it looks
// obvious — 3/4/5/6 onto Easy/Medium/Medium/Hard — and is a guess: a weekly's second
// problem is routinely an Easy and its third routinely a Hard. So it is either the real
// one, read from the REST endpoint while signed in, or EMPTY, which renders as unknown.
// A blank is honest; a wrong one is a lie the board repeats every redraw.
func (q ContestQuestion) Summary() ProblemSummary {
	return ProblemSummary{
		Title:      q.Title,
		Slug:       q.Slug,
		Difficulty: q.Difficulty,
	}
}

// Contest is one contest and its questions.
type Contest struct {
	ContestBrief

	// OriginStartTime differs from StartTime only for a virtual contest, where StartTime
	// is when this user began and OriginStartTime is when the real thing ran.
	OriginStartTime int64 `json:"originStartTime"`
	IsVirtual       bool  `json:"isVirtual"`
	ContainsPremium bool  `json:"containsPremium"`

	Questions []ContestQuestion `json:"-"`

	// Registered is whether THIS account is signed up, and Known says whether the
	// question was answerable at all. Signed out it cannot be, and "not registered" and
	// "did not ask" must not look the same: one is a warning worth interrupting for and
	// the other is nothing.
	Registered        bool `json:"-"`
	RegistrationKnown bool `json:"-"`
}

// Brief reduces a fetched contest to its schedule entry, so the registry can be filled
// from a detail fetch without a second request.
func (c Contest) Brief() ContestBrief { return c.ContestBrief }

// UpcomingContests lists the contests that have not started.
//
// Signed out. Returns the contests whose registration is open, which is a shorter list
// than "every contest with a future start time" — see qUpcomingContests.
func (c *Client) UpcomingContests(ctx context.Context) ([]ContestBrief, error) {
	var out struct {
		Contests []ContestBrief `json:"upcomingContests"`
	}
	if err := c.graphql(ctx, "upcomingContests", qUpcomingContests, nil, &out); err != nil {
		return nil, err
	}
	return out.Contests, nil
}

// ContestDetail reads one contest and its questions.
//
// Two requests, because LeetCode splits them across two root fields and neither carries
// the other. Both go through the shared limiter (D-008).
//
// An EMPTY question list is not an error and is the expected answer before the start —
// the caller decides what to say about it, because "not open yet" and "no such contest"
// are the same response and only the phase tells them apart.
func (c *Client) ContestDetail(ctx context.Context, slug string) (*Contest, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, fmt.Errorf("contest: no slug given")
	}

	var frame struct {
		// A pointer so a contest that does not exist is distinguishable from one that
		// answered with every field at its zero value.
		Contest *Contest `json:"contest"`
	}
	vars := map[string]any{"titleSlug": slug}
	if err := c.graphql(ctx, "contest", qContest, vars, &frame); err != nil {
		return nil, err
	}
	if frame.Contest == nil {
		return nil, fmt.Errorf("contest %s: %w", slug, ErrNotFound)
	}

	var list struct {
		Questions []ContestQuestion `json:"contestQuestionList"`
	}
	qvars := map[string]any{"contestSlug": slug}
	if err := c.graphql(ctx, "contestQuestionList", qContestQuestions, qvars, &list); err != nil {
		return nil, err
	}
	frame.Contest.Questions = list.Questions

	// Enrich from the REST endpoint, which knows two things GraphQL does not: whether
	// this account is registered, and the real difficulties. BEST EFFORT ON PURPOSE —
	// it needs a session, and a signed-out browse must keep working exactly as it did.
	if info, err := c.Info(ctx, slug); err == nil {
		frame.Contest.Registered = info.Registered
		frame.Contest.RegistrationKnown = true

		diff := make(map[string]Difficulty, len(info.Questions))
		for _, q := range info.Questions {
			if d := difficultyOf(q.Difficulty); d != "" {
				diff[q.Slug] = d
			}
		}
		for i := range frame.Contest.Questions {
			if d, ok := diff[frame.Contest.Questions[i].Slug]; ok {
				frame.Contest.Questions[i].Difficulty = d
			}
		}
	}
	return frame.Contest, nil
}

// ContestInfo is what the contest REST endpoint knows that GraphQL does not.
//
// Two things, and both matter:
//
//	Registered  whether THIS account is signed up. An unregistered submission is judged
//	            and scores nothing, and there is no other way to find this out.
//	Difficulty  the real Easy/Medium/Hard, which contestQuestionList does not carry.
//
// Needs a session — the endpoint answers 200 with one and is challenged by Cloudflare
// without one — so it is an enrichment, never the source of truth. Everything the app
// does works signed out through GraphQL; this only adds to it.
type ContestInfo struct {
	Registered bool `json:"registered"`
	Questions  []struct {
		Slug string `json:"title_slug"`
		// Difficulty is 1/2/3 here, not the word. LeetCode uses both spellings across
		// its API and this endpoint is the numeric one.
		Difficulty int `json:"difficulty"`
	} `json:"questions"`
}

// difficultyOf maps the contest REST endpoint's numeric difficulty onto the word the rest
// of the app uses.
//
// An unknown number returns empty rather than guessing, on the same terms as
// normalizeDifficulty: a new level should render as unknown, not as Easy.
func difficultyOf(n int) Difficulty {
	switch n {
	case 1:
		return Easy
	case 2:
		return Medium
	case 3:
		return Hard
	}
	return ""
}

// Info reads a contest's registration state and its questions' real difficulties.
//
// CAUTION, AND THIS COST A FALSE WARNING TO FIND: this endpoint answers **200 with
// `registered: false`** for a request whose session is expired. It does not 401, and a
// stale cookie is not rejected — so `registered == false` on its own means "not
// registered OR not really signed in", and those need very different advice.
//
// `Info` cannot tell the difference on its own, which is why it does not try. Callers must
// confirm the session is live before reporting anything about registration; `Registered`
// on the result is only meaningful once they have. See D-036.
func (c *Client) Info(ctx context.Context, slug string) (*ContestInfo, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, fmt.Errorf("contest info: no slug given")
	}
	var out ContestInfo
	path := "/contest/api/info/" + slug + "/"
	referer := BaseURL + "/contest/" + slug + "/"
	if err := c.getJSONFrom(ctx, path, referer, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ContestProblem reads one contest problem's full statement.
//
// Use this, not Question, for anything a contest lists. While a contest is running its
// problems are not in the problem set and `question(titleSlug:)` returns null for every
// one of them — see qContestQuestion, which records what that cost to learn.
//
// Returns the ordinary Problem type, so everything downstream — the renderer, the local
// driver codegen, the workspace layout — works on a contest problem with no special case.
// Two fields cannot be filled and are left empty rather than faked: Tags, because the
// contest node carries none, and SampleTestCase, which is the first example.
func (c *Client) ContestProblem(ctx context.Context, contest, slug string) (*Problem, error) {
	contest, slug = strings.TrimSpace(contest), strings.TrimSpace(slug)
	if contest == "" || slug == "" {
		return nil, fmt.Errorf("contest problem: need both a contest and a problem")
	}

	var out struct {
		Detail *struct {
			Question *struct {
				Problem
				// The V2 node sends the examples as a list where questionData sends one
				// newline-joined string. Decoded separately and joined below, because
				// everything downstream expects the joined form (D-003's codegen reads
				// it a line per parameter).
				ExampleList []string `json:"exampleTestcaseList"`
			} `json:"question"`
		} `json:"contestQuestion"`
	}
	vars := map[string]any{"contestSlug": contest, "questionSlug": slug}
	if err := c.graphql(ctx, "contestQuestion", qContestQuestion, vars, &out); err != nil {
		return nil, err
	}
	if out.Detail == nil || out.Detail.Question == nil {
		return nil, fmt.Errorf("contest problem %s in %s: %w", slug, contest, ErrNotFound)
	}

	p := out.Detail.Question.Problem
	p.ExampleTestcases = strings.Join(out.Detail.Question.ExampleList, "\n")
	if len(out.Detail.Question.ExampleList) > 0 {
		p.SampleTestCase = out.Detail.Question.ExampleList[0]
	}
	return &p, nil
}

// SubmitContest sends a solution to a contest's judge and returns the id to poll.
//
// THIS IS A DIFFERENT ENDPOINT FROM Submit, AND THE DIFFERENCE IS THE WHOLE POINT. A
// submission to /problems/{slug}/submit/ during a live contest is judged, is recorded
// against the problem, and SCORES NOTHING — the contest never sees it. Only
// /contest/api/{contest}/problems/{slug}/submit/ counts, and the two return the same
// shape, so a wrong call looks exactly like a right one until the standings do not move.
//
// The Referer must name the contest problem page for the same reason Submit's must name
// the problem page: LeetCode rejects a submission that does not plausibly come from one.
//
// The reply is polled with Check, which is shared with Submit — the judge writes contest
// and ordinary submissions to the same /submissions/detail/{id}/check/ endpoint.
func (c *Client) SubmitContest(ctx context.Context, contest string, s Submission) (string, error) {
	contest = strings.TrimSpace(contest)
	if contest == "" {
		return "", fmt.Errorf("submit %s: no contest given", s.Slug)
	}

	body := map[string]any{
		"lang":        s.Lang,
		"question_id": s.QuestionID,
		"typed_code":  s.Code,
	}
	var out struct {
		SubmissionID json.Number `json:"submission_id"`
	}
	path := "/contest/api/" + contest + "/problems/" + s.Slug + "/submit/"
	referer := BaseURL + "/contest/" + contest + "/problems/" + s.Slug + "/"
	if err := c.postJSONTo(ctx, path, referer, body, &out); err != nil {
		// A 404 here means the contest judge did not recognise the pair, and the usual
		// cause is that this account is not REGISTERED for the contest — LeetCode does
		// not expose a registration endpoint, so there is nothing to check beforehand.
		// Say so and name the page that always works, because a contest is the one place
		// where a confusing error costs points that cannot be won back.
		if errors.Is(err, ErrNotFound) {
			return "", fmt.Errorf("the contest judge does not know %s in %s — check you are "+
				"registered, and submit at %s if this persists: %w",
				s.Slug, contest, referer, err)
		}
		return "", err
	}
	if out.SubmissionID.String() == "" {
		return "", fmt.Errorf("submit %s to %s: judge returned no submission id", s.Slug, contest)
	}
	return out.SubmissionID.String(), nil
}
