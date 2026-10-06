package reporter

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fuzzspec/fuzzspec/internal/oracle"
)

// SARIF 2.1.0 data structures for GitHub Code Scanning
type SARIFReport struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []SARIFRun `json:"runs"`
}

type SARIFRun struct {
	Tool      SARIFTool     `json:"tool"`
	Results   []SARIFResult `json:"results"`
}

type SARIFTool struct {
	Driver SARIFDriver `json:"driver"`
}

type SARIFDriver struct {
	Name            string      `json:"name"`
	Version         string      `json:"version"`
	InformationURI  string      `json:"informationUri"`
	Rules           []SARIFRule `json:"rules"`
}

type SARIFRule struct {
	ID               string               `json:"id"`
	Name             string               `json:"name"`
	ShortDescription SARIFMessage         `json:"shortDescription"`
	FullDescription  SARIFMessage         `json:"fullDescription,omitempty"`
	DefaultConfig    SARIFRuleConfig      `json:"defaultConfiguration"`
}

type SARIFRuleConfig struct {
	Level string `json:"level"` // "error", "warning", "note"
}

type SARIFResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"` // "error", "warning", "note"
	Message   SARIFMessage    `json:"message"`
	Locations []SARIFLocation `json:"locations,omitempty"`
}

type SARIFMessage struct {
	Text string `json:"text"`
}

type SARIFLocation struct {
	PhysicalLocation SARIFPhysicalLocation `json:"physicalLocation"`
}

type SARIFPhysicalLocation struct {
	ArtifactLocation SARIFArtifactLocation `json:"artifactLocation"`
}

type SARIFArtifactLocation struct {
	URI string `json:"uri"`
}

// ExportSARIF generates a SARIF v2.1.0 compliant report file.
func ExportSARIF(filePath string, assertions []oracle.AssertionResult) error {
	rulesMap := make(map[string]SARIFRule)
	var results []SARIFResult

	for _, assertion := range assertions {
		if assertion.Verdict == oracle.VerdictPass {
			continue
		}

		ruleID := "FUZZSPEC-500"
		ruleName := "UnhandledServerCrash"
		level := "error"
		ruleDesc := "Unhandled runtime panic or HTTP 500 error triggered by edge-case input."

		if assertion.Verdict == oracle.VerdictWarn {
			ruleID = "FUZZSPEC-WARN"
			ruleName = "ContractWarning"
			level = "warning"
			ruleDesc = "Potential contract anomaly, slow latency, or unexpected response code."
		}

		findingText := "Contract anomaly detected"
		if len(assertion.Findings) > 0 {
			findingText = assertion.Findings[0].Message
			switch assertion.Findings[0].Category {
			case oracle.CategoryPolyglotStackLeak:
				ruleID = "FUZZSPEC-INFOLEAK"
				ruleName = "InformationLeakStacktrace"
				ruleDesc = "Runtime stack trace or internal database error leak detected in response body."
			case oracle.CategoryContractDrift:
				ruleID = "FUZZSPEC-DRIFT"
				ruleName = "ResponseContractDrift"
				ruleDesc = "Response body does not conform to documented OpenAPI schema."
			case oracle.CategoryLatencySLO:
				ruleID = "FUZZSPEC-LATENCY"
				ruleName = "LatencySLOViolation"
				level = "warning"
				ruleDesc = "Endpoint latency exceeded configured SLO threshold."
			}
		}

		if _, exists := rulesMap[ruleID]; !exists {
			rulesMap[ruleID] = SARIFRule{
				ID:   ruleID,
				Name: ruleName,
				ShortDescription: SARIFMessage{
					Text: ruleDesc,
				},
				DefaultConfig: SARIFRuleConfig{
					Level: level,
				},
			}
		}

		msgText := fmt.Sprintf("[%s %s] %s (Scenario: %s, Status: %d)",
			assertion.Vector.Method,
			assertion.Vector.Path,
			findingText,
			assertion.Vector.Scenario,
			assertion.Execution.StatusCode,
		)

		results = append(results, SARIFResult{
			RuleID: ruleID,
			Level:  level,
			Message: SARIFMessage{
				Text: msgText,
			},
			Locations: []SARIFLocation{
				{
					PhysicalLocation: SARIFPhysicalLocation{
						ArtifactLocation: SARIFArtifactLocation{
							URI: assertion.Vector.Path,
						},
					},
				},
			},
		})
	}

	var rules []SARIFRule
	for _, r := range rulesMap {
		rules = append(rules, r)
	}

	sarif := SARIFReport{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []SARIFRun{
			{
				Tool: SARIFTool{
					Driver: SARIFDriver{
						Name:           "FuzzSpec",
						Version:        "1.0.0",
						InformationURI: "https://github.com/fuzzspec/fuzzspec",
						Rules:          rules,
					},
				},
				Results: results,
			},
		},
	}

	data, err := json.MarshalIndent(sarif, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal SARIF: %w", err)
	}

	sanitized := []byte(SanitizeText(string(data)))
	return os.WriteFile(filePath, sanitized, 0644)
}
