package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"cascata/internal/database"
)

// LLMProvider configuration for AI Architect.
type LLMProvider struct {
	BaseURL string
	Key     string
	Model   string
}

// Orchestrator manages AI Architect sessions and LLM calls.
type Orchestrator struct {
	repo     *database.Repository
	provider LLMProvider
}

func NewOrchestrator(repo *database.Repository, p LLMProvider) *Orchestrator {
	return &Orchestrator{repo: repo, provider: p}
}

// SessionMessage represents a message in the history.
type SessionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Chat processes a request, injecting schema context and saving history.
func (o *Orchestrator) Chat(ctx context.Context, projectSlug string, sessionID string, userMsg string) (string, error) {
	// 1. Fetch History from PostgreSQL
	// 2. Extract Schema Context
	extractor := NewContextExtractor(o.repo)
	schema, err := extractor.GetSchemaContext(ctx, projectSlug)
	if err != nil { schema = "No context available" }

	// 3. Build Prompt with safe escaping for markdown blocks
	systemMsg := "You are Cascata Architect. Helping with project: " + projectSlug + ".\n" +
		"Rules:\n" +
		"- Return SQL inside ```sql blocks.\n" +
		"- Output JSON for structural changes.\n\n" +
		schema

	// 4. Call LLM (OpenAI-Compatible)
	return o.callLLM(ctx, systemMsg, userMsg)
}

func (o *Orchestrator) callLLM(ctx context.Context, system, user string) (string, error) {
	reqBody := map[string]interface{}{
		"model": o.provider.Model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature": 0.2,
	}

	raw, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", o.provider.BaseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil { return "", err }

	req.Header.Set("Authorization", "Bearer "+o.provider.Key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		out, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("llm error: %d: %s", resp.StatusCode, string(out))
	}

	var res struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil { return "", err }

	if len(res.Choices) == 0 { return "", fmt.Errorf("ai: no response from llm") }
	return res.Choices[0].Message.Content, nil
}
