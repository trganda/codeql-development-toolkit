package lifecycle

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/query"
)

func newInstallCmd(base *string) *cobra.Command {
	var lang, packName string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install CodeQL pack dependencies",
		Long: `Install lifecycle phase: install dependencies for CodeQL packs.

Runs 'codeql pack install' for packs found under <base>, optionally filtered
by language and pack name.

Corresponds to: qlt query install`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing lifecycle install", "base", *base, "language", lang, "pack", packName)
			return query.RunPackInstall(*base, lang, packName)
		},
	}
	cmd.Flags().StringVar(&lang, "language", "", "Filter by language (e.g. go, java)")
	cmd.Flags().StringVar(&packName, "pack", "", "Filter by pack name (exact match on the pack segment)")
	return cmd
}
