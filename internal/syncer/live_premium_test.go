package syncer

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Nano-AI/leetui/internal/leetcode"
)

// TestLivePremiumSchema checks the premium GraphQL documents against the real endpoint.
// Opt in with:
//
//	LEETUI_LIVE=1 go test ./internal/syncer -run TestLivePremium -v
//
// It runs SIGNED OUT on purpose. Two of the three premium operations answer without a
// session, and the third has a distinguishable gated response, so schema drift is
// detectable without touching anyone's account or their keychain.
func TestLivePremiumSchema(t *testing.T) {
	if os.Getenv("LEETUI_LIVE") != "1" {
		t.Skip("set LEETUI_LIVE=1 to run against the real API")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cl := leetcode.New(leetcode.WithRateLimit(2))

	companies, err := cl.CompanyTags(ctx)
	if err != nil {
		t.Fatalf("CompanyTags: %v", err)
	}
	if len(companies) < 100 {
		t.Errorf("got %d companies, expected hundreds — the registry query may have changed",
			len(companies))
	}
	t.Logf("registry: %d companies, first is %s (%d problems)",
		len(companies), companies[0].Name, companies[0].QuestionCount)

	// Every timeframe must resolve for a company that certainly has one. A null pack here
	// means LeetCode renamed a window, which would silently empty the picker.
	for _, tf := range leetcode.Timeframes() {
		n, err := cl.PackSize(ctx, "google", tf)
		if err != nil {
			t.Errorf("PackSize(google, %s): %v", tf, err)
			continue
		}
		if n <= 0 {
			t.Errorf("google/%s reports %d problems, want more than 0", tf, n)
		}
		t.Logf("google/%-20s %d problems", tf, n)
	}

	// A slug LeetCode does not have must be an error, not an empty pack — the two are
	// indistinguishable on the wire and telling them apart is PackSize's whole job.
	if _, err := cl.PackSize(ctx, "definitely-not-a-company", leetcode.AllTime); !errors.Is(err, leetcode.ErrNotFound) {
		t.Errorf("PackSize on a bogus company returned %v, want ErrNotFound", err)
	}

	// Two Sum's editorial is public, so it proves the solution query end to end without
	// a subscription.
	sol, err := cl.Editorial(ctx, "two-sum")
	if err != nil {
		t.Fatalf("Editorial(two-sum): %v", err)
	}
	if len(sol.Content) < 200 {
		t.Errorf("editorial content is %d bytes, expected a full article", len(sol.Content))
	}

	// A premium editorial must come back as a gate carrying its public fields, not as a
	// bare error — that is what the lock state renders from (D-006).
	locked, err := cl.Editorial(ctx, "meeting-rooms-ii")
	if !errors.Is(err, leetcode.ErrPremiumRequired) {
		t.Errorf("Editorial(meeting-rooms-ii) signed out returned %v, want ErrPremiumRequired", err)
	}
	if locked == nil || locked.Title == "" {
		t.Errorf("a gated editorial came back with nothing to show: %+v", locked)
	}
}
