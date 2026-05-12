package test

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/config"
	"github.com/trganda/codeql-development-toolkit/internal/matrix"
)

// newGetMatrixCmd returns `test run get-matrix`.
func newGetMatrixCmd(base *string) *cobra.Command {
	var osVersion string
	cmd := &cobra.Command{
		Use:   "get-matrix",
		Short: "Get a CI/CD matrix based on the current configuration",
		Run: func(cmd *cobra.Command, args []string) {
			slog.Debug("Executing test run get-matrix command", "base", *base, "os-version", osVersion)
			if err := runGetMatrix(*base, osVersion); err != nil {
				slog.Error("Get matrix failed", "err", err)
				os.Exit(1)
			}
		},
	}
	cmd.Flags().StringVar(&osVersion, "os-version", "ubuntu-latest", "Operating system(s) to use (comma-separated)")
	return cmd
}

func runGetMatrix(base, osVersions string) error {
	slog.Debug("Running get-matrix", "base", base, "os-versions", osVersions)
	cfg, err := config.LoadFromFile(base)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	cliVersion := "latest"
	if cfg != nil && cfg.CodeQLCLIVersion != "" {
		cliVersion = cfg.CodeQLCLIVersion
	}

	out, err := matrix.Build(osVersions, cliVersion)
	if err != nil {
		return err
	}

	slog.Debug("Generated matrix", "cliVersion", cliVersion)
	fmt.Printf("matrix=%s\n", string(out))
	return nil
}
