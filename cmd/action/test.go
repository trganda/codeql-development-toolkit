package action

import (
	"fmt"
	"log/slog"
	"os"
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
		PreRun: func(cmd *cobra.Command, args []string) {
			if lang != "all" && !language.IsSupported(lang) {
				slog.Error("Invalid --language", "supported", language.SupportedLanguages, "extra", "all", "got", lang)
				os.Exit(1)
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			slog.Debug("Executing test init command", "language", lang, "runner", useRunner, "branch", branch)
			if err := runTestInit(*base, lang, useRunner, branch, numThreads, overwrite); err != nil {
				slog.Error("Test init failed", "err", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite existing files")
	cmd.Flags().IntVar(&numThreads, "num-threads", 4, "Number of threads for test execution")
	cmd.Flags().StringVar(&useRunner, "use-runner", "ubuntu-latest", "GitHub Actions runner(s) to use")
	cmd.Flags().StringVar(&lang, "language", "", "Pack of target language to test in this action (use the language name, e.g. 'java' or 'python', use 'all' to test all languages)")
	cmd.Flags().StringVar(&branch, "branch", "main", "Branch to trigger automation on")
	cmd.MarkFlagRequired("language")

	return cmd
}

func runTestInit(base, lang, useRunner, branch string, numThreads int, overwrite bool) error {
	slog.Debug("Running test init", "lang", lang, "overwrite", overwrite)

	displayLang := lang
	langFlag := language.ToDirectory(lang)
	if lang == "all" {
		langFlag = ""
	} else {
		displayLang = language.ToDirectory(lang)
	}

	data := tmpl.TestInitOptions{
		Language:   displayLang,
		LangFlag:   langFlag,
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
	if _, statErr := os.Stat(installPath); statErr == nil && !overwrite {
		slog.Info("Skipped file (already exists). Use --overwrite to replace.", "path", installPath)
	}
	if err := tmpl.WriteFile(installTmpl, installPath, nil, overwrite); err != nil {
		return fmt.Errorf("write install-qlt: %w", err)
	}

	// Write run-unit-tests workflow
	workflowTmpl, err := tmpl.Get("test/actions/run-unit-tests.tmpl")
	if err != nil {
		return fmt.Errorf("load run-unit-tests template: %w", err)
	}
	workflowPath := filepath.Join(base, ".github", "workflows", fmt.Sprintf("run-codeql-unit-tests-%s.yml", lang))
	if _, statErr := os.Stat(workflowPath); statErr == nil && !overwrite {
		slog.Info("Skipped file (already exists). Use --overwrite to replace.", "path", workflowPath)
	}
	if err := tmpl.WriteFile(workflowTmpl, workflowPath, data, overwrite); err != nil {
		return fmt.Errorf("write run-unit-tests workflow: %w", err)
	}

	slog.Info(`Your repository now has the CodeQL Unit Test Runner installed in .github/workflows/. Additionally, QLT has installed necessary actions in .github/actions/install-qlt.`)

	return nil
}
