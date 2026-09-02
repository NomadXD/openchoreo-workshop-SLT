package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// BackendClient talks to subscription-service and network-ops-service on
// behalf of the employee dashboard. Unlike AgentClient (one long-lived
// streaming request per chat turn), these are simple request/response JSON
// calls, proxied straight through to the browser by the dashboard handlers.
type BackendClient struct {
	subscriptionBaseURL string
	networkOpsBaseURL   string
	http                *http.Client
}

func newBackendClient(subscriptionBaseURL, networkOpsBaseURL string) *BackendClient {
	return &BackendClient{
		subscriptionBaseURL: strings.TrimRight(subscriptionBaseURL, "/"),
		networkOpsBaseURL:   strings.TrimRight(networkOpsBaseURL, "/"),
		http:                &http.Client{},
	}
}

// backendError carries the upstream status code so handlers can propagate
// a real 404 ("no such customer") instead of flattening every failure to
// a 500/502.
type backendError struct {
	statusCode int
	body       []byte
}

func (e *backendError) Error() string {
	return fmt.Sprintf("upstream returned %d: %s", e.statusCode, string(e.body))
}

func (c *BackendClient) doJSON(ctx context.Context, method, baseURL, path string, query url.Values, body any) (json.RawMessage, error) {
	fullURL := baseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshalling request: %w", err)
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reader)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling %s: %w", fullURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response from %s: %w", fullURL, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &backendError{statusCode: resp.StatusCode, body: respBody}
	}
	if len(respBody) == 0 {
		return json.RawMessage("null"), nil
	}
	return json.RawMessage(respBody), nil
}

func (c *BackendClient) subscriptionGet(ctx context.Context, path string, query url.Values) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodGet, c.subscriptionBaseURL, path, query, nil)
}

func (c *BackendClient) networkOpsGet(ctx context.Context, path string, query url.Values) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodGet, c.networkOpsBaseURL, path, query, nil)
}

func (c *BackendClient) networkOpsPatch(ctx context.Context, path string, body any) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodPatch, c.networkOpsBaseURL, path, nil, body)
}
