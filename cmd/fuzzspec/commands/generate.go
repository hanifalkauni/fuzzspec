package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/fuzzspec/fuzzspec/internal/generator"
	"github.com/fuzzspec/fuzzspec/internal/parser"
	"github.com/spf13/cobra"
)

func newGenerateCmd() *cobra.Command {
	var specPath string
	var outputPath string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate test vectors from an OpenAPI specification without executing HTTP calls",
		Long:  `Generates deterministic boundary and adversarial test vectors and exports them to a JSON file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if specPath == "" {
				return fmt.Errorf("--spec flag is required")
			}

			fmt.Printf("📦 Ingesting OpenAPI spec: %s\n", specPath)

			p := parser.NewParser()
			parsed, err := p.Parse(context.Background(), specPath)
			if err != nil {
				return fmt.Errorf("failed to parse spec: %w", err)
			}

			mutator := generator.NewRuleMutator()
			var allVectors []generator.TestVector

			for _, op := range parsed.Operations {
				opVectors := mutator.GenerateOperationVectors(op)
				allVectors = append(allVectors, opVectors...)
			}

			fmt.Printf("⚡ Generated %d test vectors across %d endpoints.\n", len(allVectors), len(parsed.Operations))

			if outputPath != "" {
				data, err := json.MarshalIndent(allVectors, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to serialize test vectors: %w", err)
				}

				if err := os.WriteFile(outputPath, data, 0644); err != nil {
					return fmt.Errorf("failed to write output file: %w", err)
				}
				fmt.Printf("💾 Test vectors exported to: %s\n", outputPath)
			} else {
				fmt.Println("\nSample Generated Vectors:")
				limit := 5
				if len(allVectors) < limit {
					limit = len(allVectors)
				}
				for i := 0; i < limit; i++ {
					v := allVectors[i]
					fmt.Printf("  • [%s] %s %s - %s (%s)\n", v.ID, v.Method, v.Path, v.Scenario, v.MutationType)
				}
				if len(allVectors) > limit {
					fmt.Printf("  ... and %d more vectors. Use --output <file.json> to export all.\n", len(allVectors)-limit)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&specPath, "spec", "s", "", "path to OpenAPI YAML/JSON file or remote URL")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "path to save generated test vectors (JSON format)")
	return cmd
}
