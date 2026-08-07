package leetcode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// postJSON sends a JSON body to a problem-scoped endpoint.
func (c *Client) postJSON(ctx context.Context, path, slug string, body, out any) error {
	if err := c.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter: %w", err)
	}
	if !c.creds.Valid() {
		return ErrSessionExpired
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, BaseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	c.setHeaders(req, BaseURL+"/problems/"+slug+"/")

	return c.doJSON(req, path, out)
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	if err := c.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter: %w", err)
	}
	if !c.creds.Valid() {
		return ErrSessionExpired
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, BaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	c.setHeaders(req, BaseURL+"/problemset/all/")

	return c.doJSON(req, path, out)
}

func (c *Client) doJSON(req *http.Request, op string, out any) error {
	c.debugRequest(op, req)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("read %s: %w", op, err)
	}
	c.debugResponse(op, resp.StatusCode, raw)

	if err := c.statusError(op, resp.StatusCode, raw); err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		// This is the path the judge answers on, and the path where a field changing
		// shape failed every submission. The body is the whole diagnosis.
		c.debugBody(op, raw)
		return fmt.Errorf("decode %s: %w (body starts %q)", op, err, snippet(raw))
	}
	return nil
}
