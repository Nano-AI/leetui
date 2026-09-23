package syncer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/store"
)

// All traffic stays local. The fields cover the successful payloads needed to
// reach each job's storage boundary, including the contest REST enrichment.
func storageErrorClient(t *testing.T) *leetcode.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":{
			"problemsetQuestionList":{"total":2,"questions":[{"titleSlug":"example","title":"Example"}]},
			"companyTags":[{"slug":"company","name":"Company"}],
			"favoriteDetailV2":{"questionNumber":0},
			"studyPlansV2ByTag":{"studyPlans":[]},
			"studyPlanV2Detail":{"slug":"course","name":"Course","questionNum":1,
				"planSubGroups":[{"questions":[{"titleSlug":"example","title":"Example"}]}]},
			"upcomingContests":[{"titleSlug":"weekly","title":"Weekly"}],
			"contest":{"titleSlug":"weekly","title":"Weekly"},
			"contestQuestionList":[{"titleSlug":"example","title":"Example"}],
			"userStatus":{"username":"new-user","isSignedIn":true,"isPremium":true}
		}}`)
	}))
	t.Cleanup(srv.Close)
	return leetcode.New(leetcode.WithRateLimit(10000), leetcode.WithHTTPClient(&http.Client{Transport: redirectTo(srv.URL)}))
}

func TestSyncStorageFailuresAreNotSuccess(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name, table, condition string
		run                    func(*Syncer, chan<- Progress) error
		wantCursor             string
	}{
		{"problem total", "sync_state", "NEW.key = 'problems_total'", func(s *Syncer, ch chan<- Progress) error { return s.Problems(ctx, ch, true) }, "1"},
		{"problem checkpoint", "sync_state", "NEW.key = 'problems_cursor' AND NEW.value = '2'", func(s *Syncer, ch chan<- Progress) error { return s.Problems(ctx, ch, true) }, "1"},
		{"problem completion cursor", "sync_state", "NEW.key = 'problems_cursor' AND NEW.value = '0'", func(s *Syncer, ch chan<- Progress) error { return s.Problems(ctx, ch, true) }, "2"},
		{"problem completion time", "sync_state", "NEW.key = 'problems_synced_at'", func(s *Syncer, ch chan<- Progress) error { return s.Problems(ctx, ch, true) }, "2"},
		{"company registry", "sync_state", "NEW.key = 'companies_synced_at'", func(s *Syncer, ch chan<- Progress) error { return s.CompanyRegistry(ctx, ch) }, ""},
		{"pack", "sync_state", "NEW.key = 'pack:company:all'", func(s *Syncer, ch chan<- Progress) error {
			return s.Pack(ctx, "company", leetcode.Timeframe("all"), ch)
		}, ""},
		{"plan registry", "sync_state", "NEW.key = 'plans_synced_at'", func(s *Syncer, ch chan<- Progress) error { return s.PlanRegistry(ctx, ch) }, ""},
		{"plan", "sync_state", "NEW.key = 'plan:course'", func(s *Syncer, ch chan<- Progress) error { return s.StudyPlan(ctx, "course", ch) }, ""},
		{"plan metadata", "study_plans", "1", func(s *Syncer, ch chan<- Progress) error { return s.StudyPlan(ctx, "course", ch) }, ""},
		{"contest schedule", "sync_state", "NEW.key = 'contests_synced_at'", func(s *Syncer, ch chan<- Progress) error { return s.ContestSchedule(ctx, ch) }, ""},
		{"contest", "sync_state", "NEW.key = 'contest:weekly'", func(s *Syncer, ch chan<- Progress) error { return s.Contest(ctx, "weekly", ch) }, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := testStore(t)
			if err := st.SetState(ctx, store.KeyProblemsCursor, "1"); err != nil {
				t.Fatal(err)
			}
			_, err := st.DB().Exec(`CREATE TRIGGER reject_write BEFORE INSERT ON ` + tc.table + ` WHEN ` + tc.condition + ` BEGIN SELECT RAISE(ABORT, 'injected storage failure'); END`)
			if err != nil {
				t.Fatal(err)
			}
			ch := make(chan Progress, 100)
			err = tc.run(New(storageErrorClient(t), st, 100), ch)
			if err == nil || !strings.Contains(err.Error(), "injected storage failure") {
				t.Errorf("job returned %v, want storage error", err)
			}
			updates := drainProgress(ch)
			if len(updates) == 0 {
				t.Fatal("no progress")
			}
			last := updates[len(updates)-1]
			if !last.Finished || last.Err == nil || last.Note == "done" {
				t.Errorf("false completion progress: %+v", last)
			}
			for _, key := range []string{store.KeyProblemsSyncedAt, store.KeyCompaniesSyncedAt, store.KeyPlansSyncedAt, store.KeyContestsSyncedAt, store.PlanKey("course"), store.PackKey("company", "all"), store.ContestKey("weekly")} {
				if v, err := st.GetState(ctx, key); err != nil || v != "" {
					t.Errorf("failed sync stamped %s = %q: %v", key, v, err)
				}
			}
			if tc.wantCursor != "" {
				if v, err := st.GetState(ctx, store.KeyProblemsCursor); err != nil || v != tc.wantCursor {
					t.Errorf("cursor = %q, want %q: %v", v, tc.wantCursor, err)
				}
			}
		})
	}
}

func TestAccountStorageFailureKeepsPreviousIdentity(t *testing.T) {
	for _, key := range []string{store.KeyUsername, store.KeyIsPremium} {
		t.Run(key, func(t *testing.T) {
			ctx := context.Background()
			st := testStore(t)
			for k, v := range map[string]string{store.KeyUsername: "old-user", store.KeyIsPremium: "0"} {
				if err := st.SetState(ctx, k, v); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := st.DB().Exec(`CREATE TRIGGER reject_account BEFORE INSERT ON sync_state WHEN NEW.key = '` + key + `' BEGIN SELECT RAISE(ABORT, 'injected storage failure'); END`); err != nil {
				t.Fatal(err)
			}
			if _, err := New(storageErrorClient(t), st, 100).Account(ctx); err == nil {
				t.Error("account storage failure returned success")
			}
			for k, want := range map[string]string{store.KeyUsername: "old-user", store.KeyIsPremium: "0"} {
				if got, err := st.GetState(ctx, k); err != nil || got != want {
					t.Errorf("partial account update: %s = %q, want %q: %v", k, got, want, err)
				}
			}
		})
	}
}
