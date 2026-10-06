package replay

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/fuzzspec/fuzzspec/internal/fuzzer"
	"github.com/fuzzspec/fuzzspec/internal/generator"
	"github.com/fuzzspec/fuzzspec/internal/oracle"
	"github.com/fuzzspec/fuzzspec/internal/reporter"
)

// ReplaySummary contains the results of a replay session.
type ReplaySummary struct {
	TotalReplayed int                       `json:"total_replayed"`
	Resolved      int                       `json:"resolved"`
	StillFailing  int                       `json:"still_failing"`
	Results       []oracle.AssertionResult  `json:"results"`
}

// ReplayEngine re-executes previously failing test cases to verify bug fixes.
type ReplayEngine struct {
	client  *fuzzer.HTTPClient
	oracle  *oracle.QAOracle
	options fuzzer.FuzzerOptions
}

// NewReplayEngine creates a new deterministic ReplayEngine.
func NewReplayEngine(opts fuzzer.FuzzerOptions) *ReplayEngine {
	return &ReplayEngine{
		client:  fuzzer.NewHTTPClient(opts),
		oracle:  oracle.NewQAOracle(oracle.DefaultOracleOptions()),
		options: opts,
	}
}

// LoadFailingVectors extracts failed test vectors from a diagnostic JSON report file.
func (e *ReplayEngine) LoadFailingVectors(reportFilePath string) ([]generator.TestVector, error) {
	data, err := os.ReadFile(reportFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read report file '%s': %w", reportFilePath, err)
	}

	var report reporter.JSONReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to parse JSON report: %w", err)
	}

	var failingVectors []generator.TestVector
	for _, res := range report.Results {
		if res.Verdict == oracle.VerdictFail || res.IsCritical {
			failingVectors = append(failingVectors, res.Vector)
		}
	}

	return failingVectors, nil
}

// Replay executes the failing vectors against the new target and evaluates resolution status.
func (e *ReplayEngine) Replay(ctx context.Context, vectors []generator.TestVector) ReplaySummary {
	summary := ReplaySummary{
		TotalReplayed: len(vectors),
	}

	for _, vec := range vectors {
		execRes := e.client.ExecuteVector(ctx, vec)
		assertion := e.oracle.Evaluate(execRes)

		summary.Results = append(summary.Results, assertion)

		if assertion.Verdict == oracle.VerdictPass {
			summary.Resolved++
		} else {
			summary.StillFailing++
		}
	}

	return summary
}
