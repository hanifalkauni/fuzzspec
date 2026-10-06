package replay_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fuzzspec/fuzzspec/internal/fuzzer"
	"github.com/fuzzspec/fuzzspec/internal/generator"
	"github.com/fuzzspec/fuzzspec/internal/oracle"
	"github.com/fuzzspec/fuzzspec/internal/replay"
	"github.com/fuzzspec/fuzzspec/internal/reporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplayEngine_LoadFailingVectors(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fuzzspec-replay-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	reportPath := filepath.Join(tempDir, "report.json")

	mockReport := reporter.JSONReport{
		TotalTested: 2,
		Passed:      1,
		Failed:      1,
		Results: []oracle.AssertionResult{
			{
				Verdict:    oracle.VerdictPass,
				IsCritical: false,
				Vector: generator.TestVector{
					ID:       "vec_pass",
					Path:     "/users",
					Method:   "GET",
					Scenario: "Valid Baseline",
				},
			},
			{
				Verdict:    oracle.VerdictFail,
				IsCritical: true,
				Vector: generator.TestVector{
					ID:       "vec_fail_crash",
					Path:     "/users/{id}",
					Method:   "GET",
					Scenario: "Overflow Integer",
					PathParams: map[string]string{
						"id": "99999999999999999999",
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(mockReport, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(reportPath, data, 0644)
	require.NoError(t, err)

	engine := replay.NewReplayEngine(fuzzer.FuzzerOptions{BaseURL: "http://localhost:8080"})
	failingVectors, err := engine.LoadFailingVectors(reportPath)

	require.NoError(t, err)
	require.Len(t, failingVectors, 1)
	assert.Equal(t, "vec_fail_crash", failingVectors[0].ID)
	assert.Equal(t, "99999999999999999999", failingVectors[0].PathParams["id"])
}

func TestReplayEngine_ReplayFixedBug(t *testing.T) {
	// Mock server that handles overflow properly by returning 400 Bad Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": "Invalid ID format"}`))
	}))
	defer server.Close()

	engine := replay.NewReplayEngine(fuzzer.FuzzerOptions{
		BaseURL:  server.URL,
		Timeout:  2 * time.Second,
		SafeMode: false,
	})

	vectors := []generator.TestVector{
		{
			ID:                   "vec_test_fixed",
			Path:                 "/users/{id}",
			Method:               "GET",
			Scenario:             "Overflow Integer Test",
			PathParams:           map[string]string{"id": "99999999999999999999"},
			ExpectedStatusFamily: "4xx",
		},
	}

	summary := engine.Replay(context.Background(), vectors)

	assert.Equal(t, 1, summary.TotalReplayed)
	assert.Equal(t, 1, summary.Resolved)
	assert.Equal(t, 0, summary.StillFailing)
	require.Len(t, summary.Results, 1)
	assert.Equal(t, oracle.VerdictPass, summary.Results[0].Verdict)
}

func TestReplayEngine_ReplayStillFailingBug(t *testing.T) {
	// Mock server that is still crashing with HTTP 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "panic: runtime error: integer overflow"}`))
	}))
	defer server.Close()

	engine := replay.NewReplayEngine(fuzzer.FuzzerOptions{
		BaseURL:  server.URL,
		Timeout:  2 * time.Second,
		SafeMode: false,
	})

	vectors := []generator.TestVector{
		{
			ID:                   "vec_test_broken",
			Path:                 "/users/{id}",
			Method:               "GET",
			Scenario:             "Crash Trigger",
			PathParams:           map[string]string{"id": "99999999999999999999"},
			ExpectedStatusFamily: "4xx",
		},
	}

	summary := engine.Replay(context.Background(), vectors)

	assert.Equal(t, 1, summary.TotalReplayed)
	assert.Equal(t, 0, summary.Resolved)
	assert.Equal(t, 1, summary.StillFailing)
	require.Len(t, summary.Results, 1)
	assert.Equal(t, oracle.VerdictFail, summary.Results[0].Verdict)
}
