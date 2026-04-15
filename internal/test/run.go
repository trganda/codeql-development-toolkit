package test

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/trganda/codeql-development-toolkit/internal/codeql"
	"github.com/trganda/codeql-development-toolkit/internal/config"
	"github.com/trganda/codeql-development-toolkit/internal/language"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

// RunUnitTests resolves and runs all CodeQL unit tests for the given language
// under base, reporting a summary via slog.
func RunUnitTests(base, lang, codeqlArgs string, numThreads int) error {
	cfg := config.MustLoadFromFile(base)

	slog.Info("Executing unit tests",
		"language", lang,
		"codeql-cli", cfg.CodeQLCLI,
		"threads", numThreads,
		"codeql-args", codeqlArgs,
	)

	codeqlBin, err := paths.ResolveCodeQLBinary(base)
	if err != nil {
		return err
	}

	slog.Debug("Using CodeQL binary", "path", codeqlBin)

	cli := codeql.NewCLI(codeqlBin)
	res, err := cli.ResolveTests(fmt.Sprintf("%s/%s", base, language.ToDirectory(lang)))
	if err != nil {
		return fmt.Errorf("failed to resolve tests: %w", err)
	}

	var resolvedTests []string
	if err := json.Unmarshal(res.Stdout, &resolvedTests); err != nil {
		return fmt.Errorf("failed to parse resolved tests JSON: %w", err)
	}

	slog.Info("Resolved test files", "count", len(resolvedTests))

	overall := Summary{}

	for _, testFile := range resolvedTests {
		slog.Debug("Running test file", "file", testFile)
		res, err := cli.TestRun(numThreads, codeqlArgs, testFile)
		if err != nil {
			if res != nil && len(res.Stderr) > 0 {
				slog.Error("Test failed", "file", testFile, "output", strings.TrimSpace(res.StderrString()))
			}
			overall.Total++
			overall.Failed++
			continue
		}
		if res == nil || len(res.Stdout) == 0 {
			continue
		}
		events, parseErr := Parse(res.Stdout)
		if parseErr != nil {
			slog.Warn("Could not parse betterjson output, dumping raw stdout", "error", parseErr, "output", res.StdoutString())
			continue
		}
		summary, testErr := LogEvents(events)
		overall.Total += summary.Total
		overall.Passed += summary.Passed
		overall.Failed += summary.Failed
		if testErr != nil {
			slog.Warn("Test file had failures", "file", testFile, "error", testErr)
		}
	}

	slog.Info("Completed execution of all unit tests",
		"total", overall.Total,
		"passed", overall.Passed,
		"failed", overall.Failed,
	)

	return nil
}
