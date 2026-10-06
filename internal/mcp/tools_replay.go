package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fuzzspec/fuzzspec/internal/fuzzer"
	"github.com/fuzzspec/fuzzspec/internal/generator"
	"github.com/fuzzspec/fuzzspec/internal/oracle"
	"github.com/fuzzspec/fuzzspec/internal/replay"
)

func getReplayAnomalyTool() Tool {
	return Tool{
		Name:        "replay_anomaly",
		Description: "Re-executes previously failing anomalies against the target server to verify whether a code fix succeeded. Requires 0 AI tokens.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]PropertyDoc{
				"target_url": {
					Type:        "string",
					Description: "Target base URL (e.g. http://localhost:8080)",
				},
				"report_file": {
					Type:        "string",
					Description: "Optional: Path to diagnostic JSON report containing failed test cases",
				},
				"vector": {
					Type:        "object",
					Description: "Optional: Single TestVector object to replay directly without report file",
				},
			},
			Required: []string{"target_url"},
		},
	}
}

func handleReplayAnomaly(ctx context.Context, args map[string]interface{}) (ToolCallResult, error) {
	targetURL, _ := args["target_url"].(string)
	reportFile, _ := args["report_file"].(string)

	if targetURL == "" {
		return ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: "Missing required argument 'target_url'."}},
			IsError: true,
		}, nil
	}

	engine := replay.NewReplayEngine(fuzzer.FuzzerOptions{
		BaseURL:  targetURL,
		Timeout:  5 * time.Second,
		SafeMode: false,
	})

	var vectors []generator.TestVector

	if reportFile != "" {
		loaded, err := engine.LoadFailingVectors(reportFile)
		if err != nil {
			return ToolCallResult{
				Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Failed to load vectors from report file: %v", err)}},
				IsError: true,
			}, nil
		}
		vectors = append(vectors, loaded...)
	} else if vecRaw, ok := args["vector"]; ok {
		vecBytes, err := json.Marshal(vecRaw)
		if err != nil {
			return ToolCallResult{
				Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Invalid vector object: %v", err)}},
				IsError: true,
			}, nil
		}
		var vec generator.TestVector
		if err := json.Unmarshal(vecBytes, &vec); err != nil {
			return ToolCallResult{
				Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Failed to parse vector object: %v", err)}},
				IsError: true,
			}, nil
		}
		vectors = append(vectors, vec)
	} else {
		return ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: "Either 'report_file' or 'vector' must be specified for replay."}},
			IsError: true,
		}, nil
	}

	if len(vectors) == 0 {
		return ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: "No test vectors to replay."}},
		}, nil
	}

	summary := engine.Replay(ctx, vectors)

	type ReplayItemResult struct {
		ID         string `json:"id"`
		Method     string `json:"method"`
		Path       string `json:"path"`
		Scenario   string `json:"scenario"`
		StatusCode int    `json:"status_code"`
		Verdict    string `json:"verdict"` // "RESOLVED" or "STILL_FAILING"
		Finding    string `json:"finding,omitempty"`
	}

	var items []ReplayItemResult
	for _, res := range summary.Results {
		verdictStr := "RESOLVED"
		if res.Verdict != oracle.VerdictPass {
			verdictStr = "STILL_FAILING"
		}
		findingStr := ""
		if len(res.Findings) > 0 {
			findingStr = res.Findings[0].Message
		}

		items = append(items, ReplayItemResult{
			ID:         res.Vector.ID,
			Method:     res.Vector.Method,
			Path:       res.Vector.Path,
			Scenario:   res.Vector.Scenario,
			StatusCode: res.Execution.StatusCode,
			Verdict:    verdictStr,
			Finding:    findingStr,
		})
	}

	resultObj := map[string]interface{}{
		"target":         targetURL,
		"total_replayed": summary.TotalReplayed,
		"resolved":       summary.Resolved,
		"still_failing":  summary.StillFailing,
		"all_resolved":   summary.StillFailing == 0,
		"results":        items,
	}

	outJSON, err := json.MarshalIndent(resultObj, "", "  ")
	if err != nil {
		return ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Failed to format replay results: %v", err)}},
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
