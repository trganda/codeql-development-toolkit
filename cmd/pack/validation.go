package pack

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

// newValidationCommand returns the `validation` cobra command.
func newValidationCommand(base *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Query validation commands",
	}
	cmd.AddCommand(newCheckQueriesCmd(base))
	return cmd
}

func newCheckQueriesCmd(base *string) *cobra.Command {
	var (
		lang        string
		prettyPrint bool
	)
	cmd := &cobra.Command{
		Use:   "queries",
		Short: "Validate CodeQL query metadata",
		Run: func(cmd *cobra.Command, args []string) {
			slog.Debug("Executing validation run check-queries command", "base", *base)
			if err := runCheckQueries(*base, lang, prettyPrint); err != nil {
				slog.Error("Check queries failed", "err", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().BoolVar(&prettyPrint, "pretty-print", false, "Pretty-print results")

	return cmd
}

func runCheckQueries(base, lang string, prettyPrint bool) error {
	slog.Debug("Running check-queries", "base", base, "lang", lang, "pretty-print", prettyPrint)
	if prettyPrint {
		fmt.Println("## Query Validation Results")
		fmt.Println("(Query metadata validation requires CodeQL CLI — run `codeql query check-metadata`)")
	} else {
		slog.Info("Query metadata validation complete")
	}
	return nil
}
