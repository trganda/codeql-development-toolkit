package lifecycle

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/query"
)

func newCompileCmd(base *string) *cobra.Command {
	var lang, packName string
	var threads int
	cmd := &cobra.Command{
		Use:   "compile",
		Short: "Compile CodeQL queries",
		Long: `Compile lifecycle phase: compile CodeQL query files (.ql and .qll).

Runs 'codeql query compile' for packs found under <base>, optionally filtered
by language and pack name.

Corresponds to: qlt query compile`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing lifecycle compile", "base", *base, "language", lang, "pack", packName, "threads", threads)
			return query.RunCompile(*base, lang, packName, threads)
		},
	}
	cmd.Flags().StringVar(&lang, "language", "", "Filter by language (e.g. go, java)")
	cmd.Flags().StringVar(&packName, "pack", "", "Filter by pack name (exact match on the pack segment)")
	cmd.Flags().IntVar(&threads, "threads", 0, "Number of threads (0 = use all available cores)")
	return cmd
}
