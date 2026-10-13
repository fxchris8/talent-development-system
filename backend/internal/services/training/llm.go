package training

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

	"github.com/sirupsen/logrus"
)

const (
	defaultLLMBaseURL = "https://openrouter.ai/api/v1"
	defaultLLMModel   = "poolside/laguna-s-2.1:free"
	llmMaxAttempts    = 5
)

// LLMClient talks to an OpenAI-compatible chat completions API (OpenRouter).
type LLMClient struct {
	log      *logrus.Logger
	http     *http.Client
	apiKey   string
	model    string
	baseURL  string
	provider string
}

func NewLLMClient(log *logrus.Logger, apiKey, model, baseURL, provider string) *LLMClient {
	if apiKey == "" {
		log.Warn("OPENROUTER_API_KEY kosong - panggilan LLM akan gagal")
	}
	if model == "" {
		model = defaultLLMModel
	}
	if baseURL == "" {
		baseURL = defaultLLMBaseURL
	}
	return &LLMClient{
		log:      log,
		http:     &http.Client{Timeout: 5 * time.Minute},
		apiKey:   apiKey,
		model:    model,
		baseURL:  strings.TrimRight(baseURL, "/"),
		provider: strings.TrimSpace(provider),
	}
}

// CompleteJSON sends the prompt and returns the model reply with markdown fences and
// surrounding chatter removed, so it can be passed straight to json.Unmarshal.
func (c *LLMClient) CompleteJSON(ctx context.Context, system, prompt string, temperature float64, maxTokens int) (string, error) {
	if c.apiKey == "" {
		return "", errors.New("OPENROUTER_API_KEY tidak di-set")
	}

	messages := []map[string]string{}
	if system != "" {
		messages = append(messages, map[string]string{"role": "system", "content": system})
	}
	messages = append(messages, map[string]string{"role": "user", "content": prompt})

	body := map[string]any{
		"model":       c.model,
		"messages":    messages,
		"temperature": temperature,
		"max_tokens":  maxTokens,
	}
	if c.provider != "" {
		body["provider"] = map[string]any{"order": []string{c.provider}, "allow_fallbacks": true}
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	var lastErr error
	for attempt := 1; attempt <= llmMaxAttempts; attempt++ {
		content, retryable, err := c.complete(ctx, payload)
		if err == nil {
			return extractJSON(content), nil
		}
		lastErr = err
		if !retryable || attempt == llmMaxAttempts {
			break
		}
		c.log.Warnf("[LLM] attempt %d/%d gagal: %v", attempt, llmMaxAttempts, err)
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(time.Duration(attempt) * 15 * time.Second):
		}
	}
	return "", lastErr
}

func (c *LLMClient) complete(ctx context.Context, payload []byte) (content string, retryable bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("X-Title", "Talent Development System")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", true, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", true, err
	}
	if resp.StatusCode != http.StatusOK {
		c.log.Errorf("OpenRouter error: status=%d body=%s", resp.StatusCode, truncate(string(raw), 500))
		retry := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
		return "", retry, fmt.Errorf("openrouter status %d", resp.StatusCode)
	}

	var out struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", false, err
	}
	if out.Error != nil {
		return "", true, fmt.Errorf("openrouter: %s", out.Error.Message)
	}
	if len(out.Choices) == 0 || strings.TrimSpace(out.Choices[0].Message.Content) == "" {
		return "", true, errors.New("openrouter tidak mengembalikan jawaban")
	}
	if out.Choices[0].FinishReason == "length" {
		c.log.Warn("[LLM] jawaban terpotong karena batas max_tokens")
	}
	return out.Choices[0].Message.Content, false, nil
}

// extractJSON strips ```json fences and any text around the outermost JSON object.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	if start, end := strings.Index(s, "{"), strings.LastIndex(s, "}"); start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
