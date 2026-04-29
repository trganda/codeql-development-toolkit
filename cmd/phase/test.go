package phase

import (
	"log/slog"

	"github.com/spf13/cobra"
)

func newTestCmd(base *string, common *commonFlags) *cobra.Command {
	var outputPath string
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run CodeQL unit tests",
		Long: `Test phase: run CodeQL unit tests for a language.

Runs the full chain: install → compile → test.
Requires workspace initialization (run 'qlt phase init' first).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing phase test", "base", *base, "language", common.numThreads, "output", outputPath)
			var reportOutput *string
			if cmd.Flags().Changed("output") {
				reportOutput = &outputPath
			}
			return runTestChain(*base, reportOutput, common)
		},
	}
	cmd.Flags().StringVar(&outputPath, "output", "", "Write test report to the given JSON file (default when empty: <base>/target/test/test-report-<timestamp>.json)")
	cmd.Flag("output").NoOptDefVal = ""
	return cmd
}
