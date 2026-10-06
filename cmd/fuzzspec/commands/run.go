package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fuzzspec/fuzzspec/internal/config"
	"github.com/fuzzspec/fuzzspec/internal/fuzzer"
	"github.com/fuzzspec/fuzzspec/internal/generator"
	"github.com/fuzzspec/fuzzspec/internal/oracle"
	"github.com/fuzzspec/fuzzspec/internal/parser"
	"github.com/fuzzspec/fuzzspec/internal/reporter"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var specPath string
	var targetURL string
	var outputJSON string
	var outputSARIF string
	var outputJUnit string
	var outputMD string
	var concurrency int
	var rps int
	var safeMode bool
	var autoDiscover bool
	var aiProvider string
	var aiModel string
	var noAI bool

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Execute full Spec-to-Contract fuzzing against a target API",
		Long:  `Runs concurrent HTTP fuzzing with boundary, adversarial, and AI-generated test vectors.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			startTime := time.Now()

			// 1. Load Config if available
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("config error: %w", err)
			}

			// CLI flags override config file
			if specPath != "" {
				cfg.Spec = specPath
			}
			if targetURL != "" {
				cfg.Target = targetURL
			}
			if concurrency > 0 {
				cfg.Execution.Concurrency = concurrency
			}
			if rps > 0 {
				cfg.Execution.RPS = rps
			}
			if cmd.Flags().Changed("safe-mode") {
				cfg.Execution.SafeMode = safeMode
			}
			if cmd.Flags().Changed("auto-discover") {
				cfg.AutoDiscover = autoDiscover
			}
			if aiProvider != "" {
				cfg.AI.Provider = aiProvider
			}
			if aiModel != "" {
				cfg.AI.Model = aiModel
			}
			if noAI {
				cfg.AI.Enabled = false
			}

			if cfg.Target == "" {
				return fmt.Errorf("--target is required (e.g. http://localhost:8080)")
			}

			// Auto-Discover specification if enabled or if spec is empty
			if cfg.Spec == "" {
				if cfg.AutoDiscover {
					fmt.Printf("🔍 Auto-probing OpenAPI/Swagger documentation on target %s...\n", cfg.Target)
					discoverer := parser.NewAutoDiscoverer()
					discoveredSpec, err := discoverer.Discover(context.Background(), cfg.Target, cfg.CustomLanguages)
					if err != nil {
						return fmt.Errorf("auto-discovery failed: %w (provide --spec explicitly)", err)
					}
					cfg.Spec = discoveredSpec
					fmt.Printf("🎯 Auto-Discovered Specification URL: %s\n", cfg.Spec)
				} else {
					return fmt.Errorf("either --spec or --auto-discover is required")
				}
			}

			fmt.Println("🚀 Initializing FuzzSpec Engine...")
			fmt.Printf("  • Target API:   %s\n", cfg.Target)
			fmt.Printf("  • Spec Source:  %s\n", cfg.Spec)
			fmt.Printf("  • Concurrency:  %d workers\n", cfg.Execution.Concurrency)
			fmt.Printf("  • Rate Limit:   %d RPS\n", cfg.Execution.RPS)
			fmt.Printf("  • Safe Mode:    %t (only GET/HEAD/OPTIONS)\n", cfg.Execution.SafeMode)
			fmt.Printf("  • AI Fuzzing:   %t (Provider: %s, Model: %s)\n\n", cfg.AI.Enabled, cfg.AI.Provider, cfg.AI.Model)

			// 2. Ingest OpenAPI Document
			p := parser.NewParser()
			parsed, err := p.Parse(context.Background(), cfg.Spec)
			if err != nil {
				return fmt.Errorf("failed to parse spec: %w", err)
			}
			fmt.Printf("✅ Specification Loaded: %s (v%s, %d endpoints)\n", parsed.Title, parsed.Version, len(parsed.Operations))

			// 3. Generate Test Vectors (Hybrid: Deterministic Rules + AI LLM Engine)
			mutator := generator.NewRuleMutator()
			var allVectors []generator.TestVector
			for _, op := range parsed.Operations {
				vectors := mutator.GenerateOperationVectors(op)
				allVectors = append(allVectors, vectors...)
			}
			fmt.Printf("⚡ Generated %d deterministic heuristic test vectors.\n", len(allVectors))

			// AI Semantic Vector Generation
			if cfg.AI.Enabled {
				aiGen := generator.NewAIGenerator(cfg.AI.Provider, cfg.AI.Model, cfg.AI.CacheDir, cfg.AI.CacheVectors)
				var aiCount int
				for _, op := range parsed.Operations {
					aiVectors, err := aiGen.GenerateAIVectors(context.Background(), op)
					if err != nil {
						// Gracefully report error and continue with rule-based vectors
						fmt.Printf("  ⚠️ AI vector warning for %s %s: %v\n", op.Method, op.Path, err)
						continue
					}
					if len(aiVectors) > 0 {
						aiCount += len(aiVectors)
						allVectors = append(allVectors, aiVectors...)
					}
				}
				if aiCount > 0 {
					fmt.Printf("🤖 Generated %d AI semantic edge-case test vectors.\n", aiCount)
				} else {
					fmt.Printf("ℹ️  AI generation: zero vectors generated or API key not set (proceeding with heuristic vectors).\n")
				}
			}

			fmt.Printf("📦 Total Test Suite Size: %d test vectors ready to execute.\n\n", len(allVectors))

			// 4. Initialize Fuzzer Worker Pool
			fuzzerOpts := fuzzer.FuzzerOptions{
				BaseURL:       cfg.Target,
				Concurrency:   cfg.Execution.Concurrency,
				RPS:           cfg.Execution.RPS,
				Timeout:       5 * time.Second,
				MaxRetries:    cfg.Execution.Retries,
				SafeMode:      cfg.Execution.SafeMode,
				GlobalHeaders: cfg.Authentication.Headers,
				AuthType:      cfg.Authentication.Type,
			}
			if cfg.Authentication.TokenEnv != "" {
				fuzzerOpts.AuthToken = os.Getenv(cfg.Authentication.TokenEnv)
			}

			pool := fuzzer.NewWorkerPool(fuzzerOpts)

			fmt.Printf("🔥 Executing HTTP Fuzzing against target %s...\n", cfg.Target)
			execResults := pool.ExecuteVectors(context.Background(), allVectors, func(completed, total int, res fuzzer.ExecutionResult) {
				fmt.Printf("\r  ⏳ Progress: [%d/%d requests executed] ...", completed, total)
			})
			fmt.Println("\r  ✔ All HTTP requests completed!                      ")

			// 5. Evaluate Multi-Layer QA Oracles
			oracleOpts := oracle.OracleOptions{
				FailOn5xx:          cfg.Oracles.FailOn5xx,
				FailOnSchemaDrift:   cfg.Oracles.FailOnSchemaDrift,
				ScanInfoLeak:       cfg.Oracles.ScanInfoLeak,
				LatencyThresholdMs: cfg.Oracles.LatencyThresholdMs,
				CustomLanguages:    cfg.CustomLanguages,
			}
			qaOracle := oracle.NewQAOracle(oracleOpts)

			var assertions []oracle.AssertionResult
			for _, res := range execResults {
				assertion := qaOracle.Evaluate(res)
				assertions = append(assertions, assertion)
			}

			// 6. Render Terminal Summary
			terminalReporter := reporter.NewTerminalReporter(os.Stdout)
			totalDuration := time.Since(startTime)
			passed, failed, warnings := terminalReporter.RenderSummary(assertions, totalDuration)

			// 7. Export Reports (JSON, SARIF, JUnit, Markdown)
			if outputJSON != "" {
				cfg.Reporting.JSON = outputJSON
			}
			if outputSARIF != "" {
				cfg.Reporting.SARIF = outputSARIF
			}
			if outputJUnit != "" {
				cfg.Reporting.JUnit = outputJUnit
			}
			if outputMD != "" {
				cfg.Reporting.Markdown = outputMD
			}

			if cfg.Reporting.JSON != "" {
				if err := reporter.ExportJSON(cfg.Reporting.JSON, assertions, totalDuration, passed, failed, warnings); err != nil {
					fmt.Fprintf(os.Stderr, "⚠️ Failed to export JSON report: %v\n", err)
				} else {
					fmt.Printf("💾 Diagnostic JSON Report saved to: %s\n", cfg.Reporting.JSON)
				}
			}
			if cfg.Reporting.SARIF != "" {
				if err := reporter.ExportSARIF(cfg.Reporting.SARIF, assertions); err != nil {
					fmt.Fprintf(os.Stderr, "⚠️ Failed to export SARIF report: %v\n", err)
				} else {
					fmt.Printf("🛡️ GitHub Code Scanning SARIF report saved to: %s\n", cfg.Reporting.SARIF)
				}
			}
			if cfg.Reporting.JUnit != "" {
				if err := reporter.ExportJUnit(cfg.Reporting.JUnit, assertions, totalDuration); err != nil {
					fmt.Fprintf(os.Stderr, "⚠️ Failed to export JUnit XML report: %v\n", err)
				} else {
					fmt.Printf("📊 CI JUnit XML report saved to: %s\n", cfg.Reporting.JUnit)
				}
			}
			if cfg.Reporting.Markdown != "" {
				if err := reporter.ExportMarkdown(cfg.Reporting.Markdown, assertions, totalDuration, passed, failed, warnings); err != nil {
					fmt.Fprintf(os.Stderr, "⚠️ Failed to export Markdown report: %v\n", err)
				} else {
					fmt.Printf("📝 GitHub PR Markdown report saved to: %s\n", cfg.Reporting.Markdown)
				}
			}

			// 8. Quality Gate Exit Code: Fail build if critical errors exist
			if failed > 0 {
				fmt.Fprintf(os.Stderr, "\n\033[31m❌ QUALITY GATE FAILED: %d critical anomaly/500 crash detected!\033[0m\n", failed)
				os.Exit(1)
			}

			fmt.Println("\n\033[32m✨ QUALITY GATE PASSED: All endpoints handled boundary mutations safely.\033[0m")
			return nil
		},
	}

	cmd.Flags().StringVarP(&specPath, "spec", "s", "", "path to OpenAPI YAML/JSON file or remote URL")
	cmd.Flags().StringVarP(&targetURL, "target", "t", "", "target base URL (e.g. http://localhost:8080)")
	cmd.Flags().StringVar(&outputJSON, "output-json", "", "path to export diagnostic JSON report")
	cmd.Flags().StringVar(&outputSARIF, "output-sarif", "", "path to export SARIF v2.1.0 code scanning report")
	cmd.Flags().StringVar(&outputJUnit, "output-junit", "", "path to export JUnit XML test report")
	cmd.Flags().StringVar(&outputMD, "output-md", "", "path to export GitHub Actions PR Markdown report")
	cmd.Flags().IntVar(&concurrency, "concurrency", 0, "concurrency worker count (default: 10)")
	cmd.Flags().IntVar(&rps, "rps", 0, "maximum requests per second rate limit (default: 20)")
	cmd.Flags().BoolVar(&safeMode, "safe-mode", true, "restrict fuzzing to safe methods (GET, HEAD, OPTIONS)")
	cmd.Flags().BoolVar(&autoDiscover, "auto-discover", false, "automatically probe and discover OpenAPI specification from target URL")
	cmd.Flags().StringVar(&aiProvider, "ai-provider", "", "AI provider to use (gemini, openai, anthropic, local)")
	cmd.Flags().StringVar(&aiModel, "ai-model", "", "AI model name (e.g. gemini-1.5-flash, gpt-4o-mini)")
	cmd.Flags().BoolVar(&noAI, "no-ai", false, "disable AI generator, use only rule-based mutations")

	return cmd
}
