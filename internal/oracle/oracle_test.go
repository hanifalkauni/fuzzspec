package oracle_test

import (
	"testing"
	"time"

	"github.com/fuzzspec/fuzzspec/internal/fuzzer"
	"github.com/fuzzspec/fuzzspec/internal/generator"
	"github.com/fuzzspec/fuzzspec/internal/oracle"
	"github.com/stretchr/testify/assert"
)

func TestQAOracle_Evaluate500Crash(t *testing.T) {
	qa := oracle.NewQAOracle(oracle.DefaultOracleOptions())

	res := fuzzer.ExecutionResult{
		Vector:       generator.TestVector{ID: "vec_01", Path: "/users", Method: "POST"},
		StatusCode:   500,
		StatusText:   "Internal Server Error",
		ResponseBody: "panic: runtime error: invalid memory address or nil pointer dereference",
		Duration:     100 * time.Millisecond,
	}

	assertion := qa.Evaluate(res)

	assert.Equal(t, oracle.VerdictFail, assertion.Verdict)
	assert.True(t, assertion.IsCritical)
	assert.NotEmpty(t, assertion.Findings)

	// Should have both Crash5xx and PolyglotStackLeak findings
	var found5xx, foundLeak bool
	for _, f := range assertion.Findings {
		if f.Category == oracle.CategoryCrash5xx {
			found5xx = true
		}
		if f.Category == oracle.CategoryPolyglotStackLeak {
			foundLeak = true
		}
	}
	assert.True(t, found5xx, "Must detect 500 status code")
	assert.True(t, foundLeak, "Must detect Go panic stack trace")
}

func TestQAOracle_EvaluateHandled422Pass(t *testing.T) {
	qa := oracle.NewQAOracle(oracle.DefaultOracleOptions())

	res := fuzzer.ExecutionResult{
		Vector:       generator.TestVector{ID: "vec_02", Path: "/pets", Method: "POST", MutationType: generator.MutationTypeMissingRequired},
		StatusCode:   422,
		StatusText:   "Unprocessable Entity",
		ResponseBody: `{"errors": [{"field": "name", "message": "name is required"}]}`,
		Duration:     50 * time.Millisecond,
	}

	assertion := qa.Evaluate(res)

	assert.Equal(t, oracle.VerdictPass, assertion.Verdict)
	assert.False(t, assertion.IsCritical)
	assert.Empty(t, assertion.Findings)
}

func TestQAOracle_EvaluatePolyglotRuntimes(t *testing.T) {
	qa := oracle.NewQAOracle(oracle.DefaultOracleOptions())

	cases := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "Python FastAPI/Django",
			body:     "Traceback (most recent call last):\n  File 'app.py', line 42, in create\nZeroDivisionError: division by zero",
			expected: "Python",
		},
		{
			name:     "Node.js Express",
			body:     "TypeError: Cannot read properties of undefined (reading 'id')\n    at Object.<anonymous> (/app/dist/server.js:55:12)",
			expected: "NodeJS/TypeScript",
		},
		{
			name:     "Java Spring Boot",
			body:     "java.lang.NullPointerException: Cannot invoke 'String.length()' because 'name' is null\n\tat com.example.service.PetService.save",
			expected: "Java/Kotlin",
		},
		{
			name:     "PHP Laravel",
			body:     "Fatal error: Uncaught TypeError: Return value must be of type string, null returned in /var/www/app/Http/Controllers/OrderController.php:88",
			expected: "PHP/Laravel",
		},
		{
			name:     "Rust Axum",
			body:     "thread 'tokio-runtime-worker' panicked at 'called `Result::unwrap()` on an `Err` value: ParseIntError'",
			expected: "Rust",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := fuzzer.ExecutionResult{
				Vector:       generator.TestVector{ID: "vec_test", Path: "/test", Method: "POST"},
				StatusCode:   500,
				ResponseBody: tc.body,
				Duration:     50 * time.Millisecond,
			}
			assertion := qa.Evaluate(res)
			assert.Equal(t, oracle.VerdictFail, assertion.Verdict)

			var detectedLang string
			for _, f := range assertion.Findings {
				if f.Category == oracle.CategoryPolyglotStackLeak {
					detectedLang = tc.expected
					break
				}
			}
			assert.NotEmpty(t, detectedLang, "Should detect stack trace for "+tc.name)
		})
	}
}

func TestQAOracle_LatencyWarning(t *testing.T) {
	opts := oracle.DefaultOracleOptions()
	opts.LatencyThresholdMs = 500
	qa := oracle.NewQAOracle(opts)

	res := fuzzer.ExecutionResult{
		Vector:       generator.TestVector{ID: "vec_slow", Path: "/search", Method: "GET"},
		StatusCode:   200,
		ResponseBody: `{"results": []}`,
		Duration:     800 * time.Millisecond, // Exceeds 500ms
	}

	assertion := qa.Evaluate(res)
	assert.Equal(t, oracle.VerdictWarn, assertion.Verdict)
	assert.Len(t, assertion.Findings, 1)
	assert.Equal(t, oracle.CategoryLatencySLO, assertion.Findings[0].Category)
}
