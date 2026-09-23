package syncer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/store"
)

func TestRestartFailureResetsOldCheckpoint(t *testing.T) {
	ctx := context.Background()
	st := testStore(t)
	if err := st.SetState(ctx, store.KeyProblemsCursor, "200"); err != nil {
		t.Fatal(err)
	}
	cl, _ := newFake(t, &fakeLeetCode{total: 300, failAt: 1})
	ch := make(chan Progress, 10)
	if err := New(cl, st, 100).Problems(ctx, ch, false); err == nil {
		t.Fatal("expected HTTP failure")
	}
	if cursor, _ := st.GetState(ctx, store.KeyProblemsCursor); cursor != "0" {
		t.Fatalf("failed restart kept stale cursor %q; resume would skip first 200 rows", cursor)
	}
}

func TestIncompleteProblemPageDoesNotCompleteSync(t *testing.T) {
	for _, body := range []string{
		`{"data":{"problemsetQuestionList":{"total":300,"questions":[]}}}`,
		`{"data":{"problemsetQuestionList":null}}`,
		`{"data":null}`,
	} {
		t.Run(body, func(t *testing.T) {
			ctx := context.Background()
			st := testStore(t)
			if err := st.SetState(ctx, store.KeyProblemsCursor, "100"); err != nil {
				t.Fatal(err)
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
			defer srv.Close()
			cl := leetcode.New(leetcode.WithHTTPClient(&http.Client{Transport: redirectTo(srv.URL)}))
			ch := make(chan Progress, 10)
			if err := New(cl, st, 100).Problems(ctx, ch, true); err == nil {
				t.Error("incomplete response reported success")
			}
			if cursor, _ := st.GetState(ctx, store.KeyProblemsCursor); cursor != "100" {
				t.Errorf("lost resumable cursor: %q", cursor)
			}
			if stamp, _ := st.GetState(ctx, store.KeyProblemsSyncedAt); stamp != "" {
				t.Errorf("stamped incomplete sync: %q", stamp)
			}
		})
	}
}
