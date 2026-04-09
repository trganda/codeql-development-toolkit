package test

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/config"
	"github.com/trganda/codeql-development-toolkit/internal/language"
	tmpl "github.com/trganda/codeql-development-toolkit/internal/template"
)

// NewCommand returns the `test` cobra command.
func NewCommand(base, automationType *string, development, useBundle *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Unit testing commands",
	}
	cmd.AddCommand(newInitCmd(base, development))
	cmd.AddCommand(newRunCmd(base, development, useBundle))
	return cmd
}

func newInitCmd(base *string, development *bool) *cobra.Command {
	var (
		overwriteExisting bool
		numThreads        int
		useRunner         string
		lang              string
		branch            string
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize test infrastructure (GitHub Actions workflow)",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing test init command", "language", lang, "runner", useRunner, "branch", branch)
			return runTestInit(*base, *development, lang, useRunner, branch, numThreads, overwriteExisting)
		},
	}

	cmd.Flags().BoolVar(&overwriteExisting, "overwrite-existing", false, "Overwrite existing files")
	cmd.Flags().IntVar(&numThreads, "num-threads", 4, "Number of threads for test execution")
	cmd.Flags().StringVar(&useRunner, "use-runner", "ubuntu-latest", "GitHub Actions runner(s) to use")
	cmd.Flags().StringVar(&lang, "language", "", "Language to generate automation for")
	cmd.Flags().StringVar(&branch, "branch", "main", "Branch to trigger automation on")
	_ = cmd.MarkFlagRequired("language")
	return cmd
}

func newRunCmd(base *string, development *bool, useBundle *bool) *cobra.Command {
	run := &cobra.Command{
		Use:   "run",
		Short: "Run test-related commands",
	}
	run.AddCommand(newGetMatrixCmd(base))
	run.AddCommand(newExecuteUnitTestsCmd(base, useBundle))
	run.AddCommand(newValidateUnitTestsCmd())
	return run
}

func newGetMatrixCmd(base *string) *cobra.Command {
	var osVersion string
	cmd := &cobra.Command{
		Use:   "get-matrix",
		Short: "Get a CI/CD matrix based on the current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing test run get-matrix command", "base", *base, "os-version", osVersion)
			return runGetMatrix(*base, osVersion)
		},
	}
	cmd.Flags().StringVar(&osVersion, "os-version", "ubuntu-latest", "Operating system(s) to use (comma-separated)")
	return cmd
}

func newExecuteUnitTestsCmd(base *string, useBundle *bool) *cobra.Command {
	var (
		numThreads  int
		workDir     string
		lang        string
		runnerOS    string
		codeqlArgs  string
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

// testInitData holds template variables for the test init workflow.
type testInitData struct {
	Language   string
	Branch     string
	NumThreads int
	UseRunner  string
	CodeqlArgs string
	DevMode    bool
}

func runTestInit(base string, devMode bool, lang, useRunner, branch string, numThreads int, overwrite bool) error {
	slog.Debug("Running test init", "lang", lang, "devMode", devMode, "overwrite", overwrite)
	langDir := language.ToDirectory(lang)

	data := testInitData{
		Language:   langDir,
		Branch:     branch,
		NumThreads: numThreads,
		UseRunner:  useRunner,
		CodeqlArgs: "--threads=0",
		DevMode:    devMode,
	}

	// Write install-qlt action
	installTmpl, err := tmpl.Get("test/actions/install-qlt.tmpl")
	if err != nil {
		return fmt.Errorf("load install-qlt template: %w", err)
	}
	installPath := filepath.Join(base, ".github", "actions", "install-qlt", "action.yml")
	if err := tmpl.WriteFile(installTmpl, installPath, nil, overwrite); err != nil {
		return fmt.Errorf("write install-qlt: %w", err)
	}

	// Write run-unit-tests workflow
	workflowTmpl, err := tmpl.Get("test/actions/run-unit-tests.tmpl")
	if err != nil {
		return fmt.Errorf("load run-unit-tests template: %w", err)
	}
	workflowPath := filepath.Join(base, ".github", "workflows", fmt.Sprintf("run-codeql-unit-tests-%s.yml", lang))
	if err := tmpl.WriteFile(workflowTmpl, workflowPath, data, overwrite); err != nil {
		return fmt.Errorf("write run-unit-tests workflow: %w", err)
	}

	fmt.Printf(`------------------------------------------
Your repository now has the CodeQL Unit Test Runner installed in .github/workflows/.
Additionally, QLT has installed necessary actions in .github/actions/install-qlt.

Note: runner will use %d threads and the %q runner by default.
(Hint: use --overwrite-existing to regenerate files.)
`, numThreads, useRunner)
	return nil
}

// matrixEntry is a single matrix entry for GitHub Actions.
type matrixEntry struct {
	OS         string `json:"os"`
	CodeQLCLI  string `json:"codeql_cli"`
}

func runGetMatrix(base, osVersions string) error {
	slog.Debug("Running get-matrix", "base", base, "os-versions", osVersions)
	cfg, err := config.LoadFromFile(base)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	cliVersion := "latest"
	if cfg != nil && cfg.CodeQLCLI != "" {
		cliVersion = cfg.CodeQLCLI
	}

	var entries []matrixEntry
	for _, os := range strings.Split(osVersions, ",") {
		os = strings.TrimSpace(os)
		if os == "" {
			continue
		}
		entries = append(entries, matrixEntry{OS: os, CodeQLCLI: cliVersion})
	}

	matrix := map[string]any{"include": entries}
	out, err := json.Marshal(matrix)
	if err != nil {
		return fmt.Errorf("marshal matrix: %w", err)
	}

	slog.Info("Generated matrix", "entries", len(entries))
	fmt.Printf("matrix=%s\n", string(out))
	return nil
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
