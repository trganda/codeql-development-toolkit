package lifecycle

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/bundle"
	"github.com/trganda/codeql-development-toolkit/internal/config"
	"github.com/trganda/codeql-development-toolkit/internal/query"
	qlttest "github.com/trganda/codeql-development-toolkit/internal/test"
	"github.com/trganda/codeql-development-toolkit/internal/utils"
)

func newPackageCmd(base *string) *cobra.Command {
	var (
		lang         string
		bundlePath   string
		output       string
		platforms    []string
		noPrecompile bool
	)
	cmd := &cobra.Command{
		Use:   "package",
		Short: "Create a custom CodeQL bundle",
		Long: `Package lifecycle phase: create a custom CodeQL bundle.

Runs the full chain: install → compile → test → verify → package.
Requires workspace initialization (run 'qlt lifecycle init' first).

Reads packs from qlt.conf.json where Bundle=true and builds a custom bundle
using the base bundle archive downloaded by 'qlt codeql install'.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing lifecycle package", "base", *base, "language", lang)
			if err := utils.CheckWorkspace(*base); err != nil {
				return err
			}
			if err := query.RunPackInstall(*base, lang, ""); err != nil {
				return err
			}
			if err := query.RunCompile(*base, lang, "", 0); err != nil {
				return err
			}
			if err := qlttest.RunUnitTests(*base, lang, "", 4); err != nil {
				return err
			}
			fmt.Println("verify: not yet fully implemented.")
			fmt.Println("Run 'qlt validation run check-queries --language <lang>' for available checks.")
			return RunLifecyclePackage(*base, bundlePath, output, platforms, noPrecompile)
		},
	}
	cmd.Flags().StringVar(&lang, "language", "", "Filter by language for install/compile/test steps (e.g. go, java)")
	cmd.Flags().StringVar(&bundlePath, "bundle", "", "Override base bundle archive path (.tar.gz)")
	cmd.Flags().StringVar(&output, "output", "", "Override output path (.tar.gz)")
	cmd.Flags().StringArrayVar(&platforms, "platform", nil, "Target platform: linux64, osx64, win64 (repeatable)")
	cmd.Flags().BoolVar(&noPrecompile, "no-precompile", false, "Skip pre-compilation when bundling packs")
	return cmd
}

// RunLifecyclePackage runs the package phase: it loads config, resolves paths,
// and delegates to bundle.Create.
func RunLifecyclePackage(base, bundlePath, output string, platforms []string, noPrecompile bool) error {
	cfg, err := config.MustLoadFromFile(base)
	if err != nil {
		return err
	}
	packs, err := bundle.CollectConfiguredPacks(cfg)
	if err != nil {
		return err
	}
	bundlePath, err = bundle.ResolveBundleArchive(cfg, bundlePath)
	if err != nil {
		return err
	}
	output, err = bundle.ResolveOutputPath(base, cfg, output)
	if err != nil {
		return err
	}
	if err := bundle.ValidatePlatforms(platforms); err != nil {
		return err
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
