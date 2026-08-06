package leetcode

import (
	"errors"
	"net/http"
	"time"

	"github.com/grootbeat/leetui/internal/auth"
	"golang.org/x/time/rate"
)

// Endpoints.
const (
	BaseURL    = "https://leetcode.com"
	GraphQLURL = BaseURL + "/graphql/"

	// userAgent identifies leetui honestly rather than impersonating a browser. If
	// LeetCode ever blocks it, that is their call to make and the user should see it
	// as a clear error rather than have the app disguise itself.
	userAgent = "leetui (+https://github.com/grootbeat/leetui)"
)

// Errors callers are expected to branch on.
var (
	// ErrSessionExpired means the stored cookies were rejected. The TUI prompts for
	// re-auth in place rather than failing the user's action outright.
	ErrSessionExpired = auth.ErrSessionExpired

	// ErrPremiumRequired means the content is gated and this account cannot see it.
	// Panes render a lock state rather than an error (D-006).
	ErrPremiumRequired = errors.New("leetcode premium required")

	// ErrRateLimited means LeetCode pushed back. Sync jobs pause and resume.
	ErrRateLimited = errors.New("rate limited by leetcode")

	// ErrNotFound means the slug does not exist.
	ErrNotFound = errors.New("not found")
)

// Client talks to LeetCode.
//
// The zero value is not usable; construct with New.
type Client struct {
	http    *http.Client
	limiter *rate.Limiter
	creds   auth.Credentials

	// Debugf, when set, receives request traces with credentials already redacted.
	// It is safe to point at a log file.
	Debugf func(format string, args ...any)
}

// Option configures a Client.
type Option func(*Client)

// WithCredentials attaches a session. Without one the client can still read public
// problem data — which is what makes an unauthenticated first sync possible.
func WithCredentials(c auth.Credentials) Option {
	return func(cl *Client) { cl.creds = c }
}

// WithRateLimit sets the ceiling for ALL outbound requests.
func WithRateLimit(perSecond float64) Option {
	return func(cl *Client) {
		if perSecond <= 0 {
			perSecond = 2
		}
		// Burst of 1: a bulk sync should be a steady trickle, not a stutter of bursts
		// that looks like scraping.
		cl.limiter = rate.NewLimiter(rate.Limit(perSecond), 1)
	}
}

// WithHTTPClient overrides the transport, for tests.
func WithHTTPClient(h *http.Client) Option {
	return func(cl *Client) { cl.http = h }
}

// New builds a Client.
func New(opts ...Option) *Client {
	c := &Client{
		http:    &http.Client{Timeout: 30 * time.Second},
		limiter: rate.NewLimiter(2, 1),
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Authenticated reports whether a session is attached.
func (c *Client) Authenticated() bool { return c.creds.Valid() }

// SetCredentials swaps the session in place, for re-auth without rebuilding the client.
func (c *Client) SetCredentials(cr auth.Credentials) { c.creds = cr }
