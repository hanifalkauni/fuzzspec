package e2e_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/fuzzspec/fuzzspec/internal/fuzzer"
	"github.com/fuzzspec/fuzzspec/internal/generator"
	"github.com/fuzzspec/fuzzspec/internal/oracle"
	"github.com/fuzzspec/fuzzspec/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_FuzzingPipeline executes the complete end-to-end fuzzing workflow.
func TestE2E_FuzzingPipeline(t *testing.T) {
	// 1. Create a mock backend API that simulates realistic crashes on edge-cases
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Endpoint 1: GET /pets
		if r.URL.Path == "/pets" && r.Method == "GET" {
			limitStr := r.URL.Query().Get("limit")

			// Crash on Int64 overflow attempt!
			if limitStr == "9223372036854775907" {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("panic: runtime error: integer overflow in arithmetic parser"))
				return
			}

			// Handled validation error on negative limit
			if limitStr == "-999999" || limitStr == "0" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "limit must be between 1 and 100"}`))
				return
			}

			// Valid happy path
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"id": 1, "name": "Fido", "price": 10.5}]`))
			return
		}

		// Endpoint 2: GET /pets/{petId}
		if r.Method == "GET" && len(r.URL.Path) > 6 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": 1, "name": "Buddy"}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// 2. Ingest OpenAPI document
	fixturePath, err := filepath.Abs("../../test/fixtures/petstore.yaml")
	require.NoError(t, err)

	p := parser.NewParser()
	parsed, err := p.Parse(context.Background(), fixturePath)
	require.NoError(t, err)
	require.NotEmpty(t, parsed.Operations)

	// 3. Generate Heuristic Test Vectors
	mutator := generator.NewRuleMutator()
	var allVectors []generator.TestVector
	for _, op := range parsed.Operations {
		vectors := mutator.GenerateOperationVectors(op)
		allVectors = append(allVectors, vectors...)
	}
	require.NotEmpty(t, allVectors)

	// 4. Execute Concurrent Worker Pool
	opts := fuzzer.FuzzerOptions{
		BaseURL:     server.URL,
		Concurrency: 5,
		RPS:         100,
		Timeout:     2 * time.Second,
		SafeMode:    true, // Only safe GET methods
	}
	pool := fuzzer.NewWorkerPool(opts)
	execResults := pool.ExecuteVectors(context.Background(), allVectors, nil)
	require.NotEmpty(t, execResults)

	// 5. Evaluate Multi-Layer QA Oracles
	qaOracle := oracle.NewQAOracle(oracle.DefaultOracleOptions())
	var passed, failed int

	for _, res := range execResults {
		assertion := qaOracle.Evaluate(res)
		if assertion.Verdict == oracle.VerdictPass {
			passed++
		} else if assertion.Verdict == oracle.VerdictFail {
			failed++
			assert.True(t, assertion.IsCritical)
			assert.NotEmpty(t, assertion.Execution.CurlCommand)
		}
	}

	assert.Greater(t, passed, 0, "Should have passed valid and safely handled validation requests")
	assert.Greater(t, failed, 0, "Should have detected the simulated 500 integer overflow crash!")
}
