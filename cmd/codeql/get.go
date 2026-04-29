package codeql

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/config"
)

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
			cfg := config.MustLoadFromFile(*base)

			slog.Info("CodeQL CLI Version", "version", cfg.CodeQLCLIVersion)
			return nil
		},
	}
}
