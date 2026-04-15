package codeql

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/codeql"
	"github.com/trganda/codeql-development-toolkit/internal/config"
)

func newSetCmd(base string) *cobra.Command {
	set := &cobra.Command{
		Use:   "set",
		Short: "Set CodeQL configuration values",
	}
	set.AddCommand(newSetVersionCmd(base))
	return set
}

func newSetVersionCmd(base string) *cobra.Command {
	cliVersion := codeql.LatestCLIVersion()

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Set the CodeQL CLI and bundle version",
		Long: `Set the CodeQL CLI and bundle version in qlt.conf.json.

If --cli-version or --bundle-version are omitted the latest release is
fetched from GitHub's API automatically. If that request fails the
following fallback values are used:
  CLI:    ` + codeql.FallbackCLIVersion + `
  Bundle: ` + codeql.FallbackBundleVersion,
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing codeql set version command", "base", base)

			cfg := &config.QLTConfig{
				CodeQLCLI: cliVersion,
			}
			if err := cfg.SaveToFile(base); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&cliVersion, "cli-version", cliVersion, "CodeQL CLI version (e.g. 2.25.1); auto-resolved if omitted")
	return cmd
}
