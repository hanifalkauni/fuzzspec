package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool

	rootCmd = &cobra.Command{
		Use:   "fuzzspec",
		Short: "FuzzSpec - Spec-to-Contract AI Testing Harness & MCP",
		Long: `FuzzSpec is a high-performance, developer-first Spec-to-Contract QA Testing Harness.
It ingests OpenAPI/JSON Schema specifications, generates boundary and adversarial test vectors,
and executes concurrent HTTP fuzzing to prevent 500 runtime crashes.`,
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is ./fuzzspec.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose debug logging")

	rootCmd.AddCommand(newValidateCmd())
	rootCmd.AddCommand(newGenerateCmd())
	rootCmd.AddCommand(newRunCmd())
	rootCmd.AddCommand(newReplayCmd())
	rootCmd.AddCommand(newVersionCmd())
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print FuzzSpec version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("FuzzSpec v1.4.0 (Milestone 1 Foundation)")
		},
	}
}
