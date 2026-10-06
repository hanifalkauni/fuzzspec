package generator

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fuzzspec/fuzzspec/internal/parser"
)

// SimplifiedEndpoint represents a minified schema structure sent to the LLM to minimize token cost.
type SimplifiedEndpoint struct {
	Method      string                 `json:"method"`
	Path        string                 `json:"path"`
	Summary     string                 `json:"summary,omitempty"`
	Parameters  []SimplifiedParam      `json:"parameters,omitempty"`
	RequestBody map[string]interface{} `json:"request_body_schema,omitempty"`
}

type SimplifiedParam struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Required bool   `json:"required"`
	Type     string `json:"type"`
	Minimum  *int64 `json:"minimum,omitempty"`
	Maximum  *int64 `json:"maximum,omitempty"`
}

// BuildFuzzingPrompt generates a token-minified prompt for the LLM with strict JSON structured output requirements.
func BuildFuzzingPrompt(op parser.EndpointOperation) (string, error) {
	simplified := SimplifiedEndpoint{
		Method:  op.Method,
		Path:    op.Path,
		Summary: op.Summary,
	}

	for _, p := range op.Parameters {
		paramType := "string"
		var minVal, maxVal *int64
		if p.Schema != nil {
			if p.Schema.Type != nil && len(p.Schema.Type.Slice()) > 0 {
				paramType = p.Schema.Type.Slice()[0]
			}
			if p.Schema.Min != nil {
				v := int64(*p.Schema.Min)
				minVal = &v
			}
			if p.Schema.Max != nil {
				v := int64(*p.Schema.Max)
				maxVal = &v
			}
		}

		simplified.Parameters = append(simplified.Parameters, SimplifiedParam{
			Name:     p.Name,
			In:       p.In,
			Required: p.Required,
			Type:     paramType,
			Minimum:  minVal,
			Maximum:  maxVal,
		})
	}

	if op.RequestBody != nil {
		simplified.RequestBody = make(map[string]interface{})
		for mediaType, schema := range op.RequestBody.Content {
			if schema != nil {
				simplified.RequestBody[mediaType] = minifySchema(schema)
			}
		}
	}

	specJSON, err := json.Marshal(simplified)
	if err != nil {
		return "", fmt.Errorf("failed to serialize endpoint for prompt: %w", err)
	}

	prompt := fmt.Sprintf(`You are an expert Security & QA Fuzzing Engineer. Generate sophisticated, realistic edge-case payloads to test for unhandled crashes (HTTP 500), nil pointer panics, integer overflow, and type injection on this API endpoint.

Target Endpoint Schema:
%s

Instructions:
1. Generate between 3 and 8 diverse edge cases (e.g. realistic invalid formats, boundary values, empty nested structures, SQL/command injection, type mismatch).
2. Return ONLY a valid JSON array of objects. Do not wrap in markdown or backticks.
3. Output format for each object in the array:
[
  {
    "scenario": "Short description of the test scenario",
    "description": "Why this edge case could crash the backend",
    "mutation_type": "AI_SEMANTIC_EDGE",
    "target_location": "body | query | path | header",
    "target_field": "field_name",
    "path_params": {},
    "query_params": {},
    "headers": {},
    "body": {}
  }
]`, string(specJSON))

	return strings.TrimSpace(prompt), nil
}

func minifySchema(schema interface{}) map[string]interface{} {
	data, err := json.Marshal(schema)
	if err != nil {
		return map[string]interface{}{}
	}
	var res map[string]interface{}
	if err := json.Unmarshal(data, &res); err != nil {
		return map[string]interface{}{}
	}
	// Strip heavy documentation fields to compress token usage
	delete(res, "description")
	delete(res, "example")
	delete(res, "examples")
	delete(res, "externalDocs")
	return res
}
