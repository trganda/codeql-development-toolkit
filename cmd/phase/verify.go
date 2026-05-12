package phase

import (
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/trganda/codeql-development-toolkit/internal/utils"
)

func newVerifyCmd(base *string, common *utils.CommonFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "verify",
		Short: "Verify CodeQL query quality (placeholder)",
		Long: `Verify phase: run quality checks on compiled queries.

Runs the full chain: install → compile → test → verify.
Requires workspace initialization (run 'qlt phase init' first).

This phase is a placeholder. Full implementation will validate CodeQL query
metadata and run integration checks.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing phase verify", "base", *base, "numThreads", common.NumThreads)
			return runVerifyChain(*base, common)
		},
	}
}
