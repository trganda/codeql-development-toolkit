package test

import (
	"log/slog"

	"github.com/spf13/cobra"

	qlttest "github.com/trganda/codeql-development-toolkit/internal/test"
)

// newRunUnitTestsCmd returns `test run`.
func newRunUnitTestsCmd(base *string) *cobra.Command {
	var (
		numThreads int
		lang       string
		codeqlArgs string
	)
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run CodeQL unit tests",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing test run execute command",
				"language", lang, "threads", numThreads)
			return qlttest.RunUnitTests(*base, lang, codeqlArgs, numThreads)
		},
	}
	cmd.Flags().IntVar(&numThreads, "num-threads", 4, "Number of threads for test execution")
	cmd.Flags().StringVar(&lang, "language", "", "Language to run tests for")
	cmd.Flags().StringVar(&codeqlArgs, "codeql-args", "", "Extra arguments to pass to CodeQL")
	cmd.MarkFlagRequired("language")
	return cmd
}
