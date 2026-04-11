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

Runs the full chain: install → compile.
Requires workspace initialization (run 'qlt lifecycle init' first).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing lifecycle compile", "base", *base, "language", lang, "pack", packName, "threads", threads)
			if err := checkWorkspace(*base); err != nil {
				return err
			}
			if err := runInstallStep(*base, lang, packName); err != nil {
				return err
			}
			return query.RunCompile(*base, lang, packName, threads)
		},
	}
	cmd.Flags().StringVar(&lang, "language", "", "Filter by language (e.g. go, java)")
	cmd.Flags().StringVar(&packName, "pack", "", "Filter by pack name (exact match on the pack segment)")
	cmd.Flags().IntVar(&threads, "threads", 0, "Number of threads (0 = use all available cores)")
	return cmd
}
