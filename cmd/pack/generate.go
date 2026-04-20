package pack

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/codeql"
	"github.com/trganda/codeql-development-toolkit/internal/pack"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

// newGenerateCmd returns `pack generate`.
func newGenerateCmd(base *string) *cobra.Command {

	var (
		queryName   string
		lang        string
		packName    string
		scope       string
		queryKind   string
		createTests bool
		overwrite   bool
		library     bool
		bundle      bool
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Create a new CodeQL pack with scaffolding",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing pack generate new-query command",
				"name", queryName, "language", lang, "pack", packName, "kind", queryKind, "use-bundle", bundle)

			codeqlBin, err := paths.ResolveCodeQLBinary(*base)
			if err != nil {
				return fmt.Errorf("resolve CodeQL binary: %w", err)
			}

			return pack.GenerateNewQuery(codeql.NewCLI(codeqlBin), pack.GenerateArgs{
				Base:        *base,
				QueryName:   queryName,
				Lang:        lang,
				Pack:        packName,
				Scope:       scope,
				QueryKind:   queryKind,
				CreateTests: createTests,
				Overwrite:   overwrite,
				UseBundle:   bundle,
				Library:     library,
			})
		},
	}

	cmd.Flags().StringVar(&queryName, "query-name", "MyNewQuery", "Name of the first query in the new pack (e.g. MyNewQuery)")
	cmd.Flags().StringVar(&lang, "language", "", "Language (c|cpp|csharp|go|java|javascript|python|ruby)")
	cmd.Flags().StringVar(&packName, "pack", "", "CodeQL pack name")
	cmd.Flags().StringVar(&scope, "scope", "", "CodeQL pack scope (optional), use globally configured scope in qlt.conf.json if not provided")
	cmd.Flags().StringVar(&queryKind, "query-kind", "problem", "Query kind (problem|path-problem)")
	cmd.Flags().BoolVar(&createTests, "create-tests", true, "Create test scaffolding")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite existing pack related files")
	cmd.Flags().BoolVar(&library, "library", false, "Mark the generated qlpack as a library pack (sets library: true)")
	cmd.Flags().BoolVar(&bundle, "bundle", false, "Add to a custom CodeQL bundle")
	cmd.MarkFlagRequired("language")
	cmd.MarkFlagRequired("pack")

	return cmd
}
