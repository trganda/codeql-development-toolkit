package codeql

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/config"
	"github.com/trganda/codeql-development-toolkit/internal/release"
)

// NewCommand returns the `codeql` cobra command.
func NewCommand(base, automationType *string, useBundle *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "codeql",
		Short: "CodeQL version management commands",
	}
	cmd.AddCommand(newSetCmd(base))
	cmd.AddCommand(newGetCmd(base))
	cmd.AddCommand(newInstallCmd(base, useBundle))
	return cmd
}

func newSetCmd(base *string) *cobra.Command {
	set := &cobra.Command{
		Use:   "set",
		Short: "Set CodeQL configuration values",
	}
	set.AddCommand(newSetVersionCmd(base))
	return set
}

func newSetVersionCmd(base *string) *cobra.Command {
	// Resolve defaults lazily so the CLI starts up fast; the network call
	// only happens when the user does not supply the flag explicitly.
	var cliVersion, bundleVersion string

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Set the CodeQL CLI and bundle version",
		Long: `Set the CodeQL CLI and bundle version in qlt.conf.json.

If --cli-version or --bundle-version are omitted the latest release is
fetched from GitHub's API automatically. If that request fails the
following fallback values are used:
  CLI:    ` + release.FallbackCLIVersion + `
  Bundle: ` + release.FallbackBundleVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing codeql set version command", "base", *base)

			// Apply auto-resolved defaults for flags the user did not set.
			if !cmd.Flags().Changed("cli-version") {
				slog.Debug("cli-version not set, resolving from GitHub")
				cliVersion = release.LatestCLIVersion()
				slog.Info("Auto-resolved CLI version", "version", cliVersion)
			}
			if !cmd.Flags().Changed("bundle-version") {
				slog.Debug("bundle-version not set, resolving from GitHub")
				bundleVersion = release.LatestBundleVersion()
				slog.Info("Auto-resolved bundle version", "version", bundleVersion)
			}

			cfg := &config.QLTConfig{
				CodeQLCLI:       cliVersion,
				CodeQLCLIBundle: bundleVersion,
			}
			if err := cfg.SaveToFile(*base); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			slog.Info("Saved CodeQL version config", "cli", cliVersion, "bundle", bundleVersion, "path", config.ConfigFilePath(*base))
			return nil
		},
	}
	cmd.Flags().StringVar(&cliVersion, "cli-version", "", "CodeQL CLI version (e.g. 2.25.1); auto-resolved if omitted")
	cmd.Flags().StringVar(&bundleVersion, "bundle-version", "", "CodeQL bundle version (e.g. codeql-bundle-v2.25.1); auto-resolved if omitted")
	return cmd
}

func newGetCmd(base *string) *cobra.Command {
	get := &cobra.Command{
		Use:   "get",
		Short: "Get CodeQL configuration values",
	}
	get.AddCommand(newGetVersionCmd(base))
	return get
}

func newGetVersionCmd(base *string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Get the current CodeQL CLI and bundle version",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing codeql get version command", "base", *base)
			cfg, err := config.MustLoadFromFile(*base)
			if err != nil {
				return err
			}
			slog.Debug("Loaded config", "cli", cfg.CodeQLCLI, "bundle", cfg.CodeQLCLIBundle)
			fmt.Println("---------current settings---------")
			fmt.Printf("CodeQL CLI Version: %s\n", cfg.CodeQLCLI)
			fmt.Printf("CodeQL CLI Bundle Version: %s\n", cfg.CodeQLCLIBundle)
			fmt.Println("----------------------------------")
			fmt.Println("(hint: use `qlt codeql set` to modify these values.)")
			return nil
		},
	}
}
