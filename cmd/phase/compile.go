package phase

import (
	"log/slog"

	"github.com/spf13/cobra"
)

func newCompileCmd(base string, common *commonFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "compile",
		Short: "Compile CodeQL queries",
		Long: `Compile phase: compile CodeQL query files (.ql and .qll).

Runs the full chain: install → compile.
Requires workspace initialization (run 'qlt phase init' first).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing phase compile", "base", base, "language", common.language, "threads", common.numThreads)
			return runCompileChain(base, common)
		},
	}
}
