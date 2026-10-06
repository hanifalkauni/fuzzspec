package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/fuzzspec/fuzzspec/internal/parser"
	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	var specPath string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate an OpenAPI 3.0/3.1 specification for readiness",
		Long:  `Validates the structural integrity and extractable operations of an OpenAPI document.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if specPath == "" {
				return fmt.Errorf("--spec flag is required")
			}

			fmt.Printf("🔍 Parsing and validating OpenAPI spec: %s\n", specPath)

			p := parser.NewParser()
			parsed, err := p.Parse(context.Background(), specPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Validation failed: %v\n", err)
				return err
			}

			fmt.Println("✅ Specification is valid and ready for fuzzing!")
			fmt.Printf("  • Title:       %s\n", parsed.Title)
			fmt.Printf("  • Version:     %s\n", parsed.Version)
			fmt.Printf("  • Doc Version: OpenAPI %s\n", parsed.DocVersion)
			fmt.Printf("  • Operations:  %d endpoints extracted\n", len(parsed.Operations))

			fmt.Println("\n📋 Extracted Endpoints:")
			for _, op := range parsed.Operations {
				paramCount := len(op.Parameters)
				hasBody := op.RequestBody != nil
				bodyDesc := "no body"
				if hasBody {
					bodyDesc = "JSON body"
				}
				fmt.Printf("  [%-6s] %-30s (%d params, %s)\n", op.Method, op.Path, paramCount, bodyDesc)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&specPath, "spec", "s", "", "path to OpenAPI YAML/JSON file or remote URL")
	return cmd
}
