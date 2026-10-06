package oracle

import (
	"github.com/fuzzspec/fuzzspec/internal/fuzzer"
	"github.com/fuzzspec/fuzzspec/internal/generator"
)

// Verdict represents the overall test outcome.
type Verdict string

const (
	VerdictPass Verdict = "PASS"
	VerdictFail Verdict = "FAIL"
	VerdictWarn Verdict = "WARN"
)

// Severity indicates the risk level of an assertion finding.
type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityMedium   Severity = "MEDIUM"
	SeverityLow      Severity = "LOW"
)

// FindingCategory classifies the type of defect detected.
type FindingCategory string

const (
	CategoryCrash5xx          FindingCategory = "SERVER_CRASH_5XX"
	CategoryConnectionDrop    FindingCategory = "CONNECTION_DROP"
	CategoryPolyglotStackLeak FindingCategory = "POLYGLOT_STACKTRACE_LEAK"
	CategoryContractDrift     FindingCategory = "RESPONSE_CONTRACT_DRIFT"
	CategoryLatencySLO        FindingCategory = "LATENCY_SLO_VIOLATION"
	CategoryImproper200       FindingCategory = "IMPROPER_200_ON_ADVERSARIAL"
)

// Finding describes a single concrete defect or anomaly discovered in a response.
type Finding struct {
	Severity Severity        `json:"severity"`
	Category FindingCategory `json:"category"`
	Message  string          `json:"message"`
	Evidence string          `json:"evidence,omitempty"`
}

// AssertionResult represents the complete multi-layer QA evaluation of a test case.
type AssertionResult struct {
	Vector       generator.TestVector   `json:"vector"`
	Execution    fuzzer.ExecutionResult `json:"execution"`
	Verdict      Verdict                `json:"verdict"`
	Findings     []Finding              `json:"findings,omitempty"`
	Summary      string                 `json:"summary"`
	IsCritical   bool                   `json:"is_critical"`
}
