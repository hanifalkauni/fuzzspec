package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fuzzspec/fuzzspec/internal/fuzzer"
	"github.com/fuzzspec/fuzzspec/internal/oracle"
	"github.com/fuzzspec/fuzzspec/internal/replay"
	"github.com/spf13/cobra"
)

func newReplayCmd() *cobra.Command {
	var reportFile string
	var targetURL string

	cmd := &cobra.Command{
		Use:   "replay",
		Short: "Replay previously failed test vectors to verify bug fixes (zero AI cost)",
		Long:  `Reads a diagnostic JSON report, isolates the failing payloads, and re-tests them against the target server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if reportFile == "" {
				return fmt.Errorf("--file flag is required (path to previous report.json)")
			}
			if targetURL == "" {
				return fmt.Errorf("--target flag is required (e.g. http://localhost:8080)")
			}

			fmt.Println("🔄 Initializing FuzzSpec Replay Engine...")
			fmt.Printf("  • Target Server: %s\n", targetURL)
			fmt.Printf("  • Report Source: %s\n\n", reportFile)

			engine := replay.NewReplayEngine(fuzzer.FuzzerOptions{
				BaseURL:  targetURL,
				Timeout:  5 * time.Second,
				SafeMode: false,
			})

			failingVectors, err := engine.LoadFailingVectors(reportFile)
			if err != nil {
				return fmt.Errorf("failed to load failing vectors: %w", err)
			}

			if len(failingVectors) == 0 {
				fmt.Println("✨ No failed vectors found in the report file. Nothing to replay!")
				return nil
			}

			fmt.Printf("🎯 Replaying %d previously failed test case(s)...\n\n", len(failingVectors))

			summary := engine.Replay(context.Background(), failingVectors)

			for i, res := range summary.Results {
				if res.Verdict == oracle.VerdictPass {
					fmt.Printf("  [%d/%d] \033[32m✔ RESOLVED\033[0m: %s %s (%s) -> HTTP %d %s\n",
						i+1, summary.TotalReplayed, res.Vector.Method, res.Vector.Path, res.Vector.Scenario, res.Execution.StatusCode, res.Execution.StatusText)
				} else {
					fmt.Printf("  [%d/%d] \033[31m✖ STILL FAILING\033[0m: %s %s (%s) -> HTTP %d %s\n",
						i+1, summary.TotalReplayed, res.Vector.Method, res.Vector.Path, res.Vector.Scenario, res.Execution.StatusCode, res.Execution.StatusText)
					if len(res.Findings) > 0 {
						fmt.Printf("         \033[31m• %s\033[0m\n", res.Findings[0].Message)
					}
				}
			}

			fmt.Println("\n--------------------------------------------------------------------------------")
			fmt.Printf("🏁 REPLAY SUMMARY: Total: %d | \033[32mResolved: %d\033[0m | \033[31mStill Failing: %d\033[0m\n",
				summary.TotalReplayed, summary.Resolved, summary.StillFailing)
			fmt.Println("================================================================================")

			if summary.StillFailing > 0 {
				fmt.Fprintf(os.Stderr, "\n\033[31m❌ REPLAY FAILED: %d bug(s) still active on target server!\033[0m\n", summary.StillFailing)
				os.Exit(1)
			}

			fmt.Println("\n\033[32m✨ REPLAY PASSED: All previous defects have been verified as resolved!\033[0m")
			return nil
		},
	}

	cmd.Flags().StringVarP(&reportFile, "file", "f", "", "path to diagnostic JSON report containing failed payloads")
	cmd.Flags().StringVarP(&targetURL, "target", "t", "", "target base URL (e.g. http://localhost:8080)")

	return cmd
}
