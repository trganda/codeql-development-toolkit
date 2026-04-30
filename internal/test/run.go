package test

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/trganda/codeql-development-toolkit/internal/codeql"
	"github.com/trganda/codeql-development-toolkit/internal/config"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

// RunUnitTests resolves and runs all CodeQL unit tests for the given language
// under base, reporting a summary via slog. A TestReport is written to disk
// only when output is non-empty:
//   - output == ""         → <base>/target/test/test-report-<timestamp>.json
//   - output != ""         → the caller-supplied path
func RunUnitTests(base, codeqlArgs string, output string, numThreads int) error {
	cfg := config.MustLoadFromFile(base)

	slog.Info("Executing unit tests",
		"codeql-cli", cfg.CodeQLCLIVersion,
		"threads", numThreads,
		"codeql-args", codeqlArgs,
	)

	codeqlBin, err := paths.ResolveCodeQLBinary(base)
	if err != nil {
		return err
	}

	slog.Debug("Using CodeQL binary", "path", codeqlBin)

	cli := codeql.NewCLI(codeqlBin)
	res, err := cli.ResolveTests(base)
	if err != nil {
		return fmt.Errorf("failed to resolve tests: %w", err)
	}

	var resolvedTests []string
	if err := json.Unmarshal(res.Stdout, &resolvedTests); err != nil {
		return fmt.Errorf("failed to parse resolved tests JSON: %w", err)
	}

	slog.Info("Resolved test files", "count", len(resolvedTests))

	start := time.Now().UTC()
	var results []TestResult

	for _, testFile := range resolvedTests {
		slog.Debug("Running test file", "file", testFile)
		res, err := cli.TestRun(numThreads, codeqlArgs, testFile)
		if err != nil {
			stderr := ""
			if res != nil {
				stderr = strings.TrimSpace(res.StderrString())
			}
			if stderr != "" {
				slog.Error("Test failed", "file", testFile, "output", stderr)
			}
			results = append(results, TestResult{
				Name: filepath.Base(testFile),
				Path: testFile,
				Pass: false,
				Messages: []TestMessage{{
					Severity: "error",
					Message:  stderr,
				}},
			})
			continue
		}
		if res == nil || len(res.Stdout) == 0 {
			continue
		}
		parsed, parseErr := ParseResults(res.Stdout)
		if parseErr != nil {
			slog.Warn("Could not parse codeql test json, dumping raw stdout", "error", parseErr, "output", res.StdoutString())
			continue
		}
		results = append(results, parsed...)
	}

	passed, failed := LogResults(results)
	summary := ReportSummary{
		Total:      len(results),
		Passed:     passed,
		Failed:     failed,
		DurationMs: time.Since(start).Milliseconds(),
	}

	slog.Info("Completed execution of all unit tests",
		"total", summary.Total,
		"passed", summary.Passed,
		"failed", summary.Failed,
	)

	if output == "" {
		return nil // skip writing report if no output path provided
	}

	output, err = filepath.Abs(output)
	if err != nil {
		return fmt.Errorf("resolve absolute path for report output: %w", err)
	}

	report := &TestReport{
		Metadata: ReportMetadata{
			Timestamp:  start,
			NumThreads: numThreads,
		},
		Summary: summary,
		Results: results,
	}
	if err := WriteReport(output, report); err != nil {
		return fmt.Errorf("writing test report: %w", err)
	}
	slog.Info("Wrote test report", "path", output)

	return nil
}
