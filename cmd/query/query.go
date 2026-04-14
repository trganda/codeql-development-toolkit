package query

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/query"
)

// NewCommand returns the `query` cobra command.
func NewCommand(base, automationType *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query feature commands",
	}

	cmd.AddCommand(newRunCmd(base))
	return cmd
}

// newRunCmd returns `query run`.
func newRunCmd(base *string) *cobra.Command {
	var (
		queryName       string
		database        string
		lang            string
		pack            string
		format          string
		output          string
		additionalPacks string
		threads         int
	)
	run := &cobra.Command{
		Use:   "run",
		Short: "Run a CodeQL query against a database",
		Long: `Run a CodeQL query against a database using 'codeql database analyze'.

The query is located by name in order:
  1. Config lookup in qlt.conf.json (recorded by 'qlt query generate new-query')
  2. Filesystem search up to 3 levels under <base>/<language>/[pack]

The CodeQL binary is resolved in order:
  1. Bundle binary at $HOME/.qlt/bundle/<hash>/codeql/codeql (when EnableCustomCodeQLBundles=true)
  2. CLI binary at $HOME/.qlt/packages/<hash>/codeql/codeql (installed by 'qlt codeql install')
  3. codeql found on PATH`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing query run",
				"query", queryName, "database", database, "language", lang,
				"pack", pack, "format", format, "output", output, "threads", threads)
			return query.RunQuery(*base, queryName, database, lang, pack, format, output, additionalPacks, threads)
		},
	}
	run.Flags().StringVar(&queryName, "query", "", "Query name to run, e.g. MyQuery (required)")
	run.Flags().StringVar(&database, "database", "", "Path to the CodeQL database (required)")
	run.Flags().StringVar(&lang, "language", "", "Language of the query, e.g. go, java (required)")
	run.Flags().StringVar(&pack, "pack", "", "Pack name to narrow the search (optional)")
	run.Flags().StringVar(&format, "format", "sarif-latest", "Output format: sarif-latest, csv, text, dot, bqrs")
	run.Flags().StringVar(&output, "output", "", "Output file path (default: <query-name>.<ext> beside the query file)")
	run.Flags().StringVar(&additionalPacks, "additional-packs", "", "Colon-separated list of additional pack search paths")
	run.Flags().IntVar(&threads, "threads", 0, "Number of threads (0 = use all available cores)")
	run.MarkFlagRequired("query")
	run.MarkFlagRequired("database")
	run.MarkFlagRequired("language")
	return run
}
