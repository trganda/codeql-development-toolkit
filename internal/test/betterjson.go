// Parses and interprets the CodeQL betterjson test output format.
package test

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
)

// Event represents a single event from the CodeQL betterjson output format.
type Event struct {
	Type             string    `json:"type"`
	Test             string    `json:"test,omitempty"`
	TestDirectory    string    `json:"testDirectory,omitempty"`
	DatasetDirectory string    `json:"datasetDirectory,omitempty"`
	Pass             bool      `json:"pass,omitempty"`
	Messages         []Message `json:"messages,omitempty"`
	CompilationMs    int       `json:"compilationMs,omitempty"`
	EvaluationMs     int       `json:"evaluationMs,omitempty"`
	Expected         string    `json:"expected,omitempty"`
	FailureMessage   string    `json:"failureMessage,omitempty"`
}

// Message is an inline diagnostic message attached to a testCompleted event.
type Message struct {
	Message  string `json:"message,omitempty"`
	Severity string `json:"severity,omitempty"`
}

// Summary accumulates pass/fail counts across all events.
type Summary struct {
	Total  int
	Passed int
	Failed int
}

// Parse decodes the betterjson array from raw bytes.
func Parse(data []byte) ([]Event, error) {
	var events []Event
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("failed to parse betterjson output: %w", err)
	}
	return events, nil
}

// LogEvents processes events and emits structured log lines.
// It returns a summary and any test failures as an error.
func LogEvents(events []Event) (Summary, error) {
	var summary Summary
	var failures []string

	for _, e := range events {
		switch e.Type {
		case "testStarted":
			slog.Debug("Test started", "test", shortPath(e.Test))
		case "extractionStarted":
			slog.Debug("Extraction started", "directory", shortPath(e.TestDirectory))
		case "extractionSucceeded":
			slog.Debug("Extraction succeeded",
				"directory", shortPath(e.TestDirectory),
				"dataset", shortPath(e.DatasetDirectory),
			)
		case "testCompleted":
			summary.Total++
			if e.Pass {
				summary.Passed++
				slog.Info("PASS",
					"test", shortPath(e.Test),
					"compilation_ms", e.CompilationMs,
					"evaluation_ms", e.EvaluationMs,
				)
			} else {
				summary.Failed++
				attrs := []any{
					"test", shortPath(e.Test),
					"compilation_ms", e.CompilationMs,
					"evaluation_ms", e.EvaluationMs,
				}
				if e.FailureMessage != "" {
					attrs = append(attrs, "failure", e.FailureMessage)
				}
				for _, m := range e.Messages {
					attrs = append(attrs, "message", m.Message)
				}
				slog.Error("FAIL", attrs...)
				failures = append(failures, shortPath(e.Test))
			}
		default:
			slog.Debug("Test event", "type", e.Type)
		}
	}

	if len(failures) > 0 {
		return summary, fmt.Errorf("%d test(s) failed: %v", len(failures), failures)
	}
	return summary, nil
}

// shortPath returns the base name of a path to keep log lines concise.
func shortPath(p string) string {
	if p == "" {
		return ""
	}
	return filepath.Base(p)
}
