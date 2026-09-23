package leetcode

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Nano-AI/leetui/internal/auth"
)

// TestLiveContestEndpoints records how LeetCode's /contest/api/ paths respond, which is
// the evidence behind two claims in D-036. Opt in with:
//
//	LEETUI_LIVE=1 go test ./internal/leetcode -run TestLiveContest -v
//
// It needs a SESSION, and that is the finding: every /contest/api/ path answers 403 with a
// Cloudflare challenge to an anonymous client whatever the user agent, and 200 with a
// session attached. The endpoints are not bot-walled, they are signed-in-only.
//
// The submit path is checked with a GET on purpose. A wrong path returns 404; 405 Method
// Not Allowed means the path exists and wants a POST, and that is as far as the contest
// judge can be verified without making a real scoring submission to a live contest.
//
// Read-only, and it never prints a credential.
func TestLiveContestEndpoints(t *testing.T) {
	if os.Getenv("LEETUI_LIVE") != "1" {
		t.Skip("set LEETUI_LIVE=1 to run")
	}
	creds, _, err := auth.Load()
	if err != nil {
		t.Fatalf("no credentials: %v", err)
	}
	t.Logf("credentials valid: %v", creds.Valid())

	targets := []struct{ name, url, referer string }{
		{"contest info (REST)",
			BaseURL + "/contest/api/info/weekly-contest-500/",
			BaseURL + "/contest/weekly-contest-500/"},
		{"ordinary submit (GET)",
			BaseURL + "/problems/two-sum/submit/",
			BaseURL + "/problems/two-sum/"},
		{"contest submit (GET)",
			BaseURL + "/contest/api/weekly-contest-500/problems/two-sum/submit/",
			BaseURL + "/contest/weekly-contest-500/problems/two-sum/"},
	}

	client := &http.Client{Timeout: 30 * time.Second}
	for _, tg := range targets {
		req, _ := http.NewRequest(http.MethodGet, tg.url, nil)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Referer", tg.referer)
		req.Header.Set("Origin", BaseURL)
		req.Header.Set("Cookie", creds.Cookie())
		req.Header.Set("x-csrftoken", creds.CSRF)

		resp, err := client.Do(req)
		if err != nil {
			t.Logf("%-26s ERROR %v", tg.name, err)
			continue
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 400))
		resp.Body.Close()

		body := strings.ToLower(string(raw))
		kind := "other"
		switch {
		case strings.Contains(body, "just a moment"), strings.Contains(body, "challenges.cloudflare"):
			kind = "CLOUDFLARE CHALLENGE"
		case strings.HasPrefix(strings.TrimSpace(string(raw)), "{"):
			kind = "JSON"
		case strings.Contains(body, "<!doctype html"):
			kind = "HTML"
		}
		t.Logf("%-26s %d  %s", tg.name, resp.StatusCode, kind)
		time.Sleep(1200 * time.Millisecond)
	}
}
