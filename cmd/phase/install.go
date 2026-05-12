package phase

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/query"
	"github.com/trganda/codeql-development-toolkit/internal/utils"
)

func newInstallCmd(base *string, common *utils.CommonFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install CodeQL pack dependencies",
		Long: `Install phase: install dependencies for CodeQL packs.

Requires workspace initialization (run 'qlt phase init' first).
Runs 'codeql pack install' for packs found under <base>, optionally filtered
by --pack (repeatable).`,
		Run: func(cmd *cobra.Command, args []string) {
			slog.Debug("Executing phase install", "base", *base)
			if err := query.RunPackInstall(*base, common); err != nil {
				slog.Error("Phase install failed", "err", err)
				os.Exit(1)
			}
		},
	}
}
