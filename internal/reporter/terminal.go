package reporter

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fuzzspec/fuzzspec/internal/oracle"
)

// TerminalReporter renders rich ANSI colored summary tables and diagnostics.
type TerminalReporter struct {
	out io.Writer
}

// NewTerminalReporter creates a new terminal reporter.
func NewTerminalReporter(out io.Writer) *TerminalReporter {
	if out == nil {
		out = os.Stdout
	}
	return &TerminalReporter{out: out}
}

// RenderSummary prints the final evaluation summary table and cURL reproducers.
func (r *TerminalReporter) RenderSummary(assertions []oracle.AssertionResult, totalDuration time.Duration) (int, int, int) {
	var passed, failed, warnings int
	var criticalFailures []oracle.AssertionResult

	fmt.Fprintln(r.out, "\n"+"================================================================================")
	fmt.Fprintln(r.out, "                        📊 FUZZSPEC EXECUTION RESULTS                          ")
	fmt.Fprintln(r.out, "================================================================================")

	for _, a := range assertions {
		statusStr := fmt.Sprintf("%d", a.Execution.StatusCode)
		if a.Execution.IsConnectionErr {
			statusStr = "CONN_ERR"
		}

		switch a.Verdict {
		case oracle.VerdictPass:
			passed++
			fmt.Fprintf(r.out, "  \033[32m✔ PASS\033[0m [%-6s] %-30s | %-8s | %4dms | %s\n",
				a.Vector.Method, a.Vector.Path, statusStr, a.Execution.Duration.Milliseconds(), a.Vector.Scenario)
		case oracle.VerdictWarn:
			warnings++
			fmt.Fprintf(r.out, "  \033[33m⚠ WARN\033[0m [%-6s] %-30s | %-8s | %4dms | %s (%s)\n",
				a.Vector.Method, a.Vector.Path, statusStr, a.Execution.Duration.Milliseconds(), a.Vector.Scenario, a.Summary)
		case oracle.VerdictFail:
			failed++
			fmt.Fprintf(r.out, "  \033[31m✖ FAIL\033[0m [%-6s] %-30s | %-8s | %4dms | %s\n",
				a.Vector.Method, a.Vector.Path, statusStr, a.Execution.Duration.Milliseconds(), a.Vector.Scenario)
			criticalFailures = append(criticalFailures, a)
		}
	}

	fmt.Fprintln(r.out, "--------------------------------------------------------------------------------")
	fmt.Fprintf(r.out, "🏁 TOTAL: %d | \033[32mPASSED: %d\033[0m | \033[31mFAILED: %d\033[0m | \033[33mWARNED: %d\033[0m | Duration: %v\n",
		len(assertions), passed, failed, warnings, totalDuration.Round(time.Millisecond))
	fmt.Fprintln(r.out, "================================================================================")

	// Print cURL Reproducers for failed test cases
	if len(criticalFailures) > 0 {
		fmt.Fprintf(r.out, "\n\033[31m🚨 DETECTED ANOMALIES (%d defect(s) found) - cURL Reproducers:\033[0m\n", len(criticalFailures))
		for idx, fail := range criticalFailures {
			fmt.Fprintf(r.out, "\n[%d] %s %s - %s\n", idx+1, fail.Vector.Method, fail.Vector.Path, fail.Vector.Scenario)
			for _, f := range fail.Findings {
				fmt.Fprintf(r.out, "    \033[31m• [%s] %s\033[0m\n", f.Category, f.Message)
				if f.Evidence != "" {
					fmt.Fprintf(r.out, "      Evidence: %s\n", SanitizeText(strings.ReplaceAll(f.Evidence, "\n", " ")))
				}
			}
			fmt.Fprintln(r.out, "    \033[36mReproduce Command:\033[0m")
			fmt.Fprintf(r.out, "    %s\n", SanitizeText(fail.Execution.CurlCommand))
		}
		fmt.Fprintln(r.out, "")
	}

	return passed, failed, warnings
}
