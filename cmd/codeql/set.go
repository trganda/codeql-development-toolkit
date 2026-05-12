package codeql

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/codeql"
	"github.com/trganda/codeql-development-toolkit/internal/config"
)

func newSetCmd(base *string) *cobra.Command {
	set := &cobra.Command{
		Use:   "set",
		Short: "Set CodeQL configuration values",
	}
	set.AddCommand(newSetVersionCmd(base))
	return set
}

func newSetVersionCmd(base *string) *cobra.Command {
	cliVersion := codeql.LatestCLIVersion()

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Set the CodeQL CLI version",
		Long: `Set the CodeQL CLI version in qlt.conf.json.

If --cli-version is omitted the latest release is
fetched from GitHub's API automatically. If that request fails the
following fallback values are used:
  CLI:    ` + codeql.FallbackCLIVersion,
		Run: func(cmd *cobra.Command, args []string) {
			slog.Debug("Executing codeql set version command", "base", *base)

			cfg := config.MustLoadFromFile(*base)
			cfg.CodeQLCLIVersion = cliVersion
			if err := cfg.SaveToFile(*base); err != nil {
				slog.Error("Save config failed", "base", *base, "err", err)
				os.Exit(1)
			}
		},
	}
	cmd.Flags().StringVar(&cliVersion, "cli-version", cliVersion, "CodeQL CLI version (e.g. 2.25.1); auto-resolved if omitted")
	return cmd
}
