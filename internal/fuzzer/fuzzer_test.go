package fuzzer_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fuzzspec/fuzzspec/internal/fuzzer"
	"github.com/fuzzspec/fuzzspec/internal/generator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkerPool_Execution(t *testing.T) {
	var requestCount int32

	// Setup mock test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)

		// Assert correlation headers
		assert.NotEmpty(t, r.Header.Get("X-Fuzz-Request-ID"))
		assert.NotEmpty(t, r.Header.Get("X-Fuzz-Scenario"))

		if r.URL.Path == "/panic" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("panic: runtime error: nil pointer dereference"))
			return
		}

		if r.URL.Query().Get("invalid") == "true" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(`{"error": "validation failed"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	opts := fuzzer.FuzzerOptions{
		BaseURL:     server.URL,
		Concurrency: 4,
		RPS:         50,
		Timeout:     2 * time.Second,
		MaxRetries:  1,
		SafeMode:    false, // allow all methods
	}

	pool := fuzzer.NewWorkerPool(opts)

	vectors := []generator.TestVector{
		{
			ID:           "vec_01",
			Path:         "/items",
			Method:       "GET",
			Scenario:     "Happy path items",
			MutationType: generator.MutationTypeBaseline,
		},
		{
			ID:           "vec_02",
			Path:         "/items",
			Method:       "GET",
			Scenario:     "Validation error test",
			MutationType: generator.MutationTypeBoundary,
			QueryParams:  map[string]string{"invalid": "true"},
		},
		{
			ID:           "vec_03",
			Path:         "/panic",
			Method:       "GET",
			Scenario:     "Panic crash endpoint",
			MutationType: generator.MutationTypeAdversarial,
		},
	}

	var progressCalled int32
	results := pool.ExecuteVectors(context.Background(), vectors, func(completed, total int, res fuzzer.ExecutionResult) {
		atomic.AddInt32(&progressCalled, 1)
	})

	require.Len(t, results, 3)
	assert.Equal(t, int32(3), atomic.LoadInt32(&requestCount))
	assert.Equal(t, int32(3), atomic.LoadInt32(&progressCalled))

	assert.Equal(t, 200, results[0].StatusCode)
	assert.Equal(t, 422, results[1].StatusCode)
	assert.Equal(t, 500, results[2].StatusCode)
	assert.NotEmpty(t, results[0].CurlCommand)
}

func TestWorkerPool_SafeModeFiltering(t *testing.T) {
	opts := fuzzer.FuzzerOptions{
		BaseURL:     "http://localhost:8080",
		Concurrency: 2,
		RPS:         10,
		Timeout:     1 * time.Second,
		SafeMode:    true, // Only GET/HEAD/OPTIONS
	}

	pool := fuzzer.NewWorkerPool(opts)

	vectors := []generator.TestVector{
		{ID: "v1", Path: "/users", Method: "GET"},
		{ID: "v2", Path: "/users", Method: "POST"},   // Should be filtered
		{ID: "v3", Path: "/users/1", Method: "DELETE"}, // Should be filtered
	}

	// Server is not even needed since non-safe methods are filtered
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	opts.BaseURL = server.URL
	pool = fuzzer.NewWorkerPool(opts)

	results := pool.ExecuteVectors(context.Background(), vectors, nil)
	assert.Len(t, results, 1, "Only 1 GET vector should execute in Safe Mode")
	assert.Equal(t, "v1", results[0].Vector.ID)
}
