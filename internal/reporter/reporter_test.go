package reporter_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fuzzspec/fuzzspec/internal/fuzzer"
	"github.com/fuzzspec/fuzzspec/internal/generator"
	"github.com/fuzzspec/fuzzspec/internal/oracle"
	"github.com/fuzzspec/fuzzspec/internal/reporter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeText(t *testing.T) {
	raw := `Authorization: Bearer super_secret_jwt_token_12345
api_key: "abcdef123456789"
"password": "my_secure_password"`

	sanitized := reporter.SanitizeText(raw)

	assert.NotContains(t, sanitized, "super_secret_jwt_token_12345")
	assert.NotContains(t, sanitized, "my_secure_password")
	assert.Contains(t, sanitized, "[REDACTED_TOKEN]")
	assert.Contains(t, sanitized, "[REDACTED_PASSWORD]")
}

func TestTerminalReporter_RenderSummary(t *testing.T) {
	buf := new(bytes.Buffer)
	r := reporter.NewTerminalReporter(buf)

	assertions := []oracle.AssertionResult{
		{
			Vector: generator.TestVector{Method: "GET", Path: "/pets", Scenario: "Happy path"},
			Execution: fuzzer.ExecutionResult{
				StatusCode:  200,
				Duration:    45 * time.Millisecond,
				CurlCommand: "curl -X GET http://localhost/pets",
			},
			Verdict: oracle.VerdictPass,
		},
		{
			Vector: generator.TestVector{Method: "POST", Path: "/pets", Scenario: "Null name injection"},
			Execution: fuzzer.ExecutionResult{
				StatusCode:  500,
				Duration:    120 * time.Millisecond,
				CurlCommand: "curl -X POST http://localhost/pets -d '{\"name\": null}'",
			},
			Verdict:    oracle.VerdictFail,
			IsCritical: true,
			Findings: []oracle.Finding{
				{Category: oracle.CategoryCrash5xx, Message: "Server threw 500 Internal Server Error"},
			},
		},
	}

	passed, failed, warnings := r.RenderSummary(assertions, 500*time.Millisecond)

	assert.Equal(t, 1, passed)
	assert.Equal(t, 1, failed)
	assert.Equal(t, 0, warnings)

	output := buf.String()
	assert.Contains(t, output, "FUZZSPEC EXECUTION RESULTS")
	assert.Contains(t, output, "TOTAL: 2")
	assert.Contains(t, output, "DETECTED ANOMALIES")
	assert.Contains(t, output, "curl -X POST")
}

func TestExportJSON(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "report.json")

	assertions := []oracle.AssertionResult{
		{
			Vector:  generator.TestVector{Method: "GET", Path: "/health"},
			Verdict: oracle.VerdictPass,
		},
	}

	err := reporter.ExportJSON(outPath, assertions, 200*time.Millisecond, 1, 0, 0)
	require.NoError(t, err)

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"total_tested": 1`)
	assert.Contains(t, string(data), `"passed": 1`)
}

func TestExportSARIF(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "results.sarif")

	assertions := []oracle.AssertionResult{
		{
			Vector: generator.TestVector{Method: "POST", Path: "/orders", Scenario: "Integer Overflow"},
			Execution: fuzzer.ExecutionResult{
				StatusCode:  500,
				CurlCommand: "curl -X POST http://localhost/orders -d '{\"id\": 99999999999999999999}'",
			},
			Verdict:    oracle.VerdictFail,
			IsCritical: true,
			Findings: []oracle.Finding{
				{Category: oracle.CategoryPolyglotStackLeak, Message: "Unhandled Panic / Stack Trace Leak"},
			},
		},
	}

	err := reporter.ExportSARIF(outPath, assertions)
	require.NoError(t, err)

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "2.1.0")
	assert.Contains(t, content, "FUZZSPEC-INFOLEAK")
	assert.Contains(t, content, "/orders")
}

func TestExportJUnit(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "results.xml")

	assertions := []oracle.AssertionResult{
		{
			Vector: generator.TestVector{Method: "GET", Path: "/users", Scenario: "Valid List"},
			Execution: fuzzer.ExecutionResult{
				StatusCode: 200,
				Duration:   50 * time.Millisecond,
			},
			Verdict: oracle.VerdictPass,
		},
		{
			Vector: generator.TestVector{Method: "GET", Path: "/users/999", Scenario: "Overflow ID"},
			Execution: fuzzer.ExecutionResult{
				StatusCode:  500,
				StatusText:  "Internal Server Error",
				Duration:    150 * time.Millisecond,
				CurlCommand: "curl -X GET http://localhost/users/999",
			},
			Verdict:    oracle.VerdictFail,
			IsCritical: true,
			Findings: []oracle.Finding{
				{Message: "Server crashed on overflow"},
			},
		},
	}

	err := reporter.ExportJUnit(outPath, assertions, 300*time.Millisecond)
	require.NoError(t, err)

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "<testsuites")
	assert.Contains(t, content, `failures="1"`)
	assert.Contains(t, content, `<failure message="Server crashed on overflow"`)
}

func TestExportMarkdown(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "results.md")

	assertions := []oracle.AssertionResult{
		{
			Vector: generator.TestVector{Method: "GET", Path: "/items", Scenario: "Happy path"},
			Execution: fuzzer.ExecutionResult{
				StatusCode: 200,
			},
			Verdict: oracle.VerdictPass,
		},
		{
			Vector: generator.TestVector{
				Method:       "POST",
				Path:         "/items",
				Scenario:     "Malformed JSON",
				MutationType: generator.MutationTypeBoundary,
			},
			Execution: fuzzer.ExecutionResult{
				StatusCode:   500,
				CurlCommand:  "curl -X POST http://localhost/items -d '{bad}'",
				ResponseBody: `{"error": "panic"}`,
				Duration:     80 * time.Millisecond,
			},
			Verdict:    oracle.VerdictFail,
			IsCritical: true,
			Findings: []oracle.Finding{
				{Message: "Unhandled JSON parse exception"},
			},
		},
	}

	err := reporter.ExportMarkdown(outPath, assertions, 400*time.Millisecond, 1, 1, 0)
	require.NoError(t, err)

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "FuzzSpec API Contract & Boundary Fuzzing Report")
	assert.Contains(t, content, "FAILED (Critical Anomaly Detected)")
	assert.Contains(t, content, "curl -X POST http://localhost/items -d '{bad}'")
	assert.Contains(t, content, "Unhandled JSON parse exception")
}

