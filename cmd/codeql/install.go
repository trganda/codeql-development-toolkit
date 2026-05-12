package codeql

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/codeql"
)

func newInstallCmd(base *string) *cobra.Command {
	var version, platform string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Download and install the CodeQL CLI binary",
		Run: func(cmd *cobra.Command, args []string) {
			slog.Debug("Executing codeql install command",
				"base", *base, "version", version, "platform", platform)
			if err := codeql.Install(*base, platform); err != nil {
				slog.Error("CodeQL install failed", "platform", platform, "err", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVar(&platform, "platform", "", "Platform override (e.g. linux64, osx64, win64, all); auto-detected when empty. Use 'all' to download the multi-arch bundle.")
	return cmd
}
