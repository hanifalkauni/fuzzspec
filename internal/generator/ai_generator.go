package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fuzzspec/fuzzspec/internal/parser"
)

// AIGenerator orchestrates LLM API calls to generate semantic edge cases.
type AIGenerator struct {
	provider   string // "gemini", "openai", "local"
	apiKey     string
	model      string
	cache      *VectorCache
	httpClient *http.Client
}

// NewAIGenerator creates an AI generator configured with provider, model, and cache.
func NewAIGenerator(provider, model, cacheDir string, cacheEnabled bool) *AIGenerator {
	if provider == "" {
		provider = "gemini"
	}
	if model == "" {
		switch strings.ToLower(provider) {
		case "gemini":
			model = "gemini-1.5-flash"
		case "anthropic":
			model = "claude-3-5-haiku-20241022"
		default:
			model = "gpt-4o-mini"
		}
	}

	var apiKey string
	switch strings.ToLower(provider) {
	case "gemini":
		apiKey = os.Getenv("GEMINI_API_KEY")
	case "openai":
		apiKey = os.Getenv("OPENAI_API_KEY")
	case "anthropic":
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	return &AIGenerator{
		provider:   provider,
		apiKey:     apiKey,
		model:      model,
		cache:      NewVectorCache(cacheDir, cacheEnabled),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// GenerateAIVectors generates AI-powered semantic edge cases for an endpoint operation.
func (g *AIGenerator) GenerateAIVectors(ctx context.Context, op parser.EndpointOperation) ([]TestVector, error) {
	// 1. Check local cache first
	cacheKey := g.cache.ComputeKey(op)
	if cachedVectors, found := g.cache.Get(cacheKey); found {
		return cachedVectors, nil
	}

	// 2. If no API key is provided, return empty (fallback gracefully to rule-based)
	if g.apiKey == "" {
		return nil, nil
	}

	// 3. Build token-minified prompt
	prompt, err := BuildFuzzingPrompt(op)
	if err != nil {
		return nil, fmt.Errorf("failed to build AI prompt: %w", err)
	}

	// 4. Call LLM Provider
	var rawJSON string
	switch strings.ToLower(g.provider) {
	case "gemini":
		rawJSON, err = g.callGemini(ctx, prompt)
	case "openai", "local":
		rawJSON, err = g.callOpenAI(ctx, prompt)
	case "anthropic":
		rawJSON, err = g.callAnthropic(ctx, prompt)
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", g.provider)
	}

	if err != nil {
		return nil, fmt.Errorf("AI provider error (%s): %w", g.provider, err)
	}

	// 5. Parse and Normalize Output Vectors
	vectors, err := g.parseLLMOutput(rawJSON, op)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	// 6. Save to cache
	_ = g.cache.Set(cacheKey, vectors)

	return vectors, nil
}

func (g *AIGenerator) callGemini(ctx context.Context, prompt string) (string, error) {
	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", g.model, g.apiKey)

	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"responseMimeType": "application/json",
			"temperature":      0.4,
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Gemini API error (HTTP %d): %s", resp.StatusCode, string(respData))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(respData, &geminiResp); err != nil {
		return "", err
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response received from Gemini")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

func (g *AIGenerator) callOpenAI(ctx context.Context, prompt string) (string, error) {
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	apiURL := strings.TrimRight(baseURL, "/") + "/chat/completions"

	reqBody := map[string]interface{}{
		"model": g.model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are an API contract security fuzzer. Return JSON array only."},
			{"role": "user", "content": prompt},
		},
		"response_format": map[string]string{"type": "json_object"},
		"temperature":     0.4,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API error (HTTP %d): %s", resp.StatusCode, string(respData))
	}

	var openaiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respData, &openaiResp); err != nil {
		return "", err
	}

	if len(openaiResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from OpenAI")
	}

	return openaiResp.Choices[0].Message.Content, nil
}

func (g *AIGenerator) callAnthropic(ctx context.Context, prompt string) (string, error) {
	apiURL := "https://api.anthropic.com/v1/messages"

	reqBody := map[string]interface{}{
		"model":      g.model,
		"max_tokens": 4096,
		"system":     "You are an API contract security fuzzer. Return JSON array only.",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.4,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", g.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Anthropic API error (HTTP %d): %s", resp.StatusCode, string(respData))
	}

	var anthropicResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(respData, &anthropicResp); err != nil {
		return "", err
	}

	for _, c := range anthropicResp.Content {
		if c.Type == "text" {
			return c.Text, nil
		}
	}

	return "", fmt.Errorf("empty text response from Anthropic")
}

func (g *AIGenerator) parseLLMOutput(rawJSON string, op parser.EndpointOperation) ([]TestVector, error) {
	// Clean markdown backticks if present
	clean := strings.TrimSpace(rawJSON)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)

	var rawVectors []struct {
		Scenario       string                 `json:"scenario"`
		Description    string                 `json:"description"`
		MutationType   string                 `json:"mutation_type"`
		TargetLocation string                 `json:"target_location"`
		TargetField    string                 `json:"target_field"`
		PathParams     map[string]string      `json:"path_params"`
		QueryParams    map[string]string      `json:"query_params"`
		Headers        map[string]string      `json:"headers"`
		Body           interface{}            `json:"body"`
	}

	if err := json.Unmarshal([]byte(clean), &rawVectors); err != nil {
		// Attempt object unwrap if wrapped under e.g. "vectors": [...]
		var wrapped map[string]json.RawMessage
		if errWrap := json.Unmarshal([]byte(clean), &wrapped); errWrap == nil {
			for _, v := range wrapped {
				if errSub := json.Unmarshal(v, &rawVectors); errSub == nil {
					break
				}
			}
		}
	}

	var normalized []TestVector
	for i, rv := range rawVectors {
		id := fmt.Sprintf("vec_ai_%s_%s_%03d", strings.ToLower(op.Method), sanitizePath(op.Path), i+1)
		normalized = append(normalized, TestVector{
			ID:                   id,
			Path:                 op.Path,
			Method:               op.Method,
			Scenario:             rv.Scenario,
			Description:          rv.Description,
			MutationType:         MutationTypeAISemantic,
			TargetLocation:       rv.TargetLocation,
			TargetField:          rv.TargetField,
			PathParams:           rv.PathParams,
			QueryParams:          rv.QueryParams,
			Headers:              rv.Headers,
			Body:                 rv.Body,
			ExpectedStatusFamily: "4xx",
			Source:               "ai",
		})
	}

	return normalized, nil
}
