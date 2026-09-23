package leetcode

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type responseTransport func(*http.Request) (*http.Response, error)

func (f responseTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestStudyPlanWithEmptyChaptersIsGated(t *testing.T) {
	client := New(WithHTTPClient(&http.Client{Transport: responseTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(
			`{"data":{"studyPlanV2Detail":{"slug":"gated","questionNum":10,"premiumOnly":true,"planSubGroups":[{"name":"Arrays","questionNum":10,"questions":[]}]}}}`)), Request: r}, nil
	})}))
	p, err := client.StudyPlan(context.Background(), "gated")
	if !errors.Is(err, ErrPremiumRequired) {
		t.Fatalf("empty gated chapters returned %v; sync would replace cached contents", err)
	}
	if p.Slug != "gated" {
		t.Fatalf("lost gated plan metadata: %+v", p)
	}
}
