package test

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// TestReport is the qlt-owned artifact produced by RunUnitTests.
// It is stable within qlt and decoupled from codeql's CLI output format.
type TestReport struct {
	Metadata ReportMetadata `json:"metadata"`
	Summary  ReportSummary  `json:"summary"`
	Results  []TestResult   `json:"results"`
}

type ReportMetadata struct {
	Timestamp  time.Time `json:"timestamp"`
	Language   string    `json:"language"`
	NumThreads int       `json:"numThreads"`
}

type ReportSummary struct {
	Total      int   `json:"total"`
	Passed     int   `json:"passed"`
	Failed     int   `json:"failed"`
	DurationMs int64 `json:"durationMs"`
}

type TestResult struct {
	Name          string        `json:"name"`
	Path          string        `json:"path"`
	Pass          bool          `json:"pass"`
	CompilationMs int           `json:"compilationMs"`
	EvaluationMs  int           `json:"evaluationMs"`
	Expected      string        `json:"expected,omitempty"`
	Messages      []TestMessage `json:"messages,omitempty"`
}

type TestMessage struct {
	Severity string `json:"severity,omitempty"`
	Message  string `json:"message,omitempty"`
}

// rawTestCase matches codeql --format=json output one-to-one. Unexported so
// the wire format stays an implementation detail.
type rawTestCase struct {
	Test          string        `json:"test"`
	Pass          bool          `json:"pass"`
	Messages      []TestMessage `json:"messages"`
	CompilationMs int           `json:"compilationMs"`
	EvaluationMs  int           `json:"evaluationMs"`
	Expected      string        `json:"expected"`
}

// ParseResults decodes codeql's --format=json output into qlt TestResult values.
func ParseResults(data []byte) ([]TestResult, error) {
	var raw []rawTestCase
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing codeql test json: %w", err)
	}
	results := make([]TestResult, 0, len(raw))
	for _, r := range raw {
		results = append(results, TestResult{
			Name:          filepath.Base(r.Test),
			Path:          r.Test,
			Pass:          r.Pass,
			CompilationMs: r.CompilationMs,
			EvaluationMs:  r.EvaluationMs,
			Expected:      r.Expected,
			Messages:      r.Messages,
		})
	}
	return results, nil
}

// LogResults emits one PASS/FAIL slog line per result and returns the counts.
func LogResults(results []TestResult) (passed, failed int) {
	for _, r := range results {
		if r.Pass {
			passed++
			slog.Info("PASS",
				"test", r.Name,
				"compilation_ms", r.CompilationMs,
				"evaluation_ms", r.EvaluationMs,
			)
			continue
		}
		failed++
		attrs := []any{
			"test", r.Name,
			"compilation_ms", r.CompilationMs,
			"evaluation_ms", r.EvaluationMs,
		}
		for _, m := range r.Messages {
			attrs = append(attrs, "message", m.Message)
		}
		slog.Error("FAIL", attrs...)
	}
	return passed, failed
}

// WriteReport writes r to path as indented JSON, creating parent directories as needed.
func WriteReport(path string, r *TestReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating report directory: %w", err)
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling report: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing report: %w", err)
	}
	return nil
}
