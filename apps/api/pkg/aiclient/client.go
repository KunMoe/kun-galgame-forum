// Package aiclient calls an OpenAI-compatible chat completions endpoint for a
// single JSON answer.
package aiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var ErrUnconfigured = errors.New("aiclient: base url or token is not set")

// The Workers AI endpoint answers 429 once a per-minute request quota is used.
var backoff = []time.Duration{2 * time.Second, 5 * time.Second, 15 * time.Second}

type Client struct {
	base  string
	token string
	http  *http.Client
}

// New returns nil when either value is empty: callers treat a nil client as
// "the model is off".
func New(base, token string) *Client {
	if base == "" || token == "" {
		return nil
	}
	return &Client{
		base:  strings.TrimRight(base, "/"),
		token: token,
		http:  &http.Client{Timeout: 2 * time.Minute},
	}
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type request struct {
	Model          string    `json:"model"`
	Messages       []message `json:"messages"`
	Temperature    float64   `json:"temperature"`
	ResponseFormat struct {
		Type string `json:"type"`
	} `json:"response_format"`
}

type response struct {
	Choices []struct {
		Message      message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
}

// JSON sends one system and one user message and decodes the model's JSON
// object into out.
func (c *Client) JSON(ctx context.Context, model, system, user string, out any) error {
	if c == nil {
		return ErrUnconfigured
	}
	req := request{
		Model:       model,
		Messages:    []message{{Role: "system", Content: system}, {Role: "user", Content: user}},
		Temperature: 0,
	}
	req.ResponseFormat.Type = "json_object"
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	var raw []byte
	for attempt := 0; ; attempt++ {
		var status int
		raw, status, err = c.post(ctx, body)
		if err != nil {
			return err
		}
		if status == http.StatusOK {
			break
		}
		retryable := status == http.StatusTooManyRequests || status >= 500
		if !retryable || attempt >= len(backoff) {
			return fmt.Errorf("aiclient: %d: %.300s", status, raw)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("aiclient: %d after %d attempts: %w", status, attempt+1, ctx.Err())
		case <-time.After(backoff[attempt]):
		}
	}
	var parsed response
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return fmt.Errorf("aiclient: decode response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return errors.New("aiclient: no choices")
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	content = strings.TrimSuffix(strings.TrimPrefix(content, "```json"), "```")
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), out); err != nil {
		return fmt.Errorf("aiclient: decode answer %.200q: %w", content, err)
	}
	return nil
}

func (c *Client) post(ctx context.Context, body []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return raw, resp.StatusCode, err
}
