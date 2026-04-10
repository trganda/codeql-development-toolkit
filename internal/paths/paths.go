// Package paths centralizes filesystem layout constants and helpers for qlt.
package paths

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/trganda/codeql-development-toolkit/internal/config"
)

// DefaultCLIDir is the per-user directory (relative to $HOME) where qlt
// installs CodeQL CLI binaries. The layout is:
//
//	$HOME/DefaultCLIDir/<tag>/codeql/                  ← extracted CLI
//	$HOME/DefaultCLIDir/<tag>/codeql-<platform>.zip    ← cached archive
//	$HOME/DefaultCLIDir/<tag>/codeql-<platform>.zip.checksum.txt
const DefaultCLIDir = ".qlt/codeql"

// DefaultBundleDir is the per-user directory (relative to $HOME) where qlt
// stores downloaded CodeQL bundle archives. The layout is:
//
//	$HOME/DefaultBundleDir/<bundle-name>.tar.gz
const DefaultBundleDir = ".qlt/bundles"

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

// codeQLBinary returns the absolute path to the codeql executable for the
// given installed version.
func codeQLBinary(version string) (string, error) {
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

// BundleArchivePath returns the expected path for a named bundle archive under
// $HOME/.qlt/bundles/. The archive name is <bundleName>.tar.gz.
func BundleArchivePath(bundleName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DefaultBundleDir, bundleName+".tar.gz"), nil
}

// resolveCodeQLBinary returns the path to the codeql binary. It first looks
// at the version recorded in <base>/qlt.conf.json (installed by
// 'qlt codeql install'), then falls back to PATH.
func ResolveCodeQLBinary(base string) (string, error) {
	if cfg, _ := config.LoadFromFile(base); cfg != nil && cfg.CodeQLCLI != "" {
		if bin, err := codeQLBinary(cfg.CodeQLCLI); err == nil {
			if _, err := os.Stat(bin); err == nil {
				return bin, nil
			}
		}
	}

	path, err := exec.LookPath("codeql")
	if err != nil {
		return "", fmt.Errorf("codeql binary not found; run 'qlt codeql install' first")
	}
	return path, nil
}
