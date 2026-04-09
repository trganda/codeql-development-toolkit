package test

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/config"
)

// newExecuteUnitTestsCmd returns `test run execute-unit-tests`.
func newExecuteUnitTestsCmd(base *string, useBundle *bool) *cobra.Command {
	var (
		numThreads int
		workDir    string
		lang       string
		runnerOS   string
		codeqlArgs string
	)
	cmd := &cobra.Command{
		Use:   "execute-unit-tests",
		Short: "Run CodeQL unit tests",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing test run execute-unit-tests command",
				"language", lang, "runner-os", runnerOS, "threads", numThreads)
			return runExecuteUnitTests(*base, lang, runnerOS, workDir, codeqlArgs, numThreads, *useBundle)
		},
	}
	cmd.Flags().IntVar(&numThreads, "num-threads", 4, "Number of threads for test execution")
	cmd.Flags().StringVar(&workDir, "work-dir", os.TempDir(), "Directory for intermediate output files")
	cmd.Flags().StringVar(&lang, "language", "", "Language to run tests for")
	cmd.Flags().StringVar(&runnerOS, "runner-os", "", "Operating system label")
	cmd.Flags().StringVar(&codeqlArgs, "codeql-args", "", "Extra arguments to pass to CodeQL")
	_ = cmd.MarkFlagRequired("num-threads")
	_ = cmd.MarkFlagRequired("work-dir")
	_ = cmd.MarkFlagRequired("language")
	_ = cmd.MarkFlagRequired("runner-os")
	return cmd
}

func runExecuteUnitTests(base, lang, runnerOS, workDir, codeqlArgs string, numThreads int, useBundle bool) error {
	slog.Debug("Running execute-unit-tests", "base", base, "lang", lang, "runner-os", runnerOS, "threads", numThreads, "use-bundle", useBundle)
	cfg, err := config.MustLoadFromFile(base)
	if err != nil {
		return err
	}

	slog.Info("Executing unit tests",
		"language", lang,
		"codeql-cli", cfg.CodeQLCLI,
		"threads", numThreads,
		"runner-os", runnerOS,
		"work-dir", workDir,
		"codeql-args", codeqlArgs,
		"use-bundle", useBundle,
	)
	return nil
}
