package codeql

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/codeql"
)

func newInstallCmd(base *string) *cobra.Command {
	var version, platform string
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Download and install the CodeQL CLI binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing codeql install command",
				"base", *base, "version", version, "platform", platform)
			return codeql.Install(*base, platform)
		},
	}

	cmd.Flags().StringVar(&platform, "platform", "", "Platform override (e.g. linux64, osx64, win64, all); auto-detected when empty. Use 'all' to download the multi-arch bundle.")
	return cmd
}
