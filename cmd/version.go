package cmd

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
)

// Version is injected at build time via -ldflags:
//
//	-X github.com/trganda/codeql-development-toolkit/cmd.Version=<ver>
var Version = "dev"

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Get the current tool version",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing version command", "version", Version)
			fmt.Printf("QLT Version: %s\n", Version)
			return nil
		},
	}
}
