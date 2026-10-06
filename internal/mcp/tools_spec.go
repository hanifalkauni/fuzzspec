package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fuzzspec/fuzzspec/internal/parser"
)

func getInspectSpecTool() Tool {
	return Tool{
		Name:        "inspect_spec",
		Description: "Parses an OpenAPI 3.0/3.1 specification (local file or auto-discovered URL) and returns endpoints, parameters, request body schemas, and response contracts.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]PropertyDoc{
				"spec_path": {
					Type:        "string",
					Description: "Path to OpenAPI YAML/JSON file or remote HTTP URL",
				},
				"target_url": {
					Type:        "string",
					Description: "Target base URL (e.g. http://localhost:8080) to auto-discover spec if spec_path is omitted",
				},
			},
		},
	}
}

func handleInspectSpec(ctx context.Context, args map[string]interface{}) (ToolCallResult, error) {
	specPath, _ := args["spec_path"].(string)
	targetURL, _ := args["target_url"].(string)

	if specPath == "" && targetURL != "" {
		// Auto-discover spec
		discoverer := parser.NewAutoDiscoverer()
		discovered, err := discoverer.Discover(ctx, targetURL, nil)
		if err != nil {
			return ToolCallResult{
				Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Auto-discovery failed: %v", err)}},
				IsError: true,
			}, nil
		}
		specPath = discovered
	}

	if specPath == "" {
		return ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: "Either 'spec_path' or 'target_url' is required."}},
			IsError: true,
		}, nil
	}

	p := parser.NewParser()
	doc, err := p.Parse(ctx, specPath)
	if err != nil {
		return ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Failed to parse OpenAPI spec: %v", err)}},
			IsError: true,
		}, nil
	}

	type ParamInfo struct {
		Name     string `json:"name"`
		In       string `json:"in"`
		Required bool   `json:"required"`
		Type     string `json:"type,omitempty"`
	}

	type EndpointInfo struct {
		Method      string      `json:"method"`
		Path        string      `json:"path"`
		Summary     string      `json:"summary,omitempty"`
		Parameters  []ParamInfo `json:"parameters,omitempty"`
		HasBody     bool        `json:"has_request_body"`
		Responses   []string    `json:"documented_responses"`
	}

	type SpecSummary struct {
		Title       string         `json:"title"`
		Version     string         `json:"version"`
		Description string         `json:"description,omitempty"`
		TotalRoutes int            `json:"total_routes"`
		Endpoints   []EndpointInfo `json:"endpoints"`
	}

	summary := SpecSummary{
		Title:       doc.Title,
		Version:     doc.Version,
		TotalRoutes: len(doc.Operations),
	}

	for _, op := range doc.Operations {
		var params []ParamInfo
		for _, p := range op.Parameters {
			pType := "string"
			if p.Schema != nil && p.Schema.Type != nil && len(p.Schema.Type.Slice()) > 0 {
				pType = p.Schema.Type.Slice()[0]
			}
			params = append(params, ParamInfo{
				Name:     p.Name,
				In:       p.In,
				Required: p.Required,
				Type:     pType,
			})
		}

		var responses []string
		for code := range op.Responses {
			responses = append(responses, code)
		}

		summary.Endpoints = append(summary.Endpoints, EndpointInfo{
			Method:      op.Method,
			Path:        op.Path,
			Summary:     op.Summary,
			Parameters:  params,
			HasBody:     op.RequestBody != nil,
			Responses:   responses,
		})
	}

	outJSON, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Failed to format output: %v", err)}},
			IsError: true,
		}, nil
	}

	return ToolCallResult{
		Content: []ToolContent{
			{
				Type: "text",
				Text: string(outJSON),
			},
		},
	}, nil
}
