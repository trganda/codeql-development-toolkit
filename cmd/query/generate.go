package query

import (
	"fmt"
	"log/slog"

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
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if !language.IsSupported(lang) {
				return fmt.Errorf("--language must be one of %v, got %q", language.SupportedLanguages, lang)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing query generate command",
				"name", queryName, "language", lang, "pack", packName, "kind", queryKind)

			err := query.GenerateNewQuery(&query.GenerateQueryOptions{
				Base:      *base,
				Name:      queryName,
				Language:  lang,
				QueryKind: queryKind,
				PackName:  packName,
			})
			if err != nil {
				slog.Error("Query generation failed", "details", err.Error())
			}
			return nil
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
