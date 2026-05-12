package phase

import (
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/trganda/codeql-development-toolkit/internal/utils"
)

func newTestCmd(base *string, common *utils.CommonFlags) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run CodeQL unit tests",
		Long: `Test phase: run CodeQL unit tests for a language.

Runs the full chain: install → compile → test.
Requires workspace initialization (run 'qlt phase init' first).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing phase test", "base", *base, "numThreads", common.NumThreads, "output", output)
			return runTestChain(*base, output, common)
		},
	}
	cmd.Flags().StringVar(&output, "output", "", "Write test report to the given JSON file (skip generating report if empty)")

	return cmd
}
