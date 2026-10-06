package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fuzzspec/fuzzspec/internal/fuzzer"
	"github.com/fuzzspec/fuzzspec/internal/generator"
	"github.com/fuzzspec/fuzzspec/internal/oracle"
	"github.com/fuzzspec/fuzzspec/internal/parser"
)

func getFuzzEndpointTool() Tool {
	return Tool{
		Name:        "fuzz_endpoint",
		Description: "Executes boundary, adversarial, and AI-generated fuzzing against target API endpoints. Returns contract assertion results, detected 500 panics/anomalies, and exact cURL reproducers.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]PropertyDoc{
				"target_url": {
					Type:        "string",
					Description: "Target base URL (e.g. http://localhost:8080)",
				},
				"spec_path": {
					Type:        "string",
					Description: "Path to OpenAPI YAML/JSON file or remote URL (if omitted, will auto-discover from target_url)",
				},
				"path": {
					Type:        "string",
					Description: "Optional: Restrict fuzzing to a specific endpoint path (e.g. /v1/orders or /users/{id})",
				},
				"method": {
					Type:        "string",
					Description: "Optional: Restrict fuzzing to a specific HTTP method (GET, POST, PUT, DELETE, PATCH)",
				},
				"concurrency": {
					Type:        "integer",
					Description: "Concurrent worker count (default: 5)",
				},
				"rps": {
					Type:        "integer",
					Description: "Rate limit requests per second (default: 20)",
				},
				"safe_mode": {
					Type:        "boolean",
					Description: "If true, only executes safe methods (GET, HEAD, OPTIONS). Set false to test POST/PUT/DELETE mutations.",
				},
				"ai_enabled": {
					Type:        "boolean",
					Description: "Whether to enrich test suite with AI semantic edge cases (requires GEMINI_API_KEY, OPENAI_API_KEY, or ANTHROPIC_API_KEY)",
				},
			},
			Required: []string{"target_url"},
		},
	}
}

func handleFuzzEndpoint(ctx context.Context, args map[string]interface{}) (ToolCallResult, error) {
	targetURL, _ := args["target_url"].(string)
	specPath, _ := args["spec_path"].(string)
	filterPath, _ := args["path"].(string)
	filterMethod, _ := args["method"].(string)

	concurrency := 5
	if c, ok := args["concurrency"].(float64); ok && c > 0 {
		concurrency = int(c)
	}

	rps := 20
	if r, ok := args["rps"].(float64); ok && r > 0 {
		rps = int(r)
	}

	safeMode := true
	if sm, ok := args["safe_mode"].(bool); ok {
		safeMode = sm
	}

	aiEnabled := false
	if ai, ok := args["ai_enabled"].(bool); ok {
		aiEnabled = ai
	}

	if targetURL == "" {
		return ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: "Missing required argument 'target_url'."}},
			IsError: true,
		}, nil
	}

	// Auto-discover if spec is missing
	if specPath == "" {
		discoverer := parser.NewAutoDiscoverer()
		discovered, err := discoverer.Discover(ctx, targetURL, nil)
		if err != nil {
			return ToolCallResult{
				Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Auto-discovery failed: %v (please provide spec_path)", err)}},
				IsError: true,
			}, nil
		}
		specPath = discovered
	}

	p := parser.NewParser()
	doc, err := p.Parse(ctx, specPath)
	if err != nil {
		return ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Failed to parse spec: %v", err)}},
			IsError: true,
		}, nil
	}

	// Filter operations if specified
	var targetOps []parser.EndpointOperation
	for _, op := range doc.Operations {
		if filterPath != "" && !strings.EqualFold(op.Path, filterPath) {
			continue
		}
		if filterMethod != "" && !strings.EqualFold(op.Method, filterMethod) {
			continue
		}
		targetOps = append(targetOps, op)
	}

	if len(targetOps) == 0 {
		return ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("No matching endpoints found in spec for path='%s', method='%s'", filterPath, filterMethod)}},
			IsError: true,
		}, nil
	}

	// Generate Test Vectors
	mutator := generator.NewRuleMutator()
	var allVectors []generator.TestVector
	for _, op := range targetOps {
		vectors := mutator.GenerateOperationVectors(op)
		allVectors = append(allVectors, vectors...)
	}

	if aiEnabled {
		aiGen := generator.NewAIGenerator("gemini", "gemini-1.5-flash", ".fuzzspec-cache", true)
		for _, op := range targetOps {
			aiVecs, err := aiGen.GenerateAIVectors(ctx, op)
			if err == nil && len(aiVecs) > 0 {
				allVectors = append(allVectors, aiVecs...)
			}
		}
	}

	// Execute Fuzzing Pool
	pool := fuzzer.NewWorkerPool(fuzzer.FuzzerOptions{
		BaseURL:     targetURL,
		Concurrency: concurrency,
		RPS:         rps,
		Timeout:     5 * time.Second,
		SafeMode:    safeMode,
	})

	execResults := pool.ExecuteVectors(ctx, allVectors, nil)

	// Evaluate QA Oracles
	qaOracle := oracle.NewQAOracle(oracle.DefaultOracleOptions())
	var passed, failed, warnings int
	var anomalies []map[string]interface{}

	for _, res := range execResults {
		assertion := qaOracle.Evaluate(res)
		if assertion.Verdict == oracle.VerdictPass {
			passed++
		} else if assertion.Verdict == oracle.VerdictFail {
			failed++
			findingSummary := "Unhandled Anomaly"
			if len(assertion.Findings) > 0 {
				findingSummary = assertion.Findings[0].Message
			}

			remediation := "Validate input before processing and return standard 4xx error."
			if assertion.Execution.StatusCode >= 500 {
				remediation = "Add defensive nil/bounds checking or catch unhandled runtime panics."
			}

			anomalies = append(anomalies, map[string]interface{}{
				"id":           assertion.Vector.ID,
				"method":       assertion.Vector.Method,
				"path":         assertion.Vector.Path,
				"scenario":     assertion.Vector.Scenario,
				"status_code":  assertion.Execution.StatusCode,
				"finding":      findingSummary,
				"is_critical":  assertion.IsCritical,
				"curl":         assertion.Execution.CurlCommand,
				"remediation":  remediation,
			})
		} else {
			warnings++
		}
	}

	summary := map[string]interface{}{
		"target":            targetURL,
		"total_executed":    len(execResults),
		"passed":            passed,
		"failed":            failed,
		"warnings":          warnings,
		"quality_gate_pass": failed == 0,
		"anomalies":         anomalies,
	}

	outJSON, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Failed to format fuzz results: %v", err)}},
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
