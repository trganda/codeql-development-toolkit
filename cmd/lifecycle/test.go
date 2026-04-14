package lifecycle

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/query"
	qlttest "github.com/trganda/codeql-development-toolkit/internal/test"
	"github.com/trganda/codeql-development-toolkit/internal/utils"
)

func newTestCmd(base *string) *cobra.Command {
	var lang, codeqlArgs string
	var numThreads int
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run CodeQL unit tests",
		Long: `Test lifecycle phase: run CodeQL unit tests for a language.

Runs the full chain: install → compile → test.
Requires workspace initialization (run 'qlt lifecycle init' first).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing lifecycle test", "base", *base, "language", lang, "threads", numThreads)
			if err := utils.CheckWorkspace(*base); err != nil {
				return err
			}
			if err := query.RunPackInstall(*base, lang); err != nil {
				return err
			}
			if err := query.RunCompile(*base, lang, "", 0); err != nil {
				return err
			}
			return qlttest.RunUnitTests(*base, lang, codeqlArgs, numThreads)
		},
	}
	cmd.Flags().StringVar(&lang, "language", "", "Language to test (e.g. go, java)")
	cmd.Flags().StringVar(&codeqlArgs, "codeql-args", "", "Extra arguments to pass to CodeQL")
	cmd.Flags().IntVar(&numThreads, "num-threads", 4, "Number of threads for test execution")
	cmd.MarkFlagRequired("language")
	return cmd
}
