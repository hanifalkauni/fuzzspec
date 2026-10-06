package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/fuzzspec/fuzzspec/internal/oracle"
)

// JSONReport represents the full structured report artifact.
type JSONReport struct {
	GeneratedAt   time.Time                 `json:"generated_at"`
	TotalTested   int                       `json:"total_tested"`
	Passed        int                       `json:"passed"`
	Failed        int                       `json:"failed"`
	Warnings      int                       `json:"warnings"`
	TotalDuration string                    `json:"total_duration"`
	Results       []oracle.AssertionResult  `json:"results"`
}

// ExportJSON writes a complete diagnostic JSON report to disk with PII sanitization.
func ExportJSON(filePath string, assertions []oracle.AssertionResult, duration time.Duration, passed, failed, warnings int) error {
	report := JSONReport{
		GeneratedAt:   time.Now(),
		TotalTested:   len(assertions),
		Passed:        passed,
		Failed:        failed,
		Warnings:      warnings,
		TotalDuration: duration.String(),
		Results:       assertions,
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON report: %w", err)
	}

	sanitizedData := []byte(SanitizeText(string(data)))

	if err := os.WriteFile(filePath, sanitizedData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON report to '%s': %w", filePath, err)
	}

	return nil
}
