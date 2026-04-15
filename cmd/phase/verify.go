package phase

import (
	"log/slog"

	"github.com/spf13/cobra"
)

func newVerifyCmd(base string, common *commonFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "verify",
		Short: "Verify CodeQL query quality (placeholder)",
		Long: `Verify phase: run quality checks on compiled queries.

Runs the full chain: install → compile → test → verify.
Requires workspace initialization (run 'qlt phase init' first).

This phase is a placeholder. Full implementation will validate CodeQL query
metadata and run integration checks.

For now, use: qlt validation run check-queries --language <lang>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing phase verify", "base", base, "language", common.language)
			return runVerifyChain(base, common)
		},
	}
}
