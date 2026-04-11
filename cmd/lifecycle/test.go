package lifecycle

import (
	"log/slog"

	"github.com/spf13/cobra"

	qlttest "github.com/trganda/codeql-development-toolkit/internal/test"
)

func newTestCmd(base *string) *cobra.Command {
	var lang, codeqlArgs string
	var numThreads int
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run CodeQL unit tests",
		Long: `Test lifecycle phase: run CodeQL unit tests for a language.

Resolves and executes all .qlref test files found under <base>/<language>
using 'codeql test run', reporting a pass/fail summary.

Corresponds to: qlt test run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing lifecycle test", "base", *base, "language", lang, "threads", numThreads)
			return qlttest.RunUnitTests(*base, lang, codeqlArgs, numThreads)
		},
	}
	cmd.Flags().StringVar(&lang, "language", "", "Language to test (e.g. go, java)")
	cmd.Flags().StringVar(&codeqlArgs, "codeql-args", "", "Extra arguments to pass to CodeQL")
	cmd.Flags().IntVar(&numThreads, "num-threads", 4, "Number of threads for test execution")
	cmd.MarkFlagRequired("language")
	return cmd
}
