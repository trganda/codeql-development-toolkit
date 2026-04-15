package phase

import (
	"log/slog"

	"github.com/spf13/cobra"
)

func newTestCmd(base string, common *commonFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Run CodeQL unit tests",
		Long: `Test phase: run CodeQL unit tests for a language.

Runs the full chain: install → compile → test.
Requires workspace initialization (run 'qlt phase init' first).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing phase test", "base", base, "language", common.language, "threads", common.numThreads)
			return runTestChain(base, common)
		},
	}
}
