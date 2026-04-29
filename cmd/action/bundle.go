package action

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	tmpl "github.com/trganda/codeql-development-toolkit/internal/template"
)

func newInitBundleTestCmd(base *string) *cobra.Command {
	var (
		langs             []string
		branch            string
		overwriteExisting bool
	)
	cmd := &cobra.Command{
		Use:   "bundle-test",
		Short: "Initialize bundle support",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing bundle init command", "base", *base, "languages", langs, "branch", branch)
			return runBundleInit(*base, langs, branch, overwriteExisting)
		},
	}
	cmd.Flags().StringArrayVar(&langs, "language", nil, "Language(s) for bundle (repeat for multiple, e.g. --language java --language cpp)")
	cmd.Flags().StringVar(&branch, "branch", "main", "Branch to trigger automation on")
	cmd.Flags().BoolVar(&overwriteExisting, "overwrite", false, "Overwrite existing files")
	cmd.MarkFlagRequired("language")
	return cmd
}

// bundleInitData holds template variables for bundle init.
type bundleInitData struct {
	Languages []string
	Branch    string
}

func runBundleInit(base string, langs []string, branch string, overwrite bool) error {
	slog.Debug("Running bundle init", "base", base, "langs", langs, "branch", branch)
	data := bundleInitData{
		Languages: langs,
		Branch:    branch,
	}

	// Write install-qlt action
	installTmpl, err := tmpl.Get("shared/actions/install-qlt.tmpl")
	if err != nil {
		return fmt.Errorf("load install-qlt template: %w", err)
	}
	installPath := filepath.Join(base, ".github", "actions", "install-qlt", "action.yml")
	if _, statErr := os.Stat(installPath); statErr == nil && !overwrite {
		slog.Info("Skipped file (already exists). Use --overwrite-existing to replace.", "path", installPath)
	}
	if err := tmpl.WriteFile(installTmpl, installPath, nil, overwrite); err != nil {
		return fmt.Errorf("write install-qlt: %w", err)
	}

	// Write bundle integration test workflow
	bundleTmpl, err := tmpl.Get("bundle/actions/run-bundle-integration-tests.tmpl")
	if err != nil {
		return fmt.Errorf("load bundle template: %w", err)
	}
	bundlePath := filepath.Join(base, ".github", "workflows", "run-bundle-integration-tests.yml")
	if _, statErr := os.Stat(bundlePath); statErr == nil && !overwrite {
		slog.Info("Skipped file (already exists). Use --overwrite-existing to replace.", "path", bundlePath)
	}
	if err := tmpl.WriteFile(bundleTmpl, bundlePath, data, overwrite); err != nil {
		return fmt.Errorf("write bundle workflow: %w", err)
	}

	slog.Info("Bundle integration test workflow initialized", "base", base)
	return nil
}
