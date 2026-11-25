package leetcode

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/auth"
)

func TestSubmitStudyPlanContext(t *testing.T) {
	for _, tc := range []struct {
		name, plan, refererQuery string
	}{
		{name: "regular"},
		{name: "study plan", plan: "leetcode-75", refererQuery: "?envId=leetcode-75&envType=study-plan-v2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const problem = BaseURL + "/problems/h-index/"
			client := New(
				WithCredentials(auth.Credentials{Session: "test-session", CSRF: "test-csrf"}),
				WithHTTPClient(&http.Client{Transport: responseTransport(func(r *http.Request) (*http.Response, error) {
					if r.Method != http.MethodPost || r.URL.String() != problem+"submit/" {
						t.Errorf("request = %s %s", r.Method, r.URL)
					}
					if got := r.Referer(); got != problem+tc.refererQuery {
						t.Errorf("Referer = %q, want %q", got, problem+tc.refererQuery)
					}
					var body map[string]string
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					want := map[string]string{"lang": "cpp", "question_id": "274", "typed_code": "class Solution {};"}
					if tc.plan != "" {
						want["study_plan_slug"] = tc.plan
					}
					if !reflect.DeepEqual(body, want) {
						t.Errorf("body = %v, want %v", body, want)
					}
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"submission_id":123}`))}, nil
				})}),
			)
			id, err := client.Submit(t.Context(), Submission{
				Slug: "h-index", QuestionID: "274", Lang: "cpp", Code: "class Solution {};",
				StudyPlanSlug: tc.plan,
			})
			if err != nil || id != "123" {
				t.Fatalf("Submit = %q, %v", id, err)
			}
		})
	}
}
