package leetcode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Nano-AI/leetui/internal/auth"
)

// ---------------------------------------------------------------------------
// Transport
// ---------------------------------------------------------------------------

type gqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
	Operation string         `json:"operationName,omitempty"`
}

type gqlError struct {
	Message string `json:"message"`
}

type gqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []gqlError      `json:"errors"`
}

// graphql executes a query and unmarshals data into out.
//
// It blocks on the rate limiter first — every caller, no exceptions.
func (c *Client) graphql(ctx context.Context, op, query string, vars map[string]any, out any) error {
	if err := c.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter: %w", err)
	}

	body, err := json.Marshal(gqlRequest{Query: query, Variables: vars, Operation: op})
	if err != nil {
		return fmt.Errorf("encode %s: %w", op, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, GraphQLURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build %s: %w", op, err)
	}
	c.setHeaders(req, BaseURL+"/problemset/all/")

	c.debugRequest(op, req)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return fmt.Errorf("read %s: %w", op, err)
	}
	c.debugResponse(op, resp.StatusCode, raw)

	if err := c.statusError(op, resp.StatusCode, raw); err != nil {
		return err
	}

	var gr gqlResponse
	if err := json.Unmarshal(raw, &gr); err != nil {
		// The body goes in the trace as well as the error. A decode failure is the one
		// case where the message alone is not enough — "cannot unmarshal string into
		// Go struct field" names the field and says nothing about what arrived, and
		// that is exactly the bug that failed every submission for a week.
		c.debugBody(op, raw)
		return fmt.Errorf("decode %s: %w (body starts %q)", op, err, snippet(raw))
	}
	if len(gr.Errors) > 0 {
		return c.graphqlError(op, gr.Errors)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(gr.Data, out); err != nil {
		return fmt.Errorf("decode %s data: %w", op, err)
	}
	return nil
}

// setHeaders applies the headers LeetCode requires. The Referer is not optional —
// requests without a plausible one are rejected.
func (c *Client) setHeaders(req *http.Request, referer string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", referer)
	req.Header.Set("Origin", BaseURL)

	if c.creds.Valid() {
		req.Header.Set("Cookie", c.creds.Cookie())
		req.Header.Set("x-csrftoken", c.creds.CSRF)
	}
}

// statusError maps an HTTP status onto a typed error the TUI can branch on.
func (c *Client) statusError(op string, code int, body []byte) error {
	switch {
	case code == http.StatusOK:
		return nil
	case code == http.StatusUnauthorized || code == http.StatusForbidden:
		// LeetCode returns 403 both for an expired session and for premium-gated
		// content. The body distinguishes them.
		if bytes.Contains(bytes.ToLower(body), []byte("subscribe")) ||
			bytes.Contains(bytes.ToLower(body), []byte("premium")) {
			return ErrPremiumRequired
		}
		return ErrSessionExpired
	case code == http.StatusNotFound:
		return ErrNotFound
	case code == http.StatusTooManyRequests:
		return ErrRateLimited
	default:
		return fmt.Errorf("%s: leetcode returned %d: %s", op, code, snippet(body))
	}
}

// graphqlError maps GraphQL-level errors, which arrive with HTTP 200.
func (c *Client) graphqlError(op string, errs []gqlError) error {
	msgs := make([]string, 0, len(errs))
	for _, e := range errs {
		low := strings.ToLower(e.Message)
		switch {
		case strings.Contains(low, "authentication") || strings.Contains(low, "signed in") ||
			strings.Contains(low, "login"):
			return ErrSessionExpired
		case strings.Contains(low, "subscri") || strings.Contains(low, "premium"):
			return ErrPremiumRequired
		}
		msgs = append(msgs, e.Message)
	}
	return fmt.Errorf("%s: %s", op, strings.Join(msgs, "; "))
}

// debugRequest traces a request with credentials redacted.
//
// Never change this to dump req.Header wholesale — that is how a session ends up in a
// log file the user pastes into a bug report.
func (c *Client) debugRequest(op string, req *http.Request) {
	if c.Debugf == nil {
		return
	}
	authState := "anonymous"
	if c.creds.Valid() {
		authState = "session " + auth.Redact(c.creds.Session)
	}
	c.Debugf("-> %s %s [%s]", op, req.URL.Path, authState)
}

// debugResponse traces what came back.
//
// The body is NOT logged here. A successful response is thousands of problems, and a
// debug log nobody can read is a debug log nobody reads.
func (c *Client) debugResponse(op string, status int, body []byte) {
	if c.Debugf == nil {
		return
	}
	c.Debugf("<- %s %d (%d bytes)", op, status, len(body))
}

// debugBody records a response that could not be understood.
//
// Capped, and only on a failure — this is for the case where knowing what the server
// actually sent is the whole diagnosis.
func (c *Client) debugBody(op string, body []byte) {
	if c.Debugf == nil {
		return
	}
	const max = 2000
	s := string(body)
	if len(s) > max {
		s = s[:max] + "…"
	}
	c.Debugf("!! %s could not be decoded; body: %s", op, s)
}

func snippet(b []byte) string {
	const n = 200
	s := strings.TrimSpace(string(b))
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
