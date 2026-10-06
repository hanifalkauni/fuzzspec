package fuzzer

import (
	"context"
	"sync"

	"github.com/fuzzspec/fuzzspec/internal/generator"
)

// ProgressCallback is invoked as each test vector finishes execution.
type ProgressCallback func(completed, total int, result ExecutionResult)

// WorkerPool coordinates parallel HTTP fuzzing execution.
type WorkerPool struct {
	client      *HTTPClient
	rateLimiter *RateLimiter
	options     FuzzerOptions
}

// NewWorkerPool creates a new concurrency worker pool.
func NewWorkerPool(opts FuzzerOptions) *WorkerPool {
	return &WorkerPool{
		client:      NewHTTPClient(opts),
		rateLimiter: NewRateLimiter(opts.RPS),
		options:     opts,
	}
}

// ExecuteVectors executes a batch of test vectors concurrently.
func (p *WorkerPool) ExecuteVectors(ctx context.Context, vectors []generator.TestVector, callback ProgressCallback) []ExecutionResult {
	total := len(vectors)
	if total == 0 {
		return nil
	}

	results := make([]ExecutionResult, total)
	jobs := make(chan struct {
		index  int
		vector generator.TestVector
	}, total)

	// Filter vectors if SafeMode is active (only GET, HEAD, OPTIONS)
	filteredVectors := make([]generator.TestVector, 0, total)
	for _, v := range vectors {
		if p.options.SafeMode && !isSafeMethod(v.Method) {
			continue // Skip mutating methods in safe mode
		}
		filteredVectors = append(filteredVectors, v)
	}

	total = len(filteredVectors)
	if total == 0 {
		return nil
	}

	// Queue up jobs
	for i, v := range filteredVectors {
		jobs <- struct {
			index  int
			vector generator.TestVector
		}{index: i, vector: v}
	}
	close(jobs)

	concurrency := p.options.Concurrency
	if concurrency <= 0 {
		concurrency = 10
	}
	if concurrency > total {
		concurrency = total
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	completedCount := 0

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				// Honor context cancellation
				if ctx.Err() != nil {
					return
				}

				// Rate limiting wait
				if err := p.rateLimiter.Wait(ctx); err != nil {
					return
				}

				// Execute HTTP request
				res := p.client.ExecuteVector(ctx, job.vector)

				mu.Lock()
				results[job.index] = res
				completedCount++
				currentCompleted := completedCount
				mu.Unlock()

				if callback != nil {
					callback(currentCompleted, total, res)
				}
			}
		}()
	}

	wg.Wait()
	return results[:completedCount]
}

func isSafeMethod(method string) bool {
	switch method {
	case "GET", "HEAD", "OPTIONS":
		return true
	default:
		return false
	}
}
