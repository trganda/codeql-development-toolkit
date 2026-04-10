package bundle

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/spf13/cobra"
	tmpl "github.com/trganda/codeql-development-toolkit/internal/template"
)

func newInitCmd(base *string) *cobra.Command {
	var (
		lang              string
		overwriteExisting bool
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize bundle support",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing bundle init command", "base", *base, "language", lang)
			return runBundleInit(*base, lang, overwriteExisting)
		},
	}
	cmd.Flags().StringVar(&lang, "language", "", "Language for bundle")
	cmd.Flags().BoolVar(&overwriteExisting, "overwrite-existing", false, "Overwrite existing files")
	return cmd
}

// bundleInitData holds template variables for bundle init.
type bundleInitData struct {
	Language string
}

func runBundleInit(base, lang string, overwrite bool) error {
	slog.Debug("Running bundle init", "base", base, "lang", lang)
	data := bundleInitData{
		Language: lang,
	}

	// Write install-qlt action
	installTmpl, err := tmpl.Get("bundle/actions/install-qlt.tmpl")
	if err != nil {
		return fmt.Errorf("load install-qlt template: %w", err)
	}
	installPath := filepath.Join(base, ".github", "actions", "install-qlt", "action.yml")
	if err := tmpl.WriteFile(installTmpl, installPath, nil, overwrite); err != nil {
		return fmt.Errorf("write install-qlt: %w", err)
	}

	// Write bundle integration test workflow
	bundleTmpl, err := tmpl.Get("bundle/actions/run-bundle-integration-tests.tmpl")
	if err != nil {
		return fmt.Errorf("load bundle template: %w", err)
	}
	bundlePath := filepath.Join(base, ".github", "workflows", "run-bundle-integration-tests.yml")
	if err := tmpl.WriteFile(bundleTmpl, bundlePath, data, overwrite); err != nil {
		return fmt.Errorf("write bundle workflow: %w", err)
	}

	slog.Info("Bundle integration test workflow initialized", "base", base)
	return nil
}
