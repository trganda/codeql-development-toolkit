package pack

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/codeql"
	"github.com/trganda/codeql-development-toolkit/internal/language"
	"github.com/trganda/codeql-development-toolkit/internal/pack"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

// newAddTestCmd returns `pack add-test`.
func newAddTestCmd(base *string) *cobra.Command {

	var (
		queryName string
		lang      string
		packName  string
		queryKind string
		overwrite bool
	)

	cmd := &cobra.Command{
		Use:   "add-test",
		Short: "Add a test pack to an existing CodeQL pack",
		Long: "Create the test pack scaffolding (test/qlpack.yml and test files for a query) " +
			"for an already-generated CodeQL pack. The target pack must already exist; " +
			"only files under test/ are written.",
		PreRun: func(cmd *cobra.Command, args []string) {
			if !language.IsSupported(lang) {
				slog.Error("Invalid --language", "supported", language.SupportedLanguages, "got", lang)
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			slog.Debug("Executing pack add-test command",
				"name", queryName, "language", lang, "pack", packName, "kind", queryKind)

			codeqlBin, err := paths.ResolveCodeQLBinary(*base)
			if err != nil {
				slog.Error("Resolve CodeQL binary failed", "base", *base, "err", err)
				os.Exit(1)
			}

			if err := pack.AddTestPack(codeql.NewCLI(codeqlBin), pack.GeneratePackOptions{
				Base:      *base,
				QueryName: queryName,
				Lang:      lang,
				Pack:      packName,
				Overwrite: overwrite,
			}); err != nil {
				slog.Error("Add test pack failed", "pack", packName, "err", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVar(&queryName, "query-name", "MyNewQuery", "Name of the query to scaffold test files for (must match a query in the existing pack)")
	cmd.Flags().StringVar(&lang, "language", "", "Language (cpp|csharp|go|java|javascript|python|ruby)")
	cmd.Flags().StringVar(&packName, "pack", "", "CodeQL pack name (e.g. trganda/new-pack)")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite existing test pack related files")
	cmd.MarkFlagRequired("language")
	cmd.MarkFlagRequired("pack")

	return cmd
}
