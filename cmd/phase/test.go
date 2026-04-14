package phase

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
)

func newTestCmd(base *string, common *commonFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Run CodeQL unit tests",
		Long: `Test phase: run CodeQL unit tests for a language.

Runs the full chain: install → compile → test.
Requires workspace initialization (run 'qlt phase init' first).`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if common.language == "" {
				return fmt.Errorf("required flag \"language\" not set")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing phase test", "base", *base, "language", common.language, "threads", common.numThreads)
			return runTestChain(*base, common)
		},
	}
}
