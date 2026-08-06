package leetcode

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Difficulty is LeetCode's rating. Stored as its wire string ("Easy"/"Medium"/"Hard").
type Difficulty string

const (
	Easy   Difficulty = "Easy"
	Medium Difficulty = "Medium"
	Hard   Difficulty = "Hard"
)

// Status is the signed-in user's progress on a problem. Empty when signed out.
type Status string

const (
	StatusNone      Status = ""
	StatusAccepted  Status = "ac"
	StatusAttempted Status = "notac"
)

// Tag is a topic tag.
type Tag struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// ProblemSummary is a row from problemsetQuestionList — enough to populate the board
// without fetching each problem's statement.
type ProblemSummary struct {
	// FrontendID is the number shown on the site. It is a STRING on the wire because a
	// handful of problems use non-numeric IDs (LCP/LCS contest series), so parse it
	// with NumericID rather than assuming it is an integer.
	FrontendID       string     `json:"frontendQuestionId"`
	Title            string     `json:"title"`
	Slug             string     `json:"titleSlug"`
	Difficulty       Difficulty `json:"difficulty"`
	AcRate           float64    `json:"acRate"`
	PaidOnly         bool       `json:"paidOnly"`
	Status           Status     `json:"status"`
	Tags             []Tag      `json:"topicTags"`
	HasSolution      bool       `json:"hasSolution"`
	HasVideoSolution bool       `json:"hasVideoSolution"`
}

// NumericID returns the frontend ID as an integer, or 0 for non-numeric IDs.
func (p ProblemSummary) NumericID() int {
	n, err := strconv.Atoi(strings.TrimSpace(p.FrontendID))
	if err != nil {
		return 0
	}
	return n
}

// CodeSnippet is a starter template for one language.
type CodeSnippet struct {
	Lang     string `json:"lang"`     // display name, e.g. "Python3"
	LangSlug string `json:"langSlug"` // API slug, e.g. "python3"
	Code     string `json:"code"`
}

// Problem is the full detail from questionData.
type Problem struct {
	// QuestionID is LeetCode's internal ID. This is what the submit endpoint wants —
	// NOT the frontend ID. Mixing them up submits against the wrong problem.
	QuestionID string        `json:"questionId"`
	FrontendID string        `json:"questionFrontendId"`
	Title      string        `json:"title"`
	Slug       string        `json:"titleSlug"`
	Content    string        `json:"content"` // HTML; rendered via internal/render (D-007)
	Difficulty Difficulty    `json:"difficulty"`
	PaidOnly   bool          `json:"isPaidOnly"`
	Likes      int           `json:"likes"`
	Dislikes   int           `json:"dislikes"`
	Tags       []Tag         `json:"topicTags"`
	Snippets   []CodeSnippet `json:"codeSnippets"`
	Hints      []string      `json:"hints"`

	// SampleTestCase is the single example case; ExampleTestcases is all of them,
	// newline-separated with one line per parameter.
	SampleTestCase   string `json:"sampleTestCase"`
	ExampleTestcases string `json:"exampleTestcases"`

	// MetaData is a JSON string describing the function signature: name, param names
	// and types, and return type. This is what drives local driver codegen (D-003).
	// It does NOT encode in-place mutation, order-insensitivity, or float tolerance.
	MetaData string `json:"metaData"`

	EnableRunCode bool `json:"enableRunCode"`
	HasSolution   bool `json:"hasSolution"`
}

// NumericID returns the frontend ID as an integer, or 0 for non-numeric IDs.
func (p Problem) NumericID() int {
	n, err := strconv.Atoi(strings.TrimSpace(p.FrontendID))
	if err != nil {
		return 0
	}
	return n
}

// Snippet returns the starter code for a language slug.
func (p Problem) Snippet(langSlug string) (CodeSnippet, bool) {
	for _, s := range p.Snippets {
		if s.LangSlug == langSlug {
			return s, true
		}
	}
	return CodeSnippet{}, false
}

// MetaFunc is the parsed MetaData signature.
type MetaFunc struct {
	Name   string `json:"name"`
	Params []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"params"`
	Return struct {
		Type string `json:"type"`
		Size int    `json:"size"`
	} `json:"return"`
	// Manual marks problems whose driver LeetCode generates by hand. These are the
	// design problems (constructor + op/arg sequence) and are the ones most likely to
	// need a comparator override.
	Manual bool `json:"manual"`

	// Classname is set for design problems, e.g. "LRUCache".
	Classname string `json:"classname"`
	// Constructor and Methods are present for design problems.
	Constructor json.RawMessage `json:"constructor"`
	Methods     json.RawMessage `json:"methods"`
}

// ParseMeta decodes the MetaData JSON string.
func (p Problem) ParseMeta() (MetaFunc, error) {
	var m MetaFunc
	err := json.Unmarshal([]byte(p.MetaData), &m)
	return m, err
}

// IsDesign reports whether this is a design problem — a class with a constructor and a
// sequence of operations, rather than a single function. These need a different driver
// shape and are the most common source of local/judge disagreement.
func (m MetaFunc) IsDesign() bool {
	return m.Classname != "" || len(m.Methods) > 0
}

// UserStatus is the signed-in user's account state.
type UserStatus struct {
	UserID     json.Number `json:"userId"`
	IsSignedIn bool        `json:"isSignedIn"`
	IsPremium  bool        `json:"isPremium"`
	Username   string      `json:"username"`
}

// problemListResponse is the shape of a problemsetQuestionList reply.
type problemListResponse struct {
	ProblemsetQuestionList struct {
		Total     int              `json:"total"`
		Questions []ProblemSummary `json:"questions"`
	} `json:"problemsetQuestionList"`
}

type questionResponse struct {
	Question *Problem `json:"question"`
}

type userStatusResponse struct {
	UserStatus UserStatus `json:"userStatus"`
}
