package bundle

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/trganda/codeql-development-toolkit/internal/bundle"
	"github.com/trganda/codeql-development-toolkit/internal/config"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

func newCreateCmd(base *string) *cobra.Command {
	var (
		bundlePath   string
		output       string
		packs        []string
		platforms    []string
		noPrecompile bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a custom CodeQL bundle",
		Long: `Create a custom CodeQL bundle by extending the base bundle specified in
qlt.conf.json with additional CodeQL packs from the workspace.

The base bundle archive is expected at $HOME/.qlt/bundles/<CodeQLCLIBundle>.tar.gz
unless overridden with --bundle.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing bundle create command", "base", *base)
			return runBundleCreate(*base, bundlePath, output, packs, platforms, noPrecompile)
		},
	}

	cmd.Flags().StringVar(&bundlePath, "bundle", "", "Path to the base CodeQL bundle archive (.tar.gz); defaults to $HOME/.qlt/bundles/<CodeQLCLIBundle>.tar.gz from config")
	cmd.Flags().StringVar(&output, "output", "", "Output path: a .tar.gz file for platform-agnostic, or a directory when --platform is used")
	cmd.Flags().StringArrayVar(&packs, "pack", nil, "Pack name to include (repeatable); e.g. --pack foo/cpp-customizations")
	cmd.Flags().StringArrayVar(&platforms, "platform", nil, "Target platform: linux64, osx64, win64 (repeatable); omit for platform-agnostic")
	cmd.Flags().BoolVar(&noPrecompile, "no-precompile", false, "Skip pre-compilation when bundling packs")

	cmd.MarkFlagRequired("output")
	cmd.MarkFlagRequired("pack")

	return cmd
}

func runBundleCreate(base, bundlePath, output string, packs, platforms []string, noPrecompile bool) error {
	cfg, err := config.MustLoadFromFile(base)
	if err != nil {
		return err
	}

	// Resolve the base bundle archive.
	if bundlePath == "" {
		if cfg.CodeQLCLIBundle == "" {
			return fmt.Errorf("CodeQLCLIBundle is not set in qlt.conf.json; run 'qlt codeql set version' or provide --bundle")
		}
		bundlePath, err = paths.BundleArchivePath(cfg.CodeQLCLIBundle)
		if err != nil {
			return fmt.Errorf("resolving bundle path: %w", err)
		}
	}

	if _, err := os.Stat(bundlePath); err != nil {
		return fmt.Errorf("bundle archive not found at %s: %w", bundlePath, err)
	}

	// Validate platforms.
	for _, p := range platforms {
		switch p {
		case "linux64", "osx64", "win64":
		default:
			return fmt.Errorf("unknown platform %q; must be one of: linux64, osx64, win64", p)
		}
	}

	slog.Info("Creating custom CodeQL bundle",
		"base-bundle", bundlePath,
		"output", output,
		"packs", packs,
		"platforms", platforms,
	)

	return bundle.Create(bundle.CreateOptions{
		BundlePath:   bundlePath,
		WorkspaceDir: base,
		Packs:        packs,
		OutputPath:   output,
		Platforms:    platforms,
		NoPrecompile: noPrecompile,
	})
}
