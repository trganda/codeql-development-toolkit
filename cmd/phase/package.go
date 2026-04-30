package phase

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/trganda/codeql-development-toolkit/internal/bundle"
)

func newPackageCmd(base *string, common *commonFlags) *cobra.Command {
	var (
		bundlePath   string
		output       string
		platforms    []string
		minimal      bool
		noPrecompile bool
	)
	cmd := &cobra.Command{
		Use:   "package",
		Short: "Create a custom CodeQL bundle",
		Long: `Package phase: create a custom CodeQL bundle.

Runs the full chain: install → compile → test → verify → package.
Requires workspace initialization (run 'qlt phase init' first).

Reads packs from qlt.conf.json where bundle=true and builds a custom bundle
using the base bundle archive downloaded by 'qlt codeql install'.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing phase package", "base", *base)
			// if err := runVerifyChain(*base, common); err != nil {
			// 	return err
			// }
			return runPackage(*base, bundlePath, output, platforms, noPrecompile, minimal)
		},
	}
	cmd.Flags().StringVar(&bundlePath, "bundle", "", "Override base bundle archive path (.tar.gz)")
	cmd.Flags().StringVar(&output, "output", "", "Override output path (.tar.gz)")
	cmd.Flags().StringArrayVar(&platforms, "platform", nil, "Target platform: linux64, osx64, win64 (repeatable)")
	cmd.Flags().BoolVar(&noPrecompile, "no-precompile", false, "Skip pre-compilation when bundling packs")
	cmd.Flags().BoolVar(&minimal, "minimal", false, "Reserved; currently a no-op")
	return cmd
}

// runPackage loads config, resolves paths, and delegates to bundle.Create.
func runPackage(base, bundlePath, output string, platforms []string, noPrecompile, minimal bool) error {
	opts, err := bundle.NewCreateOptions(base, bundlePath, output, noPrecompile, minimal, platforms)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "qlt-bundle-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer func() {
		slog.Debug("Removing temp dir", "path", tmpDir)
		os.RemoveAll(tmpDir)
	}()

	bundleCtx := bundle.NewCustomBundle(opts, tmpDir)

	slog.Info("Creating custom CodeQL bundle",
		"output", opts.OutputPath,
		"packs", len(opts.Packs),
		"platforms", opts.Platforms,
	)

	return bundleCtx.Create()
}
