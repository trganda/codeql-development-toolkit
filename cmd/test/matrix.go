package test

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"

	"github.com/trganda/codeql-development-toolkit/internal/config"
)

// newGetMatrixCmd returns `test run get-matrix`.
func newGetMatrixCmd(base *string) *cobra.Command {
	var osVersion string
	cmd := &cobra.Command{
		Use:   "get-matrix",
		Short: "Get a CI/CD matrix based on the current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			slog.Debug("Executing test run get-matrix command", "base", *base, "os-version", osVersion)
			return runGetMatrix(*base, osVersion)
		},
	}
	cmd.Flags().StringVar(&osVersion, "os-version", "ubuntu-latest", "Operating system(s) to use (comma-separated)")
	return cmd
}

// matrixEntry is a single matrix entry for GitHub Actions.
type matrixEntry struct {
	OS        string `json:"os"`
	CodeQLCLI string `json:"codeql_cli"`
}

func runGetMatrix(base, osVersions string) error {
	slog.Debug("Running get-matrix", "base", base, "os-versions", osVersions)
	cfg, err := config.LoadFromFile(base)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	cliVersion := "latest"
	if cfg != nil && cfg.CodeQLCLI != "" {
		cliVersion = cfg.CodeQLCLI
	}

	var entries []matrixEntry
	for _, os := range strings.Split(osVersions, ",") {
		os = strings.TrimSpace(os)
		if os == "" {
			continue
		}
		entries = append(entries, matrixEntry{OS: os, CodeQLCLI: cliVersion})
	}

	matrix := map[string]any{"include": entries}
	out, err := json.Marshal(matrix)
	if err != nil {
		return fmt.Errorf("marshal matrix: %w", err)
	}

	slog.Debug("Generated matrix", "entries", len(entries))
	fmt.Printf("matrix=%s\n", string(out))
	return nil
}
