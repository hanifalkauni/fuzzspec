package fuzzer

import (
	"net/http"
	"time"

	"github.com/fuzzspec/fuzzspec/internal/generator"
)

// ExecutionResult captures the full HTTP lifecycle of a fuzzed request.
type ExecutionResult struct {
	Vector          generator.TestVector `json:"vector"`
	ExecutedAt      time.Time            `json:"executed_at"`
	Duration        time.Duration        `json:"duration_ms"`
	StatusCode      int                  `json:"status_code"`
	StatusText      string               `json:"status_text"`
	RequestURL      string               `json:"request_url"`
	RequestMethod   string               `json:"request_method"`
	RequestHeaders  http.Header          `json:"request_headers"`
	RequestBody     string               `json:"request_body,omitempty"`
	ResponseHeaders http.Header          `json:"response_headers"`
	ResponseBody    string               `json:"response_body,omitempty"`
	CurlCommand     string               `json:"curl_command"`
	Error           string               `json:"error,omitempty"`
	IsConnectionErr bool                 `json:"is_connection_error"`
}

// FuzzerOptions configures the HTTP execution engine.
type FuzzerOptions struct {
	BaseURL            string            `json:"base_url"`
	Concurrency        int               `json:"concurrency"`
	RPS                int               `json:"rps"`
	Timeout            time.Duration     `json:"timeout"`
	MaxRetries         int               `json:"max_retries"`
	InsecureSkipVerify bool              `json:"insecure_skip_verify"`
	GlobalHeaders      map[string]string `json:"global_headers"`
	AuthToken          string            `json:"auth_token,omitempty"`
	AuthType           string            `json:"auth_type,omitempty"` // "bearer", "apikey", etc.
	SafeMode           bool              `json:"safe_mode"`
}

// DefaultFuzzerOptions returns standard baseline options.
func DefaultFuzzerOptions() FuzzerOptions {
	return FuzzerOptions{
		Concurrency:   10,
		RPS:           20,
		Timeout:       5 * time.Second,
		MaxRetries:    2,
		GlobalHeaders: make(map[string]string),
		SafeMode:      true,
	}
}

// ExecutionStats tracks aggregate execution metrics during a fuzzing run.
type ExecutionStats struct {
	TotalRequests int           `json:"total_requests"`
	Completed     int           `json:"completed"`
	Passed        int           `json:"passed"`
	Failed        int           `json:"failed"`
	Warnings      int           `json:"warnings"`
	TotalDuration time.Duration `json:"total_duration"`
	AvgLatency    time.Duration `json:"avg_latency"`
}
