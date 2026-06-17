package a2a

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Client calls a remote agent that speaks the A2A protocol. It is the
// outbound counterpart to the gateway: where the gateway exposes a Go
// Micro agent over A2A, the Client lets a Go Micro agent or flow call an
// agent on any framework, by URL.
type Client struct {
	url  string
	http *http.Client
}

// NewClient returns a Client for the agent at url (its JSON-RPC endpoint,
// i.e. the `url` field of the agent's card).
func NewClient(url string) *Client {
	return &Client{url: url, http: &http.Client{Timeout: 60 * time.Second}}
}

// WithHTTPClient sets the underlying HTTP client (for timeouts, auth
// transports, etc.).
func (c *Client) WithHTTPClient(h *http.Client) *Client {
	if h != nil {
		c.http = h
	}
	return c
}

// Card fetches the remote agent's Agent Card.
func (c *Client) Card(ctx context.Context) (*AgentCard, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"/.well-known/agent.json", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent card: status %d", resp.StatusCode)
	}
	var card AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return nil, err
	}
	return &card, nil
}

// Send sends a text message to the remote agent and returns its reply.
// If the agent returns a task that isn't yet terminal, Send polls
// tasks/get until it completes or ctx is done.
func (c *Client) Send(ctx context.Context, text string) (string, error) {
	res, err := c.call(ctx, "message/send", sendParams{Message: Message{
		Role:      "user",
		Kind:      "message",
		MessageID: uuid.New().String(),
		Parts:     []Part{{Kind: "text", Text: text}},
	}})
	if err != nil {
		return "", err
	}

	// The result is a Message or a Task; the "kind" field disambiguates.
	var probe struct {
		Kind string `json:"kind"`
	}
	_ = json.Unmarshal(res, &probe)

	if probe.Kind == "message" {
		var m Message
		if err := json.Unmarshal(res, &m); err != nil {
			return "", err
		}
		return textOf(m.Parts), nil
	}

	var task Task
	if err := json.Unmarshal(res, &task); err != nil {
		return "", err
	}
	for !terminal(task.Status.State) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
		got, err := c.call(ctx, "tasks/get", getParams{ID: task.ID})
		if err != nil {
			return "", err
		}
		if err := json.Unmarshal(got, &task); err != nil {
			return "", err
		}
	}
	if task.Status.State != stateCompleted {
		return "", fmt.Errorf("remote task %s ended in state %q", task.ID, task.Status.State)
	}
	return artifactsText(task.Artifacts), nil
}

// call performs one JSON-RPC request and returns the raw result.
func (c *Client) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      uuid.New().String(),
		"method":  method,
		"params":  params,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out struct {
		Result json.RawMessage `json:"result"`
		Error  *rpcError       `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.Error != nil {
		return nil, fmt.Errorf("a2a %s: %s (%d)", method, out.Error.Message, out.Error.Code)
	}
	return out.Result, nil
}

func terminal(state string) bool {
	switch state {
	case "completed", "failed", "canceled", "rejected":
		return true
	}
	return false
}

func artifactsText(arts []Artifact) string {
	var parts []Part
	for _, a := range arts {
		parts = append(parts, a.Parts...)
	}
	return textOf(parts)
}
