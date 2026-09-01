package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// AgentClient talks to the internal chat-agent service.
type AgentClient struct {
	baseURL string
	http    *http.Client
}

func newAgentClient(baseURL string) *AgentClient {
	return &AgentClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		// No overall timeout: this is a streaming response and turns can
		// legitimately take a while. Callers drive cancellation via ctx.
		http: &http.Client{},
	}
}

// AgentRequest is the body chat-gateway sends to POST /agent/stream.
type AgentRequest struct {
	Role             string        `json:"role"`
	ActorID          string        `json:"actorId"`
	TargetCustomerID string        `json:"targetCustomerId"`
	ConversationID   string        `json:"conversationId"`
	Message          string        `json:"message"`
	History          []HistoryItem `json:"history"`
}

// StreamTurn POSTs a turn to chat-agent and returns the response for the
// caller to read as NDJSON. The caller MUST close the returned body.
// An error is returned if the request fails outright or the response
// status is not 2xx (i.e. before any streaming has started).
func (c *AgentClient) StreamTurn(ctx context.Context, req AgentRequest) (io.ReadCloser, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshalling agent request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/agent/stream", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building agent request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("calling chat-agent: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("chat-agent returned status %d: %s", resp.StatusCode, string(snippet))
	}

	return resp.Body, nil
}
