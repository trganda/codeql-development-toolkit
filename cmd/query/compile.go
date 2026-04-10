package query

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/query"
)

// newCompileCmd returns `query compile`.
func newCompileCmd(base *string) *cobra.Command {
	var lang, pack string
	var threads int
	cmd := &cobra.Command{
		Use:   "compile",
		Short: "Compile CodeQL query files (.ql and .qll)",
		Long: `Compile all .ql and .qll files using 'codeql query compile'.

Files are searched under <base>/<language>/<pack>/src/.
If --language and --pack are omitted, all query files found under <base> are compiled.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing query compile command", "base", *base, "language", lang, "pack", pack)
			return query.RunCompile(*base, lang, pack, threads)
		},
	}
	cmd.Flags().StringVar(&lang, "language", "", "Language of the query pack (e.g. go, java)")
	cmd.Flags().StringVar(&pack, "pack", "", "Pack name to compile")
	cmd.Flags().IntVar(&threads, "threads", 0, "Number of threads (0 = use all available cores)")
	return cmd
}
