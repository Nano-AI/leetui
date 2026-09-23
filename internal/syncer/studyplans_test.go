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
	"github.com/Nano-AI/leetui/internal/store"
)

// fakePlans serves the two study plan operations.
//
// known is the set of plans that exist; anything else answers null, which is LeetCode's
// real behaviour for a retired slug and the case the registry has to survive. gated marks
// plans that return their metadata with an EMPTY group list — the free-account response,
// and the one that must not be mistaken for a plan with no problems.
type fakePlans struct {
	known  map[string]int // slug -> question count
	gated  map[string]bool
	byTag  map[string][]string
	detail int // how many studyPlanV2Detail calls were served
}

func (f *fakePlans) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Operation string `json:"operationName"`
			Variables struct {
				PlanSlug string `json:"planSlug"`
				TagSlug  string `json:"tagSlug"`
			} `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")

		switch body.Operation {
		case "studyPlanV2Detail":
			f.detail++
			slug := body.Variables.PlanSlug
			n, ok := f.known[slug]
			if !ok {
				fmt.Fprint(w, `{"data":{"studyPlanV2Detail":null}}`)
				return
			}
			if f.gated[slug] {
				fmt.Fprintf(w, `{"data":{"studyPlanV2Detail":{"slug":%q,"name":%q,
					"highlight":"gated","questionNum":%d,"premiumOnly":true,
					"planSubGroups":[]}}}`, slug, slug, n)
				return
			}
			// Two chapters, so the flattened order is testable. Difficulty and status use
			// the plan endpoint's OWN spelling, which is what the client has to normalise.
			var first, second []string
			for i := 0; i < n; i++ {
				q := fmt.Sprintf(`{"titleSlug":"%s-%d","title":"Problem %d",
					"questionFrontendId":"%d","difficulty":"MEDIUM","status":"TO_DO",
					"paidOnly":false,"topicTags":[]}`, slug, i+1, i+1, n-i)
				if i < n/2 {
					first = append(first, q)
				} else {
					second = append(second, q)
				}
			}
			fmt.Fprintf(w, `{"data":{"studyPlanV2Detail":{"slug":%q,"name":%q,
				"highlight":"pitch","questionNum":%d,"premiumOnly":false,
				"planSubGroups":[
					{"slug":"g1","name":"Array / String","questionNum":%d,"questions":[%s]},
					{"slug":"g2","name":"Two Pointers","questionNum":%d,"questions":[%s]}]}}}`,
				slug, slug, n,
				len(first), strings.Join(first, ","),
				len(second), strings.Join(second, ","))

		case "studyPlansV2ByTag":
			var out []string
			for _, slug := range f.byTag[body.Variables.TagSlug] {
				out = append(out, fmt.Sprintf(
					`{"slug":%q,"name":%q,"highlight":"from a tag","questionNum":%d,"premiumOnly":false}`,
					slug, slug, f.known[slug]))
			}
			fmt.Fprintf(w, `{"data":{"studyPlansV2ByTag":{"total":%d,"hasMore":false,"studyPlans":[%s]}}}`,
				len(out), strings.Join(out, ","))

		default:
			t.Errorf("unexpected operation %q", body.Operation)
			fmt.Fprint(w, `{"data":{}}`)
		}
	}
}

func newPlanFake(t *testing.T, f *fakePlans) *leetcode.Client {
	t.Helper()
	srv := httptest.NewServer(f.handler(t))
	t.Cleanup(srv.Close)
	return leetcode.New(
		leetcode.WithRateLimit(1000),
		leetcode.WithHTTPClient(&http.Client{Transport: redirectTo(srv.URL), Timeout: 10 * time.Second}),
	)
}

func TestStudyPlanSyncsWholeAndInOrder(t *testing.T) {
	st := testStore(t)
	fake := &fakePlans{known: map[string]int{"leetcode-75": 6}}
	sy := New(newPlanFake(t, fake), st, 100)

	ch := make(chan Progress, 16)
	go func() { _ = sy.StudyPlan(context.Background(), "leetcode-75", ch) }()
	progress := drainProgress(ch)

	last := progress[len(progress)-1]
	if !last.Finished || last.Err != nil {
		t.Fatalf("plan finished as %+v", last)
	}
	if last.Done != 6 {
		t.Errorf("finished at %d, want 6", last.Done)
	}
	// The contrast with a company pack: one request, no paging loop.
	if fake.detail != 1 {
		t.Errorf("made %d detail requests for a 6-problem plan, want 1", fake.detail)
	}

	n, err := st.PlanCount(context.Background(), "leetcode-75")
	if err != nil || n != 6 {
		t.Fatalf("stored %d problems (err %v), want 6", n, err)
	}

	// The plan endpoint's "MEDIUM" must have been normalised on the way in, or every row
	// on the board renders as Easy.
	rows, err := st.Query(context.Background(),
		store.Filter{Plan: "leetcode-75", Sort: "plan"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(rows) != 6 {
		t.Fatalf("board shows %d rows, want 6", len(rows))
	}
	if rows[0].Difficulty != "Medium" {
		t.Errorf("difficulty stored as %q, want Medium — the plan endpoint says MEDIUM", rows[0].Difficulty)
	}
	// Plan order, not ID order: the fake numbers its problems descending.
	if rows[0].Slug != "leetcode-75-1" {
		t.Errorf("top row is %q, want leetcode-75-1 — plan order beats ID order", rows[0].Slug)
	}
}

// TestStudyPlanGateIsNotAnEmptyPlan is the silent failure a premium plan would otherwise
// cause: a well-formed response with no chapters is an account without a subscription,
// not a plan with nothing in it.
func TestStudyPlanGateIsNotAnEmptyPlan(t *testing.T) {
	st := testStore(t)
	sy := New(newPlanFake(t, &fakePlans{
		known: map[string]int{"premium-algo-100": 100},
		gated: map[string]bool{"premium-algo-100": true},
	}), st, 100)

	ch := make(chan Progress, 16)
	go func() { _ = sy.StudyPlan(context.Background(), "premium-algo-100", ch) }()
	progress := drainProgress(ch)

	last := progress[len(progress)-1]
	if !errors.Is(last.Err, leetcode.ErrPremiumRequired) {
		t.Fatalf("gated plan finished as %+v, want ErrPremiumRequired", last)
	}
	if n, _ := st.PlanCount(context.Background(), "premium-algo-100"); n != 0 {
		t.Errorf("gated plan wrote %d rows", n)
	}
}

func TestStudyPlanRejectsUnknownSlug(t *testing.T) {
	st := testStore(t)
	sy := New(newPlanFake(t, &fakePlans{known: map[string]int{"leetcode-75": 3}}), st, 100)

	ch := make(chan Progress, 8)
	go func() { _ = sy.StudyPlan(context.Background(), "not-a-plan", ch) }()
	progress := drainProgress(ch)

	if last := progress[len(progress)-1]; !errors.Is(last.Err, leetcode.ErrNotFound) {
		t.Fatalf("unknown plan finished as %+v, want ErrNotFound", last)
	}
}

// TestPlanRegistryUnionsSeedsAndTags is the reason the registry is built the way it is
// (D-031): neither source is the catalogue on its own.
func TestPlanRegistryUnionsSeedsAndTags(t *testing.T) {
	st := testStore(t)

	// One seed slug that no tag knows about, and one tag-only plan that is not a seed.
	known := map[string]int{"top-100-liked": 100, "brand-new-plan": 20}
	for _, s := range leetcode.SeedPlans() {
		known[s] = 10
	}
	fake := &fakePlans{
		known: known,
		byTag: map[string][]string{"interview": {"brand-new-plan", "top-100-liked"}},
	}
	sy := New(newPlanFake(t, fake), st, 100)

	ch := make(chan Progress, 64)
	go func() { _ = sy.PlanRegistry(context.Background(), ch) }()
	drainProgress(ch)

	plans, err := st.Plans(context.Background(), "")
	if err != nil {
		t.Fatalf("Plans: %v", err)
	}
	got := map[string]bool{}
	for _, p := range plans {
		got[p.Slug] = true
	}
	// Every seed survived...
	for _, s := range leetcode.SeedPlans() {
		if !got[s] {
			t.Errorf("seed plan %q is missing from the registry", s)
		}
	}
	// ...and the tag sweep contributed one no seed list could know about.
	if !got["brand-new-plan"] {
		t.Error("a tag-only plan did not reach the registry; discovery is doing nothing")
	}
}

// TestPlanRegistrySurvivesARetiredSlug: a seed slug LeetCode has dropped must cost that
// one plan, not the whole registry. "sql-50" became "top-sql-50" once already.
func TestPlanRegistrySurvivesARetiredSlug(t *testing.T) {
	st := testStore(t)

	known := map[string]int{}
	for _, s := range leetcode.SeedPlans() {
		known[s] = 10
	}
	// LeetCode retires one of them.
	delete(known, "binary-search")

	sy := New(newPlanFake(t, &fakePlans{known: known}), st, 100)
	ch := make(chan Progress, 64)
	go func() { _ = sy.PlanRegistry(context.Background(), ch) }()
	progress := drainProgress(ch)

	if last := progress[len(progress)-1]; last.Err != nil {
		t.Fatalf("one dead slug failed the whole registry: %v", last.Err)
	}
	plans, err := st.Plans(context.Background(), "")
	if err != nil {
		t.Fatalf("Plans: %v", err)
	}
	if len(plans) != len(leetcode.SeedPlans())-1 {
		t.Errorf("registry has %d plans, want %d — the other seeds should be unaffected",
			len(plans), len(leetcode.SeedPlans())-1)
	}
}

// TestPlanRegistryKeepsGatedPlans: a premium plan reports its name and size even when
// gated, so it belongs in the picker with a note rather than being hidden from someone
// who may well be paying for it.
func TestPlanRegistryKeepsGatedPlans(t *testing.T) {
	st := testStore(t)

	known := map[string]int{}
	for _, s := range leetcode.SeedPlans() {
		known[s] = 10
	}
	sy := New(newPlanFake(t, &fakePlans{
		known: known,
		gated: map[string]bool{"premium-algo-100": true},
	}), st, 100)

	ch := make(chan Progress, 64)
	go func() { _ = sy.PlanRegistry(context.Background(), ch) }()
	drainProgress(ch)

	plans, err := st.Plans(context.Background(), "premium-algo-100")
	if err != nil {
		t.Fatalf("Plans: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("the gated plan is missing from the registry (%d rows)", len(plans))
	}
	if !plans[0].PremiumOnly {
		t.Error("the gated plan is not flagged premium, so the picker cannot say so")
	}
}
