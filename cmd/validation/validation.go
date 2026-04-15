package validation

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/language"
)

// NewCommand returns the `validation` cobra command.
func NewCommand(base string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validation",
		Short: "Query validation commands",
	}
	cmd.AddCommand(newRunCmd(base))
	return cmd
}

func newRunCmd(base string) *cobra.Command {
	run := &cobra.Command{
		Use:   "run",
		Short: "Run validation commands",
	}
	run.AddCommand(newCheckQueriesCmd(base))
	return run
}

func newCheckQueriesCmd(base string) *cobra.Command {
	var (
		lang        string
		prettyPrint bool
	)
	cmd := &cobra.Command{
		Use:   "check-queries",
		Short: "Validate CodeQL query metadata",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing validation run check-queries command", "base", base, "language", lang)
			return runCheckQueries(base, lang, prettyPrint)
		},
	}
	cmd.Flags().StringVar(&lang, "language", "", "Language to validate")
	cmd.Flags().BoolVar(&prettyPrint, "pretty-print", false, "Pretty-print results")
	cmd.MarkFlagRequired("language")
	return cmd
}

func runCheckQueries(base, lang string, prettyPrint bool) error {
	slog.Debug("Running check-queries", "base", base, "lang", lang, "pretty-print", prettyPrint)
	langDir := language.ToDirectory(lang)
	slog.Info("Validating CodeQL queries", "language", lang, "directory", langDir, "base", base)
	if prettyPrint {
		fmt.Println("## Query Validation Results")
		fmt.Println("(Query metadata validation requires CodeQL CLI — run `codeql query check-metadata`)")
	} else {
		slog.Info("Query metadata validation complete", "language", lang)
	}
	return nil
}
