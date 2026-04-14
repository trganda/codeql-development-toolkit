package phase

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/bundle"
)

func newPackageCmd(base *string, common *commonFlags) *cobra.Command {
	var (
		bundlePath   string
		output       string
		platforms    []string
		noPrecompile bool
	)
	cmd := &cobra.Command{
		Use:   "package",
		Short: "Create a custom CodeQL bundle",
		Long: `Package phase: create a custom CodeQL bundle.

Runs the full chain: install → compile → test → verify → package.
Requires workspace initialization (run 'qlt phase init' first).

Reads packs from qlt.conf.json where Bundle=true and builds a custom bundle
using the base bundle archive downloaded by 'qlt codeql install'.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing phase package", "base", *base, "language", common.language)
			if err := runVerifyChain(*base, common); err != nil {
				return err
			}
			return runPackage(*base, bundlePath, output, platforms, noPrecompile)
		},
	}
	cmd.Flags().StringVar(&bundlePath, "bundle", "", "Override base bundle archive path (.tar.gz)")
	cmd.Flags().StringVar(&output, "output", "", "Override output path (.tar.gz)")
	cmd.Flags().StringArrayVar(&platforms, "platform", nil, "Target platform: linux64, osx64, win64 (repeatable)")
	cmd.Flags().BoolVar(&noPrecompile, "no-precompile", false, "Skip pre-compilation when bundling packs")
	return cmd
}

// runPackage loads config, resolves paths, and delegates to bundle.Create.
func runPackage(base, bundlePath, output string, platforms []string, noPrecompile bool) error {
	opts, err := bundle.NewCreateOptions(base, bundlePath, noPrecompile, false, platforms)
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
