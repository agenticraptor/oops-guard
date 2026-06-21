package llm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

// DefaultAnthropicModel is a fast, inexpensive default well suited to a quick
// risk second-opinion. Override it with --model or OOPS_GUARD_MODEL.
const DefaultAnthropicModel = "claude-haiku-4-5-20251001"

type anthropicClient struct {
	apiKey  string
	model   string
	baseURL string
}

func newAnthropic(model, baseURL string) (Client, error) {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return nil, ErrNoCredentials
	}
	if model == "" {
		model = DefaultAnthropicModel
	}
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	return &anthropicClient{apiKey: key, model: model, baseURL: strings.TrimRight(baseURL, "/")}, nil
}

func (c *anthropicClient) Name() string  { return "anthropic" }
func (c *anthropicClient) Model() string { return c.model }

func (c *anthropicClient) Complete(ctx context.Context, req Request) (string, error) {
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	payload := map[string]any{
		"model":       c.model,
		"max_tokens":  maxTokens,
		"temperature": req.Temperature,
		"messages": []map[string]any{
			{"role": "user", "content": req.Prompt},
		},
	}
	if req.System != "" {
		payload["system"] = req.System
	}
	headers := map[string]string{
		"x-api-key":         c.apiKey,
		"anthropic-version": "2023-06-01",
	}

	var out struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := postJSON(ctx, c.baseURL+"/v1/messages", headers, payload, &out); err != nil {
		return "", err
	}
	if out.Error != nil {
		return "", errors.New(out.Error.Message)
	}
	var b strings.Builder
	for _, blk := range out.Content {
		if blk.Type == "text" {
			fmt.Fprint(&b, blk.Text)
		}
	}
	return strings.TrimSpace(b.String()), nil
}
