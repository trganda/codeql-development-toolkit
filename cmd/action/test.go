package action

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/trganda/codeql-development-toolkit/internal/language"
	tmpl "github.com/trganda/codeql-development-toolkit/internal/template"
)

func newTestInitCommand(base *string) *cobra.Command {
	var (
		overwrite  bool
		numThreads int
		useRunner  string
		lang       string
		branch     string
	)

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Initialize test infrastructure (GitHub Actions workflow)",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing test init command", "language", lang, "runner", useRunner, "branch", branch)
			return runTestInit(*base, lang, useRunner, branch, numThreads, overwrite)
		},
	}

	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite existing files")
	cmd.Flags().IntVar(&numThreads, "num-threads", 4, "Number of threads for test execution")
	cmd.Flags().StringVar(&useRunner, "use-runner", "ubuntu-latest", "GitHub Actions runner(s) to use")
	cmd.Flags().StringVar(&lang, "language", "", "Language to initialize CodeQL action for")
	cmd.Flags().StringVar(&branch, "branch", "main", "Branch to trigger automation on")
	cmd.MarkFlagRequired("language")

	return cmd
}

// testInitOptions holds template variables for the test init workflow.
type testInitOptions struct {
	Language   string
	Branch     string
	NumThreads int
	UseRunner  string
	CodeqlArgs string
}

func runTestInit(base, lang, useRunner, branch string, numThreads int, overwrite bool) error {
	slog.Debug("Running test init", "lang", lang, "overwrite", overwrite)
	langDir := language.ToDirectory(lang)

	data := testInitOptions{
		Language:   langDir,
		Branch:     branch,
		NumThreads: numThreads,
		UseRunner:  useRunner,
	}

	// Write install-qlt action
	installTmpl, err := tmpl.Get("shared/actions/install-qlt.tmpl")
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

	slog.Info(`Your repository now has the CodeQL Unit Test Runner installed in .github/workflows/. Additionally, QLT has installed necessary actions in .github/actions/install-qlt.`)

	return nil
}
