package test

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/config"
	"github.com/trganda/codeql-development-toolkit/internal/executil"
	"github.com/trganda/codeql-development-toolkit/internal/language"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

// newRunUnitTestsCmd returns `test run`.
func newRunUnitTestsCmd(base *string, useBundle *bool) *cobra.Command {
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
			return runExecuteUnitTests(*base, lang, codeqlArgs, numThreads, *useBundle)
		},
	}
	cmd.Flags().IntVar(&numThreads, "num-threads", 4, "Number of threads for test execution")
	cmd.Flags().StringVar(&lang, "language", "", "Language to run tests for")
	cmd.Flags().StringVar(&codeqlArgs, "codeql-args", "", "Extra arguments to pass to CodeQL")
	cmd.MarkFlagRequired("num-threads")
	cmd.MarkFlagRequired("language")
	return cmd
}

func runExecuteUnitTests(base, lang, codeqlArgs string, numThreads int, useBundle bool) error {
	slog.Debug("Running unit tests", "base", base, "lang", lang, "threads", numThreads, "use-bundle", useBundle)
	cfg, err := config.MustLoadFromFile(base)
	if err != nil {
		return err
	}

	slog.Info("Executing unit tests",
		"language", lang,
		"codeql-cli", cfg.CodeQLCLI,
		"threads", numThreads,
		"codeql-args", codeqlArgs,
		"use-bundle", useBundle,
	)

	codeql, err := paths.ResolveCodeQLBinary(base)
	if err != nil {
		return err
	}

	slog.Debug("Using CodeQL binary", "path", codeql)

	args := []string{"resolve", "tests", "--format", "json", fmt.Sprintf("%s/%s", base, language.ToDirectory(lang))}
	runner := executil.NewRunner(codeql)
	res, err := runner.Run(args...)
	if err != nil {
		return fmt.Errorf("failed to resolve tests: %w", err)
	}

	var resolvedTests []string
	if err := json.Unmarshal(res.Stdout, &resolvedTests); err != nil {
		return fmt.Errorf("failed to parse resolved tests JSON: %w", err)
	}

	slog.Info("Resolved test files", "count", len(resolvedTests))

	for _, testFile := range resolvedTests {
		slog.Debug("Running test file", "file", testFile)
		testArgs := []string{"test", "run", "--threads", fmt.Sprintf("%d", numThreads)}
		if codeqlArgs != "" {
			testArgs = append(testArgs, codeqlArgs)
		}
		testArgs = append(testArgs, testFile)
		res, err := runner.Run(testArgs...)
		if err != nil {
			if res != nil && len(res.Stderr) > 0 {
				slog.Error("Test command stderr result", "output", res.StderrString())
			}
			return fmt.Errorf("failed to run test %s: %w", testFile, err)
		}
		if res != nil && len(res.Stdout) > 0 {
			slog.Debug("Test command stdout result", "output", res.StdoutString())
		}
	}

	slog.Info("Completed execution of all unit tests")

	return nil
}
