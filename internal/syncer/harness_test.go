package syncer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/store"
)

func testStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.OpenPath(filepath.Join(t.TempDir(), "sync.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// fakeLeetCode serves problemsetQuestionList pages from a synthetic corpus, so the sync
// loop's paging, checkpointing, and resume behaviour can be tested without the network.
type fakeLeetCode struct {
	total    int
	calls    int
	failAt   int  // return 500 on this call index (1-based); 0 disables
	rateAt   int  // return 429 on this call index; 0 disables
	rateOnce bool // only rate-limit the first time
	rated    bool
}

func (f *fakeLeetCode) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.calls++

		var body struct {
			Variables struct {
				Skip  int `json:"skip"`
				Limit int `json:"limit"`
			} `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}

		if f.failAt == f.calls {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"errors":[{"message":"boom"}]}`)
			return
		}
		if f.rateAt == f.calls && !(f.rateOnce && f.rated) {
			f.rated = true
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		skip, limit := body.Variables.Skip, body.Variables.Limit
		var qs []string
		for i := skip; i < skip+limit && i < f.total; i++ {
			qs = append(qs, fmt.Sprintf(`{
				"acRate": %d.5, "difficulty": "Easy", "frontendQuestionId": "%d",
				"paidOnly": false, "status": null, "title": "Problem %d",
				"titleSlug": "problem-%d", "topicTags": [{"name":"Array","slug":"array"}],
				"hasSolution": true, "hasVideoSolution": false
			}`, i%90, i+1, i+1, i+1))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"data":{"problemsetQuestionList":{"total":%d,"questions":[%s]}}}`,
			f.total, strings.Join(qs, ","))
	}
}

func newFake(t *testing.T, f *fakeLeetCode) (*leetcode.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(f.handler(t))
	t.Cleanup(srv.Close)

	// Point the client at the fake by swapping its transport's destination.
	cl := leetcode.New(
		leetcode.WithRateLimit(1000), // no artificial delay in tests
		leetcode.WithHTTPClient(&http.Client{
			Transport: redirectTo(srv.URL),
			Timeout:   10 * time.Second,
		}),
	)
	return cl, srv
}

// redirectTo rewrites every request to the test server.
type redirectTo string

func (base redirectTo) RoundTrip(r *http.Request) (*http.Response, error) {
	u := *r.URL
	target := strings.TrimPrefix(string(base), "http://")
	u.Scheme, u.Host = "http", target
	r2 := r.Clone(r.Context())
	r2.URL = &u
	r2.Host = target
	return http.DefaultTransport.RoundTrip(r2)
}

func drainProgress(ch <-chan Progress) []Progress {
	var out []Progress
	for p := range ch {
		out = append(out, p)
	}
	return out
}
