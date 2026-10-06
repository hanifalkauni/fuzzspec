package oracle

import (
	"fmt"
	"time"

	"github.com/fuzzspec/fuzzspec/internal/config"
	"github.com/fuzzspec/fuzzspec/internal/fuzzer"
	"github.com/fuzzspec/fuzzspec/internal/generator"
)

// OracleOptions configures the assertion evaluation thresholds.
type OracleOptions struct {
	FailOn5xx          bool
	FailOnSchemaDrift   bool
	ScanInfoLeak       bool
	LatencyThresholdMs int
	CustomLanguages    []config.CustomLang
}

// DefaultOracleOptions returns standard oracle options.
func DefaultOracleOptions() OracleOptions {
	return OracleOptions{
		FailOn5xx:          true,
		FailOnSchemaDrift:   true,
		ScanInfoLeak:       true,
		LatencyThresholdMs: 3000,
	}
}

// QAOracle evaluates executed HTTP results across multi-layered test oracles.
type QAOracle struct {
	scanner *PolyglotScanner
	options OracleOptions
}

// NewQAOracle creates a new assertion engine.
func NewQAOracle(opts OracleOptions) *QAOracle {
	return &QAOracle{
		scanner: NewPolyglotScanner(opts.CustomLanguages),
		options: opts,
	}
}

// Evaluate performs multi-layer assertion on a single HTTP execution result.
func (o *QAOracle) Evaluate(res fuzzer.ExecutionResult) AssertionResult {
	result := AssertionResult{
		Vector:    res.Vector,
		Execution: res,
		Verdict:   VerdictPass,
	}

	// 1. Connection / Network Drops
	if res.IsConnectionErr {
		result.Verdict = VerdictFail
		result.IsCritical = true
		result.Findings = append(result.Findings, Finding{
			Severity: SeverityCritical,
			Category: CategoryConnectionDrop,
			Message:  "Server dropped TCP connection or crashed during request execution",
			Evidence: res.Error,
		})
		result.Summary = "Critical connection drop / server crash"
		return result
	}

	// 2. HTTP 5xx Server Exception Oracle
	if res.StatusCode >= 500 && res.StatusCode <= 599 {
		if o.options.FailOn5xx {
			result.Verdict = VerdictFail
			result.IsCritical = true
			result.Findings = append(result.Findings, Finding{
				Severity: SeverityCritical,
				Category: CategoryCrash5xx,
				Message:  fmt.Sprintf("Server threw unhandled %d %s", res.StatusCode, res.StatusText),
				Evidence: truncateString(res.ResponseBody, 300),
			})
		}
	}

	// 3. Polyglot Stack Trace & Panic Leak Oracle
	if o.options.ScanInfoLeak {
		hasLeak, lang, msg := o.scanner.ScanResponse(res.ResponseBody)
		if hasLeak {
			result.Verdict = VerdictFail
			result.IsCritical = true
			result.Findings = append(result.Findings, Finding{
				Severity: SeverityCritical,
				Category: CategoryPolyglotStackLeak,
				Message:  fmt.Sprintf("Information disclosure: %s stack trace leaked in response body", lang),
				Evidence: msg,
			})
		}
	}

	// 4. Latency / SLO Violation Oracle
	if o.options.LatencyThresholdMs > 0 {
		threshold := time.Duration(o.options.LatencyThresholdMs) * time.Millisecond
		if res.Duration > threshold {
			if result.Verdict == VerdictPass {
				result.Verdict = VerdictWarn
			}
			result.Findings = append(result.Findings, Finding{
				Severity: SeverityMedium,
				Category: CategoryLatencySLO,
				Message:  fmt.Sprintf("Latency threshold exceeded: response took %v (limit: %v)", res.Duration, threshold),
				Evidence: fmt.Sprintf("Duration: %d ms", res.Duration.Milliseconds()),
			})
		}
	}

	// 5. Baseline Valid Check: Baseline Happy Path should return 2xx
	if res.Vector.MutationType == generator.MutationTypeBaseline {
		if res.StatusCode >= 400 && res.StatusCode <= 499 {
			if result.Verdict == VerdictPass {
				result.Verdict = VerdictWarn
			}
			result.Findings = append(result.Findings, Finding{
				Severity: SeverityLow,
				Category: CategoryContractDrift,
				Message:  fmt.Sprintf("Baseline valid request was rejected with client error %d %s", res.StatusCode, res.StatusText),
				Evidence: truncateString(res.ResponseBody, 200),
			})
		}
	}

	// Set overall human-readable summary
	if len(result.Findings) == 0 {
		result.Summary = fmt.Sprintf("HTTP %d %s handled safely", res.StatusCode, res.StatusText)
	} else {
		result.Summary = result.Findings[0].Message
	}

	return result
}

func truncateString(str string, maxLen int) string {
	if len(str) <= maxLen {
		return str
	}
	return str[:maxLen] + "... [truncated]"
}
