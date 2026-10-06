package fuzzer

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fuzzspec/fuzzspec/internal/generator"
)

// HTTPClient is an instrumented HTTP client for sending fuzzed payloads.
type HTTPClient struct {
	client  *http.Client
	options FuzzerOptions
}

// NewHTTPClient creates an instrumented client with custom timeout and transport settings.
func NewHTTPClient(opts FuzzerOptions) *HTTPClient {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: opts.InsecureSkipVerify,
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   opts.Timeout,
	}

	return &HTTPClient{
		client:  client,
		options: opts,
	}
}

// ExecuteVector executes a single TestVector against the target API.
func (c *HTTPClient) ExecuteVector(ctx context.Context, vector generator.TestVector) ExecutionResult {
	result := ExecutionResult{
		Vector:     vector,
		ExecutedAt: time.Now(),
	}

	// 1. Build Full Target URL (Resolving Path & Query Parameters)
	targetURL, err := c.buildURL(vector)
	if err != nil {
		result.Error = fmt.Sprintf("failed to construct URL: %v", err)
		result.CurlCommand = fmt.Sprintf("# Failed URL build: %v", err)
		return result
	}
	result.RequestURL = targetURL
	result.RequestMethod = vector.Method

	// 2. Prepare Request Body
	var bodyReader io.Reader
	var bodyBytes []byte
	if vector.Body != nil {
		switch b := vector.Body.(type) {
		case string:
			bodyBytes = []byte(b)
		case []byte:
			bodyBytes = b
		default:
			var err error
			bodyBytes, err = json.Marshal(b)
			if err != nil {
				result.Error = fmt.Sprintf("failed to serialize request body: %v", err)
				return result
			}
		}
		bodyReader = bytes.NewReader(bodyBytes)
		result.RequestBody = string(bodyBytes)
	}

	// 3. Construct HTTP Request
	req, err := http.NewRequestWithContext(ctx, vector.Method, targetURL, bodyReader)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create HTTP request: %v", err)
		return result
	}

	// 4. Inject Headers
	req.Header.Set("User-Agent", "FuzzSpec/1.4.0 (API Contract Testing Harness)")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	if vector.Body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Correlation headers for backend observability
	req.Header.Set("X-Fuzz-Request-ID", vector.ID)
	req.Header.Set("X-Fuzz-Scenario", vector.Scenario)

	// Global headers & Authentication
	for k, v := range c.options.GlobalHeaders {
		req.Header.Set(k, v)
	}
	if c.options.AuthToken != "" {
		switch strings.ToLower(c.options.AuthType) {
		case "apikey":
			req.Header.Set("X-API-Key", c.options.AuthToken)
		default: // bearer
			req.Header.Set("Authorization", "Bearer "+c.options.AuthToken)
		}
	}

	// Per-vector custom headers
	for k, v := range vector.Headers {
		req.Header.Set(k, v)
	}

	result.RequestHeaders = req.Header.Clone()

	// 5. Generate Reproducible cURL Command
	result.CurlCommand = c.buildCurlCommand(req, bodyBytes)

	// 6. Execute with Retry (for connection drops)
	var resp *http.Response
	var duration time.Duration
	maxAttempts := c.options.MaxRetries + 1

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		start := time.Now()
		resp, err = c.client.Do(req)
		duration = time.Since(start)

		if err == nil {
			break // Success
		}

		// If context cancelled or timeout expired, don't retry
		if ctx.Err() != nil {
			break
		}

		// Exponential backoff for network drops
		if attempt < maxAttempts {
			time.Sleep(time.Duration(attempt*50) * time.Millisecond)
			// Reset body reader for retry
			if bodyBytes != nil {
				req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			}
		}
	}

	result.Duration = duration

	if err != nil {
		result.Error = err.Error()
		result.IsConnectionErr = true
		return result
	}
	defer resp.Body.Close()

	// 7. Read Response
	result.StatusCode = resp.StatusCode
	result.StatusText = resp.Status
	result.ResponseHeaders = resp.Header.Clone()

	respBodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		result.ResponseBody = fmt.Sprintf("[Error reading response body: %v]", readErr)
	} else {
		result.ResponseBody = string(respBodyBytes)
	}

	return result
}

// buildURL substitutes path parameters and appends query strings.
func (c *HTTPClient) buildURL(vector generator.TestVector) (string, error) {
	baseURL := strings.TrimRight(c.options.BaseURL, "/")
	resolvedPath := vector.Path

	// Substitute {param} in path
	for paramName, paramVal := range vector.PathParams {
		placeholder := fmt.Sprintf("{%s}", paramName)
		escapedVal := url.PathEscape(paramVal)
		resolvedPath = strings.ReplaceAll(resolvedPath, placeholder, escapedVal)
	}

	fullURL := baseURL + "/" + strings.TrimLeft(resolvedPath, "/")

	// Append Query Parameters
	if len(vector.QueryParams) > 0 {
		q := url.Values{}
		for k, v := range vector.QueryParams {
			q.Add(k, v)
		}
		if strings.Contains(fullURL, "?") {
			fullURL += "&" + q.Encode()
		} else {
			fullURL += "?" + q.Encode()
		}
	}

	return fullURL, nil
}

// buildCurlCommand renders an executable cURL reproducer string.
func (c *HTTPClient) buildCurlCommand(req *http.Request, bodyBytes []byte) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("curl -X %s '%s'", req.Method, req.URL.String()))

	for k, vals := range req.Header {
		for _, v := range vals {
			parts = append(parts, fmt.Sprintf("-H '%s: %s'", k, v))
		}
	}

	if len(bodyBytes) > 0 {
		cleanBody := strings.ReplaceAll(string(bodyBytes), "'", "'\\''")
		parts = append(parts, fmt.Sprintf("-d '%s'", cleanBody))
	}

	return strings.Join(parts, " \\\n  ")
}
