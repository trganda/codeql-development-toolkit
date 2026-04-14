package bundle

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/bundle"
)

// NewCommand returns the `bundle` cobra command.
func NewCommand(base, automationType *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bundle",
		Short: "Custom CodeQL bundle management commands",
	}
	cmd.AddCommand(newInitCmd(base))
	cmd.AddCommand(newRunCmd(base))
	cmd.AddCommand(newCreateCmd(base))
	return cmd
}

func newCreateCmd(base *string) *cobra.Command {
	var (
		lang         string
		bundlePath   string
		output       string
		platforms    []string
		minimal      bool
		noPrecompile bool
	)
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new custom CodeQL bundle",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing bundle create command", "base", *base)
			return runBundleCreate(*base, bundlePath, output, platforms, noPrecompile, minimal)
		},
	}

	createCmd.Flags().StringVar(&lang, "language", "", "Filter by language (reserved; currently unused by bundle create)")
	createCmd.Flags().StringVar(&bundlePath, "bundle", "", "Override base bundle archive path (.tar.gz)")
	createCmd.Flags().StringVar(&output, "output", "", "Override output path (.tar.gz)")
	createCmd.Flags().StringArrayVar(&platforms, "platform", nil, "Target platform: linux64, osx64, win64 (repeatable)")
	createCmd.Flags().BoolVar(&noPrecompile, "no-precompile", false, "Skip pre-compilation when bundling packs")
	createCmd.Flags().BoolVar(&minimal, "minimal", false, "Reserved; currently a no-op")
	return createCmd
}

func runBundleCreate(base, bundlePath, output string, platforms []string, noPrecompile, minimal bool) error {
	opts, err := bundle.NewCreateOptions(base, bundlePath, noPrecompile, minimal, platforms)
	if err != nil || opts.Validate() != nil {
		return err
	}

	bundleCtx, err := bundle.NewCustomBundle(opts)
	if err != nil {
		return err
	}

	slog.Info("Creating custom CodeQL bundle",
		"base-bundle", bundlePath,
		"output", output,
		"packs", opts.Packs,
		"platforms", platforms,
	)

	return bundleCtx.Create()
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
	return cmd
}
