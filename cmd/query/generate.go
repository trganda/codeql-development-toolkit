package query

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/trganda/codeql-development-toolkit/internal/language"
	"github.com/trganda/codeql-development-toolkit/internal/query"
)

func newGenerateCmd(base *string) *cobra.Command {

	var (
		queryName string
		lang      string
		packName  string
		queryKind string
	)

	gen := cobra.Command{
		Use:   "generate",
		Short: "Generate a query and add it to a pack",
		Long:  `Generate a query from a template. This command creates a new query file based on the specified template and parameters.`,
		PreRun: func(cmd *cobra.Command, args []string) {
			if !language.IsSupported(lang) {
				slog.Error("Invalid --language", "supported", language.SupportedLanguages, "got", lang)
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			slog.Debug("Executing query generate command",
				"name", queryName, "language", lang, "pack", packName, "kind", queryKind)

			if err := query.GenerateNewQuery(&query.GenerateQueryOptions{
				Base:      *base,
				Name:      queryName,
				Language:  lang,
				QueryKind: queryKind,
				PackName:  packName,
			}); err != nil {
				slog.Error("Query generation failed", "err", err)
				os.Exit(1)
			}
		},
	}
	gen.Flags().StringVar(&queryName, "query-name", "GeneratedQuery", "Name of the query in the pack (e.g. GeneratedQuery)")
	gen.Flags().StringVar(&lang, "language", "", "Language (cpp|csharp|go|java|javascript|python|ruby)")
	gen.Flags().StringVar(&packName, "pack", "", "CodeQL pack name")
	gen.Flags().StringVar(&queryKind, "query-kind", "problem", "Query kind of the query (problem|path-problem)")
	gen.MarkFlagRequired("language")
	gen.MarkFlagRequired("pack")

	return &gen
}
