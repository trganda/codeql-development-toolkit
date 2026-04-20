package query

import (
	"log/slog"

	"github.com/spf13/cobra"
)

func newGenerateCmd(base *string) *cobra.Command {

	var (
		queryName string
		lang      string
		packName  string
		scope     string
		queryKind string
	)

	gen := cobra.Command{
		Use:   "generate",
		Short: "Generate a query and add it to a pack",
		Long:  `Generate a query from a template. This command creates a new query file based on the specified template and parameters.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing query generate command",
				"name", queryName, "language", lang, "pack", packName, "kind", queryKind)

			// Placeholder for actual generation logic
			slog.Info("Query generation — coming soon")
			return nil
		},
	}
	gen.Flags().StringVar(&queryName, "query-name", "GeneratedQuery", "Name of the query in the pack (e.g. GeneratedQuery)")
	gen.Flags().StringVar(&lang, "language", "", "Language (c|cpp|csharp|go|java|javascript|python|ruby)")
	gen.Flags().StringVar(&packName, "pack", "", "CodeQL pack name")
	gen.Flags().StringVar(&scope, "scope", "", "CodeQL pack scope (optional), use globally configured scope in qlt.conf.json if not provided")
	gen.Flags().StringVar(&queryKind, "query-kind", "problem", "Query kind of the query (problem|path-problem)")
	gen.MarkFlagRequired("language")
	gen.MarkFlagRequired("pack")

	return &gen
}
