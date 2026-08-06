package leetcode

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Submitting and polling the judge.
//
// These are REST, not GraphQL, and they are the endpoints the website's own editor
// uses. Two things matter:
//
//   - The body wants `question_id`, which is LeetCode's INTERNAL id, not the number on
//     the problem page. Sending the frontend id submits against a different problem.
//   - `Referer` must name the problem page. Requests without a plausible one are
//     rejected outright.

// VerdictCode is the judge's numeric verdict.
//
// Named Verdict rather than Status to keep it distinct from Status in models.go, which
// is the user's own progress on a problem ("ac"/"notac"). They are different axes and
// conflating them submits the wrong thing.
type VerdictCode int

// Judge verdicts, as LeetCode numbers them.
const (
	VerdictAccepted            VerdictCode = 10
	VerdictWrongAnswer         VerdictCode = 11
	VerdictMemoryLimitExceeded VerdictCode = 12
	VerdictOutputLimitExceeded VerdictCode = 13
	VerdictTimeLimitExceeded   VerdictCode = 14
	VerdictRuntimeError        VerdictCode = 15
	VerdictInternalError       VerdictCode = 16
	VerdictCompileError        VerdictCode = 20
)

// Judgement is the result of a submission or a remote run.
type Judgement struct {
	// State is "PENDING", "STARTED", or "SUCCESS". SUCCESS means the judge finished,
	// not that the answer was right — StatusCode carries that.
	State      string      `json:"state"`
	StatusCode VerdictCode `json:"status_code"`
	StatusMsg  string      `json:"status_msg"`

	Runtime           string  `json:"status_runtime"`
	Memory            string  `json:"status_memory"`
	RuntimePercentile float64 `json:"runtime_percentile"`
	MemoryPercentile  float64 `json:"memory_percentile"`

	TotalCorrect   int `json:"total_correct"`
	TotalTestcases int `json:"total_testcases"`

	// Set when something went wrong, in rough order of how much the user needs it.
	CompileError     string   `json:"compile_error"`
	FullCompileError string   `json:"full_compile_error"`
	RuntimeError     string   `json:"runtime_error"`
	LastTestcase     string   `json:"last_testcase"`
	ExpectedOutput   string   `json:"expected_output"`
	CodeOutput       []string `json:"code_output"`

	SubmissionID json.Number `json:"submission_id"`
}

// Done reports whether the judge has finished.
func (j Judgement) Done() bool { return j.State == "SUCCESS" }

// Accepted reports whether the solution passed.
func (j Judgement) Accepted() bool { return j.StatusCode == VerdictAccepted }

// Submission is what gets sent to the judge.
type Submission struct {
	Slug string
	// QuestionID is LeetCode's internal id — Problem.QuestionID, never FrontendID.
	QuestionID string
	Lang       string
	Code       string
}

// Submit sends a solution to the judge and returns the id to poll.
func (c *Client) Submit(ctx context.Context, s Submission) (string, error) {
	body := map[string]any{
		"lang":        s.Lang,
		"question_id": s.QuestionID,
		"typed_code":  s.Code,
	}
	var out struct {
		SubmissionID json.Number `json:"submission_id"`
	}
	if err := c.postJSON(ctx, "/problems/"+s.Slug+"/submit/", s.Slug, body, &out); err != nil {
		return "", err
	}
	if out.SubmissionID.String() == "" {
		return "", fmt.Errorf("submit %s: judge returned no submission id", s.Slug)
	}
	return out.SubmissionID.String(), nil
}

// RunRemote sends a solution plus input to the judge without recording a submission.
// This is the fallback for languages with no local driver (D-004).
func (c *Client) RunRemote(ctx context.Context, s Submission, input string) (string, error) {
	body := map[string]any{
		"lang":        s.Lang,
		"question_id": s.QuestionID,
		"typed_code":  s.Code,
		"data_input":  input,
	}
	var out struct {
		InterpretID string `json:"interpret_id"`
	}
	if err := c.postJSON(ctx, "/problems/"+s.Slug+"/interpret_solution/", s.Slug, body, &out); err != nil {
		return "", err
	}
	if out.InterpretID == "" {
		return "", fmt.Errorf("run %s: judge returned no interpret id", s.Slug)
	}
	return out.InterpretID, nil
}

// Check polls one judgement.
func (c *Client) Check(ctx context.Context, id string) (Judgement, error) {
	var j Judgement
	err := c.getJSON(ctx, "/submissions/detail/"+id+"/check/", &j)
	return j, err
}

// PollInterval is how often the judge is asked. The website polls about this often; a
// tighter loop just burns the shared rate limiter.
const PollInterval = 900 * time.Millisecond

// Poll waits for a judgement, reporting each intermediate state through onUpdate.
//
// Every poll passes through the same rate limiter as everything else (D-008), so a long
// judge queue cannot starve a sync running alongside it.
func (c *Client) Poll(ctx context.Context, id string, onUpdate func(Judgement)) (Judgement, error) {
	for {
		j, err := c.Check(ctx, id)
		if err != nil {
			return j, err
		}
		if onUpdate != nil {
			onUpdate(j)
		}
		if j.Done() {
			return j, nil
		}

		select {
		case <-time.After(PollInterval):
		case <-ctx.Done():
			return j, ctx.Err()
		}
	}
}
