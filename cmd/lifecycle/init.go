package lifecycle

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/codeql"
	"github.com/trganda/codeql-development-toolkit/internal/config"
	"github.com/trganda/codeql-development-toolkit/internal/query"
	"github.com/trganda/codeql-development-toolkit/internal/release"
)

func newInitCmd(base *string) *cobra.Command {
	var (
		scope             string
		useBundle         bool
		bundleVersion     string
		overwriteExisting bool
		installCodeQL     bool
		codeqlVersion     string
		platform          string
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the CodeQL development workspace",
		Long: `Initialize lifecycle phase: set up the CodeQL development environment.

Writes codeql-workspace.yml under <base> and optionally updates qlt.conf.json
when --scope or --use-bundle is provided.

Pass --install-codeql to also download and install the CodeQL CLI or bundle
in the same step (equivalent to running 'qlt codeql install' afterwards).
When --use-bundle is set together with --install-codeql, the bundle is
installed instead of the standalone CLI.

Corresponds to: qlt query init  +  (optionally) qlt codeql install`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var cfg *config.QLTConfig
			var err error
			slog.Debug("Executing lifecycle init", "base", *base, "scope", scope,
				"use-bundle", useBundle, "install-codeql", installCodeQL)
			if cfg, err = query.InitWorkspace(*base, scope, codeqlVersion, bundleVersion, useBundle, overwriteExisting); err != nil {
				return err
			}

			if installCodeQL {
				slog.Debug("Installing CodeQL as part of lifecycle init",
					"version", codeqlVersion, "platform", platform, "bundleVersion", bundleVersion)
				return codeql.Install(*base, platform, cfg)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&scope, "scope", "", "CodeQL pack scope (GitHub username or org, e.g. trganda)")
	cmd.Flags().BoolVar(&useBundle, "use-bundle", false, "Enable custom CodeQL bundle support")
	cmd.Flags().BoolVar(&overwriteExisting, "overwrite-existing", false, "Overwrite existing files")
	cmd.Flags().BoolVar(&installCodeQL, "install-codeql", false, "Download and install the CodeQL CLI or bundle after workspace init")
	cmd.Flags().StringVar(&codeqlVersion, "codeql-version", release.LatestCLIVersion(), "CodeQL CLI version to use (e.g. 2.25.1);")
	cmd.Flags().StringVar(&platform, "platform", "", "Platform override (linux64, osx64, win64, all); auto-detected when empty.")
	cmd.Flags().StringVar(&bundleVersion, "bundle-version", release.LatestBundleVersion(), "CodeQL bundle version (e.g. codeql-bundle-v2.25.1); auto-resolved if omitted")
	return cmd
}
