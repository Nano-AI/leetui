package syncer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

// fakePremium serves the three premium operations a company pack needs. gated switches
// favoriteQuestionList to LeetCode's real free-account behaviour: an empty question list
// rather than an error, which is exactly the case the client has to detect.
type fakePremium struct {
	packSize int
	gated    bool
	pages    int // how many favoriteQuestionList calls were served
}

func (f *fakePremium) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Operation string `json:"operationName"`
			Variables struct {
				FavoriteSlug string `json:"favoriteSlug"`
				Skip         int    `json:"skip"`
				Limit        int    `json:"limit"`
			} `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")

		switch body.Operation {
		case "companyTags":
			fmt.Fprint(w, `{"data":{"companyTags":[
				{"name":"Google","slug":"google","questionCount":2335},
				{"name":"Meta","slug":"facebook","questionCount":1402}]}}`)

		case "favoriteDetailV2":
			if !strings.HasPrefix(body.Variables.FavoriteSlug, "google-") {
				// LeetCode answers an unknown pack with null, not an error.
				fmt.Fprint(w, `{"data":{"favoriteDetailV2":null}}`)
				return
			}
			fmt.Fprintf(w, `{"data":{"favoriteDetailV2":{"name":"Google","slug":%q,"questionNumber":%d}}}`,
				body.Variables.FavoriteSlug, f.packSize)

		case "favoriteQuestionList":
			f.pages++
			if f.gated {
				fmt.Fprintf(w, `{"data":{"favoriteQuestionList":{"totalLength":%d,"hasMore":false,"questions":[]}}}`,
					f.packSize)
				return
			}
			skip, limit := body.Variables.Skip, body.Variables.Limit
			var qs []string
			for i := skip; i < skip+limit && i < f.packSize; i++ {
				qs = append(qs, fmt.Sprintf(`{
					"titleSlug":"problem-%d","title":"Problem %d","questionFrontendId":"%d",
					"difficulty":"Medium","status":null,"paidOnly":false,
					"frequency":%d.0,"acRate":50.0,
					"topicTags":[{"name":"Array","slug":"array"}]}`, i+1, i+1, i+1, f.packSize-i))
			}
			fmt.Fprintf(w, `{"data":{"favoriteQuestionList":{"totalLength":%d,"hasMore":%t,"questions":[%s]}}}`,
				f.packSize, skip+limit < f.packSize, strings.Join(qs, ","))

		default:
			t.Errorf("unexpected operation %q", body.Operation)
			fmt.Fprint(w, `{"data":{}}`)
		}
	}
}

func newPremiumFake(t *testing.T, f *fakePremium) *leetcode.Client {
	t.Helper()
	srv := httptest.NewServer(f.handler(t))
	t.Cleanup(srv.Close)
	return leetcode.New(
		leetcode.WithRateLimit(1000),
		leetcode.WithHTTPClient(&http.Client{Transport: redirectTo(srv.URL), Timeout: 10 * time.Second}),
	)
}

func TestCompanyRegistrySyncs(t *testing.T) {
	st := testStore(t)
	cl := newPremiumFake(t, &fakePremium{packSize: 0})
	sy := New(cl, st, 100)

	ch := make(chan Progress, 8)
	go func() { _ = sy.CompanyRegistry(context.Background(), ch) }()
	drainProgress(ch)

	got, err := st.Companies(context.Background(), "")
	if err != nil {
		t.Fatalf("Companies: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("stored %d companies, want 2", len(got))
	}
	if got[0].Slug != "google" || got[0].QuestionCount != 2335 {
		t.Errorf("first company is %+v, want google with 2335", got[0])
	}
}

func TestPackPagesAndStores(t *testing.T) {
	st := testStore(t)
	fake := &fakePremium{packSize: 250}
	sy := New(newPremiumFake(t, fake), st, 100)

	ch := make(chan Progress, 16)
	go func() { _ = sy.Pack(context.Background(), "google", leetcode.ThreeMonths, ch) }()
	progress := drainProgress(ch)

	last := progress[len(progress)-1]
	if !last.Finished || last.Err != nil {
		t.Fatalf("pack finished as %+v", last)
	}
	if last.Done != 250 || last.Total != 250 {
		t.Errorf("finished at %d/%d, want 250/250", last.Done, last.Total)
	}
	if fake.pages != 3 {
		t.Errorf("made %d page requests for 250 problems at 100 per page, want 3", fake.pages)
	}

	n, err := st.PackCount(context.Background(), "google", "three-months")
	if err != nil || n != 250 {
		t.Errorf("stored %d problems (err %v), want 250", n, err)
	}
}

// TestPackGateIsNotAnEmptyList is the failure mode that would be silent otherwise: a free
// account gets a well-formed response with zero questions, and reporting that as "this
// company asks nothing" would be a lie.
func TestPackGateIsNotAnEmptyList(t *testing.T) {
	st := testStore(t)
	sy := New(newPremiumFake(t, &fakePremium{packSize: 199, gated: true}), st, 100)

	ch := make(chan Progress, 16)
	go func() { _ = sy.Pack(context.Background(), "google", leetcode.Thirty, ch) }()
	progress := drainProgress(ch)

	last := progress[len(progress)-1]
	if !errors.Is(last.Err, leetcode.ErrPremiumRequired) {
		t.Fatalf("gated pack finished as %+v, want ErrPremiumRequired", last)
	}
	// Nothing may be written: a half-empty pack that looked complete is worse than none.
	if n, _ := st.PackCount(context.Background(), "google", "thirty-days"); n != 0 {
		t.Errorf("gated pack wrote %d rows", n)
	}
}

func TestPackRejectsUnknownCompany(t *testing.T) {
	st := testStore(t)
	sy := New(newPremiumFake(t, &fakePremium{packSize: 10}), st, 100)

	ch := make(chan Progress, 8)
	go func() { _ = sy.Pack(context.Background(), "not-a-company", leetcode.AllTime, ch) }()
	progress := drainProgress(ch)

	last := progress[len(progress)-1]
	if !errors.Is(last.Err, leetcode.ErrNotFound) {
		t.Fatalf("unknown company finished as %+v, want ErrNotFound", last)
	}
}

func TestPackRejectsUnknownTimeframe(t *testing.T) {
	st := testStore(t)
	sy := New(newPremiumFake(t, &fakePremium{packSize: 10}), st, 100)

	ch := make(chan Progress, 8)
	// A timeframe LeetCode does not offer must fail before any request goes out.
	go func() { _ = sy.Pack(context.Background(), "google", leetcode.Timeframe("one-year"), ch) }()
	progress := drainProgress(ch)

	if last := progress[len(progress)-1]; last.Err == nil {
		t.Fatalf("unknown timeframe finished as %+v, want an error", last)
	}
}
