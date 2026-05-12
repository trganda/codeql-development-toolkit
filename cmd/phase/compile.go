package phase

import (
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/trganda/codeql-development-toolkit/internal/utils"
)

func newCompileCmd(base *string, common *utils.CommonFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "compile",
		Short: "Compile CodeQL queries",
		Long: `Compile phase: compile CodeQL query files (.ql and .qll).

Runs the full chain: install → compile.
Requires workspace initialization (run 'qlt phase init' first).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing phase compile", "base", *base, "threads", common.NumThreads)
			return runCompileChain(*base, common)
		},
	}
}
