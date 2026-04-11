package lifecycle

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/bundle"
	"github.com/trganda/codeql-development-toolkit/internal/config"
	"github.com/trganda/codeql-development-toolkit/internal/paths"
)

func newPackageCmd(base *string) *cobra.Command {
	var (
		bundlePath   string
		output       string
		platforms    []string
		noPrecompile bool
	)
	cmd := &cobra.Command{
		Use:   "package",
		Short: "Create a custom CodeQL bundle",
		Long: `Package lifecycle phase: create a custom CodeQL bundle.

Reads packs from qlt.conf.json where Bundle=true and builds a custom bundle
using the base bundle archive downloaded by 'qlt codeql install'.

Corresponds to: qlt bundle create`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing lifecycle package", "base", *base)
			return runLifecyclePackage(*base, bundlePath, output, platforms, noPrecompile)
		},
	}
	cmd.Flags().StringVar(&bundlePath, "bundle", "", "Override base bundle archive path (.tar.gz)")
	cmd.Flags().StringVar(&output, "output", "", "Override output path (.tar.gz)")
	cmd.Flags().StringArrayVar(&platforms, "platform", nil, "Target platform: linux64, osx64, win64 (repeatable)")
	cmd.Flags().BoolVar(&noPrecompile, "no-precompile", false, "Skip pre-compilation when bundling packs")
	return cmd
}

func runLifecyclePackage(base, bundlePath, output string, platforms []string, noPrecompile bool) error {
	cfg, err := config.MustLoadFromFile(base)
	if err != nil {
		return err
	}

	// Collect packs configured for bundling.
	var packs []string
	for _, p := range cfg.CodeQLPackConfiguration {
		if p.Bundle {
			packs = append(packs, p.Name)
		}
	}
	if len(packs) == 0 {
		return fmt.Errorf("no packs configured for bundling in qlt.conf.json; set Bundle=true on at least one CodeQLPackConfiguration entry")
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

	// Resolve default output path.
	if output == "" {
		if cfg.CodeQLCLIBundle == "" {
			return fmt.Errorf("CodeQLCLIBundle is not set in qlt.conf.json; provide --output or run 'qlt codeql set version'")
		}
		output, err = paths.CustomBundlePath(cfg.CodeQLCLIBundle)
		if err != nil {
			return fmt.Errorf("resolving custom bundle output path: %w", err)
		}
		slog.Info("Using default custom bundle output path", "path", output)
	}

	// Ensure output directory exists.
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
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
