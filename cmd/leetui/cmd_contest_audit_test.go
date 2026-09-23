package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Nano-AI/leetui/internal/config"
	"github.com/Nano-AI/leetui/internal/leetcode"
	"github.com/Nano-AI/leetui/internal/store"
	"github.com/Nano-AI/leetui/internal/syncer"
)

type offlineTransport struct{}

func (offlineTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("offline test transport")
}

func offlineContestApp(t *testing.T) *app {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	c := leetcode.New(leetcode.WithHTTPClient(&http.Client{Transport: offlineTransport{}}), leetcode.WithRateLimit(100000))
	return &app{store: st, client: c, sync: syncer.New(c, st, 10), cfg: config.Config{Workspace: t.TempDir(), DefaultLang: "python3"}}
}

func TestContestPullPartialFailureIsNotSuccess(t *testing.T) {
	a := offlineContestApp(t)
	ctx := context.Background()
	c := &leetcode.Contest{ContestBrief: leetcode.ContestBrief{Slug: "weekly-test", StartTime: 1, Duration: 1},
		Questions: []leetcode.ContestQuestion{{Slug: "good", Credit: 3}, {Slug: "bad", Credit: 4}}}
	if err := a.store.SetContest(ctx, c); err != nil {
		t.Fatal(err)
	}
	if err := a.store.SetDetail(ctx, &leetcode.Problem{Slug: "good", Title: "Good", FrontendID: "1", Content: "<p>Good</p>",
		Snippets: []leetcode.CodeSnippet{{LangSlug: "python3", Code: "class Solution: pass"}}}); err != nil {
		t.Fatal(err)
	}
	code, err := contestPull(a, c.Slug, "python3")
	if code != exitProblem || err == nil {
		t.Errorf("partial pull = %d, %v; want 2 and summary error", code, err)
	}
	if _, err := os.Stat(filepath.Join(a.cfg.Workspace, "0001-good", "solution.py")); err != nil {
		t.Fatalf("successful problem was not preserved: %v", err)
	}
}

func TestContestNetworkFailureDoesNotClaimSessionExpired(t *testing.T) {
	a := offlineContestApp(t)
	f, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	previous := os.Stderr
	os.Stderr = f
	defer func() { os.Stderr = previous }()
	warnUnregistered(context.Background(), a, store.Contest{Slug: "weekly-test", StartTime: time.Now().Add(time.Hour).Unix()})
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "expired") || strings.Contains(string(b), "NOT REGISTERED") {
		t.Fatalf("network failure misreported as account state: %s", b)
	}
}
