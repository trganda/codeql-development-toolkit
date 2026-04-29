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
	"github.com/trganda/codeql-development-toolkit/internal/language"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

// RunUnitTests resolves and runs all CodeQL unit tests for the given language
// under base, reporting a summary via slog. A TestReport is written to disk
// only when reportOutput is non-nil:
//   - reportOutput == nil         → no report written
//   - *reportOutput == ""         → <base>/target/test/test-report-<timestamp>.json
//   - *reportOutput != ""         → the caller-supplied path
func RunUnitTests(base, lang, codeqlArgs string, reportOutput *string, numThreads int) error {
	cfg := config.MustLoadFromFile(base)

	slog.Info("Executing unit tests",
		"language", lang,
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
	testDir := base
	if lang != "" && strings.ToLower(lang) != "all" {
		testDir = filepath.Join(base, language.ToDirectory(lang))
	}
	res, err := cli.ResolveTests(testDir)
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

	if reportOutput == nil {
		return nil
	}
	outputPath := *reportOutput
	if outputPath == "" {
		name := fmt.Sprintf("test-report-%s.json", start.Format("20060102T150405Z"))
		outputPath = filepath.Join(base, "target", "test", name)
	}

	report := &TestReport{
		Metadata: ReportMetadata{
			Timestamp:  start,
			Language:   lang,
			NumThreads: numThreads,
		},
		Summary: summary,
		Results: results,
	}
	if err := WriteReport(outputPath, report); err != nil {
		return fmt.Errorf("writing test report: %w", err)
	}
	slog.Info("Wrote test report", "path", outputPath)

	return nil
}
