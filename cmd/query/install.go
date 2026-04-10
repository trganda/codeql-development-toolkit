package query

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/query"
)

// newInstallCmd returns `query install`.
func newInstallCmd(base *string) *cobra.Command {
	var lang, pack string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install dependencies for a query pack",
		Long: `Install dependencies declared in a query pack's qlpack.yml using 'codeql pack install'.

The qlpack.yml is located at <base>/<language>/<pack>/src/qlpack.yml.
If --language and --pack are omitted, every qlpack.yml found under <base> is installed.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing query install command", "base", *base, "language", lang, "pack", pack)
			return query.RunPackInstall(*base, lang, pack)
		},
	}
	cmd.Flags().StringVar(&lang, "language", "", "Language of the query pack (e.g. go, java)")
	cmd.Flags().StringVar(&pack, "pack", "", "Pack name to install dependencies for")
	return cmd
}
