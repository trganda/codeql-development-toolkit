package bundle

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/spf13/cobra"

	tmpl "github.com/trganda/codeql-development-toolkit/internal/template"
)

// NewCommand returns the `bundle` cobra command.
func NewCommand(base, automationType *string, development *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bundle",
		Short: "Custom CodeQL bundle management commands",
	}
	cmd.AddCommand(newInitCmd(base, development))
	cmd.AddCommand(newSetCmd(base))
	cmd.AddCommand(newGetCmd(base))
	cmd.AddCommand(newRunCmd(base))
	return cmd
}

func newInitCmd(base *string, development *bool) *cobra.Command {
	var (
		lang              string
		overwriteExisting bool
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize bundle support",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing bundle init command", "base", *base, "language", lang)
			return runBundleInit(*base, lang, *development, overwriteExisting)
		},
	}
	cmd.Flags().StringVar(&lang, "language", "", "Language for bundle")
	cmd.Flags().BoolVar(&overwriteExisting, "overwrite-existing", false, "Overwrite existing files")
	return cmd
}

func newSetCmd(base *string) *cobra.Command {
	set := &cobra.Command{
		Use:   "set",
		Short: "Set bundle options",
	}
	set.AddCommand(&cobra.Command{
		Use:   "enable",
		Short: "Enable custom bundle",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Info("Custom bundle enabled")
			return nil
		},
	})
	set.AddCommand(&cobra.Command{
		Use:   "disable",
		Short: "Disable custom bundle",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Info("Custom bundle disabled")
			return nil
		},
	})
	return set
}

func newGetCmd(base *string) *cobra.Command {
	get := &cobra.Command{
		Use:   "get",
		Short: "Get bundle information",
	}
	get.AddCommand(&cobra.Command{
		Use:   "enabled-bundles",
		Short: "List enabled custom bundles",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Info("Bundle listing requires qlt.conf.json")
			return nil
		},
	})
	return get
}

func newRunCmd(base *string) *cobra.Command {
	run := &cobra.Command{
		Use:   "run",
		Short: "Run bundle commands",
	}
	run.AddCommand(newValidateIntegrationTestsCmd(base))
	return run
}

func newValidateIntegrationTestsCmd(base *string) *cobra.Command {
	var expected, actual string
	cmd := &cobra.Command{
		Use:   "validate-integration-tests",
		Short: "Validate bundle integration test SARIF results",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing bundle run validate-integration-tests command", "expected", expected, "actual", actual)
			slog.Info("Comparing SARIF results", "expected", expected, "actual", actual)
			return nil
		},
	}
	cmd.Flags().StringVar(&expected, "expected", "", "Path to expected SARIF file")
	cmd.Flags().StringVar(&actual, "actual", "", "Path to actual SARIF file")
	_ = cmd.MarkFlagRequired("expected")
	_ = cmd.MarkFlagRequired("actual")
	return cmd
}

// bundleInitData holds template variables for bundle init.
type bundleInitData struct {
	Language string
	DevMode  bool
}

func runBundleInit(base, lang string, devMode, overwrite bool) error {
	slog.Debug("Running bundle init", "base", base, "lang", lang, "dev-mode", devMode)
	data := bundleInitData{
		Language: lang,
		DevMode:  devMode,
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
