package phase

import (
	"log/slog"

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
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing phase install", "base", *base)
			return query.RunPackInstall(*base, common)
		},
	}
}
