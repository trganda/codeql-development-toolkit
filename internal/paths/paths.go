// Package paths centralizes filesystem layout constants and helpers for qlt.
package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DefaultCLIDir is the per-user directory (relative to $HOME) where qlt
// installs CodeQL CLI binaries. The layout is:
//
//	$HOME/DefaultCLIDir/<tag>/codeql/                  ← extracted CLI
//	$HOME/DefaultCLIDir/<tag>/codeql-<platform>.zip    ← cached archive
//	$HOME/DefaultCLIDir/<tag>/codeql-<platform>.zip.checksum.txt
const DefaultCLIDir = ".qlt/codeql"

// VersionTag normalizes a version string to a "v"-prefixed git tag
// (e.g. "2.25.1" → "v2.25.1", "v2.25.1" → "v2.25.1").
func VersionTag(version string) string {
	return "v" + strings.TrimPrefix(version, "v")
}

// CLIInstallDir returns $HOME/.qlt/codeql/<tag> for the given version.
func CLIInstallDir(version string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DefaultCLIDir, VersionTag(version)), nil
}

// CodeQLBinary returns the absolute path to the codeql executable for the
// given installed version.
func CodeQLBinary(version string) (string, error) {
	dir, err := CLIInstallDir(version)
	if err != nil {
		return "", err
	}
	bin := "codeql"
	if runtime.GOOS == "windows" {
		bin = "codeql.exe"
	}
	return filepath.Join(dir, "codeql", bin), nil
}
