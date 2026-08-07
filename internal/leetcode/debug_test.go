package leetcode

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nano-AI/leetui/internal/auth"
)

// D-002's security invariant: a session cookie never reaches a log. The whole point of a
// debug log is that it gets pasted into a bug report, so this is the one test that has to
// hold no matter what else changes about tracing.
func TestDebugTraceNeverCarriesTheSession(t *testing.T) {
	const session = "eyJhbGciOiJIUzI1NiJ9.THIS_IS_THE_SECRET_PAYLOAD_NOBODY_MAY_LOG.sig_viM"
	const csrf = "csrf_token_value_that_must_not_appear"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":{}}`))
	}))
	defer srv.Close()

	var log strings.Builder
	c := New(
		WithCredentials(auth.Credentials{Session: session, CSRF: csrf}),
		WithHTTPClient(srv.Client()),
	)
	c.Debugf = func(format string, args ...any) {
		log.WriteString(strings.TrimSpace(fmt.Sprintf(format, args...)) + "\n")
	}

	// Drive a real request through the traced path.
	_ = c.graphql(context.Background(), "probe", "query{}", nil, nil)

	out := log.String()
	if out == "" {
		t.Fatal("nothing was traced; the guard below would pass vacuously")
	}
	for _, secret := range []string{session, csrf, "THIS_IS_THE_SECRET_PAYLOAD_NOBODY_MAY_LOG"} {
		if strings.Contains(out, secret) {
			t.Errorf("the trace carries a credential:\n%s", out)
		}
	}
	// It must still identify WHICH session, or a trace cannot tell two accounts apart.
	if !strings.Contains(out, auth.Redact(session)) {
		t.Errorf("the trace does not name the session in redacted form:\n%s", out)
	}
}
