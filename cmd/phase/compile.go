package phase

import (
	"log/slog"
	"os"

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
		Run: func(cmd *cobra.Command, args []string) {
			slog.Debug("Executing phase compile", "base", *base, "threads", common.NumThreads)
			if err := runCompileChain(*base, common); err != nil {
				slog.Error("Phase compile failed", "err", err)
				os.Exit(1)
			}
		},
	}
}
