package test

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// newValidateUnitTestsCmd returns `test run validate-unit-tests`.
func newValidateUnitTestsCmd() *cobra.Command {
	var (
		resultsDir  string
		prettyPrint bool
	)
	cmd := &cobra.Command{
		Use:   "validate-unit-tests",
		Short: "Validate unit test results for CI/CD",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing test run validate-unit-tests command", "results-dir", resultsDir)
			return runValidateUnitTests(resultsDir, prettyPrint)
		},
	}
	cmd.Flags().StringVar(&resultsDir, "results-directory", "", "Directory containing test result files")
	cmd.Flags().BoolVar(&prettyPrint, "pretty-print", false, "Pretty-print results (no failure exit code)")
	_ = cmd.MarkFlagRequired("results-directory")
	return cmd
}

// testResult is a simplified test result structure.
type testResult struct {
	Passed  int    `json:"passed"`
	Failed  int    `json:"failed"`
	Message string `json:"message,omitempty"`
}

func runValidateUnitTests(resultsDir string, prettyPrint bool) error {
	slog.Debug("Running validate-unit-tests", "results-dir", resultsDir, "pretty-print", prettyPrint)
	entries, err := os.ReadDir(resultsDir)
	if err != nil {
		return fmt.Errorf("read results directory: %w", err)
	}

	var totalPassed, totalFailed int
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(resultsDir, entry.Name()))
		if err != nil {
			continue
		}
		var result testResult
		if err := json.Unmarshal(data, &result); err != nil {
			continue
		}
		totalPassed += result.Passed
		totalFailed += result.Failed
	}

	if prettyPrint {
		fmt.Printf("## Test Results\n\n| Status | Count |\n|--------|-------|\n| Passed | %d |\n| Failed | %d |\n", totalPassed, totalFailed)
		return nil
	}

	slog.Info("Validated unit tests", "passed", totalPassed, "failed", totalFailed)
	if totalFailed > 0 {
		return fmt.Errorf("%d test(s) failed", totalFailed)
	}
	fmt.Printf("All %d test(s) passed\n", totalPassed)
	return nil
}
