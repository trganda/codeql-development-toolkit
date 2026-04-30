package cmd

import (
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

// Version is injected at build time via -ldflags:
//
//	-X github.com/trganda/codeql-development-toolkit/cmd.Version=<ver>
var Version = "dev"

func resolvedVersion() string {
	// Prefer version injected by -ldflags in release builds.
	if Version != "" && Version != "dev" {
		return Version
	}

	// For `go install module@version`, Go embeds module version metadata.
	if bi, ok := debug.ReadBuildInfo(); ok {
		if v := normalizeVersion(bi.Main.Version); v != "" {
			return v
		}
	}

	return Version
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || v == "(devel)" {
		return ""
	}
	return strings.TrimPrefix(v, "v")
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Get the current tool version",
		RunE: func(cmd *cobra.Command, args []string) error {
			version := resolvedVersion()
			slog.Debug("Executing version command", "version", version)
			fmt.Printf("QLT Version: %s\n", version)
			return nil
		},
	}
}
