package reporter

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fuzzspec/fuzzspec/internal/oracle"
)

// ExportMarkdown writes a GitHub Actions / GitLab CI friendly Markdown report.
func ExportMarkdown(filePath string, assertions []oracle.AssertionResult, duration time.Duration, passed, failed, warnings int) error {
	var sb strings.Builder

	statusBadge := "✅ **PASSED**"
	statusColor := "green"
	if failed > 0 {
		statusBadge = "❌ **FAILED (Critical Anomaly Detected)**"
		statusColor = "red"
	}

	sb.WriteString("# 🚀 FuzzSpec API Contract & Boundary Fuzzing Report\n\n")
	sb.WriteString(fmt.Sprintf("> **Quality Gate Status:** %s\n\n", statusBadge))

	// Summary Table
	sb.WriteString("### 📊 Test Suite Summary\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("| :--- | :--- |\n")
	sb.WriteString(fmt.Sprintf("| **Total Requests Executed** | `%d` |\n", len(assertions)))
	sb.WriteString(fmt.Sprintf("| **Passed (Safe Handling)** | `🟢 %d` |\n", passed))
	sb.WriteString(fmt.Sprintf("| **Critical Failures (500 / Panic / InfoLeak)** | `🔴 %d` |\n", failed))
	sb.WriteString(fmt.Sprintf("| **Warnings (Contract Drift / Latency)** | `🟡 %d` |\n", warnings))
	sb.WriteString(fmt.Sprintf("| **Total Execution Duration** | `%s` |\n\n", duration.Round(time.Millisecond)))

	// Anomalies Section
	var failedAssertions []oracle.AssertionResult
	for _, a := range assertions {
		if a.Verdict == oracle.VerdictFail {
			failedAssertions = append(failedAssertions, a)
		}
	}

	if len(failedAssertions) > 0 {
		sb.WriteString("### 🚨 Detected Anomalies & Crash Payloads\n\n")
		sb.WriteString("The following edge-case payloads caused unhandled backend crashes or contract violations:\n\n")

		for i, a := range failedAssertions {
			findingText := "Unhandled Exception"
			if len(a.Findings) > 0 {
				findingText = a.Findings[0].Message
			}

			remediation := "Validate input before processing and return standard HTTP 4xx error."
			if a.Execution.StatusCode >= 500 {
				remediation = "Add defensive nil/bounds check to avoid backend runtime panic."
			}

			sb.WriteString(fmt.Sprintf("<details>\n<summary><b>🔴 [%d] %s %s</b> — <code>%s</code> (HTTP %d)</summary>\n\n",
				i+1,
				a.Vector.Method,
				a.Vector.Path,
				a.Vector.Scenario,
				a.Execution.StatusCode,
			))

			sb.WriteString(fmt.Sprintf("* **Scenario:** %s\n", a.Vector.Scenario))
			sb.WriteString(fmt.Sprintf("* **Mutation Type:** `%s`\n", a.Vector.MutationType))
			sb.WriteString(fmt.Sprintf("* **Finding:** `%s`\n", findingText))
			sb.WriteString(fmt.Sprintf("* **Latency:** `%s`\n\n", a.Execution.Duration.Round(time.Microsecond)))

			sb.WriteString("**cURL Reproducer:**\n```bash\n")
			sb.WriteString(a.Execution.CurlCommand)
			sb.WriteString("\n```\n\n")

			if a.Execution.ResponseBody != "" {
				sb.WriteString("**Response Body:**\n```json\n")
				sb.WriteString(strings.TrimSpace(a.Execution.ResponseBody))
				sb.WriteString("\n```\n\n")
			}

			sb.WriteString(fmt.Sprintf("**💡 Remediation Suggestion:**\n> %s\n\n", remediation))
			sb.WriteString("</details>\n\n")
		}
	} else {
		sb.WriteString("### ✨ All Endpoints Passed Safely\n\n")
		sb.WriteString("All boundary, adversarial, and AI mutations were handled gracefully by the target API without any 500 server crashes or unexpected panics.\n\n")
	}

	sb.WriteString("---\n")
	sb.WriteString("*Generated automatically by [FuzzSpec](https://github.com/fuzzspec/fuzzspec) — Spec-to-Contract AI Testing Harness.*\n")

	_ = statusColor // Silence unused warning

	sanitized := []byte(SanitizeText(sb.String()))
	return os.WriteFile(filePath, sanitized, 0644)
}
