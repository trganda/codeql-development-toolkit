// Package paths centralizes filesystem layout constants and helpers for qlt.
package paths

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/trganda/codeql-development-toolkit/internal/config"
)

// DefaultPackagesDir is the per-user directory (relative to $HOME) where qlt
// installs CodeQL CLI binaries. The layout is:
//
//	$HOME/DefaultPackagesDir/<md5(version)>/codeql/              ← extracted CLI
//	$HOME/DefaultPackagesDir/<md5(version)>/codeql-<platform>.zip
//	$HOME/DefaultPackagesDir/<md5(version)>/codeql-<platform>.zip.checksum.txt
const DefaultPackagesDir = ".qlt/packages"

// DefaultBundleDir is the per-user directory (relative to $HOME) where qlt
// stores downloaded CodeQL bundle archives. The layout is:
//
//	$HOME/DefaultBundleDir/<md5(bundleName)>/codeql/             ← extracted bundle
//	$HOME/DefaultBundleDir/<md5(bundleName)>/codeql-bundle.tar.gz
//	$HOME/DefaultBundleDir/<md5(bundleName)>/codeql-bundle.tar.gz.checksum.txt
const DefaultBundleDir = ".qlt/bundle"

// DefaultCustomBundleDir is the directory (relative to --base) where qlt
// stores custom CodeQL bundles created by `qlt lifecycle package`. The layout is:
//
//	<base>/target/custom-bundle/<md5(bundleName)>/codeql-bundle.tar.gz
const DefaultCustomBundleDir = "target/custom-bundle"

// versionHash returns the lowercase MD5 hex digest of s (32 chars).
func versionHash(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// VersionTag normalizes a version string to a "v"-prefixed git tag
// (e.g. "2.25.1" → "v2.25.1", "v2.25.1" → "v2.25.1").
func VersionTag(version string) string {
	return "v" + strings.TrimPrefix(version, "v")
}

// CLIInstallDir returns $HOME/.qlt/packages/<md5(version)> for the given version.
func CLIInstallDir(version string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DefaultPackagesDir, versionHash(version)), nil
}

// codeQLBinary returns the absolute path to the codeql executable for the
// given installed CLI version.
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

// BundleInstallDir returns $HOME/.qlt/bundle/<md5(bundleName)> for the given bundle.
func BundleInstallDir(bundleName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, DefaultBundleDir, versionHash(bundleName)), nil
}

// bundleCodeQLBinary returns the codeql binary path inside an installed bundle.
func bundleCodeQLBinary(bundleName string) (string, error) {
	dir, err := BundleInstallDir(bundleName)
	if err != nil {
		return "", err
	}
	bin := "codeql"
	if runtime.GOOS == "windows" {
		bin = "codeql.exe"
	}
	return filepath.Join(dir, "codeql", bin), nil
}

// BundleArchivePath returns the expected path for a named bundle archive:
// $HOME/.qlt/bundle/<md5(bundleName)>/codeql-bundle.tar.gz.
func BundleArchivePath(bundleName string) (string, error) {
	dir, err := BundleInstallDir(bundleName)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "codeql-bundle.tar.gz"), nil
}

// CustomBundlePath returns the output path for a custom bundle created by
// `qlt lifecycle package`: <base>/target/custom-bundle/<md5(bundleName)>/codeql-bundle.tar.gz.
func CustomBundlePath(base, bundleName string) (string, error) {
	return filepath.Join(base, DefaultCustomBundleDir, versionHash(bundleName), "codeql-bundle.tar.gz"), nil
}

// ResolveCodeQLBinary returns the path to the codeql binary. When
// EnableCustomCodeQLBundles is true in config it resolves the binary from the
// installed bundle; otherwise it resolves the standalone CLI installation.
// Falls back to codeql found on PATH.
func ResolveCodeQLBinary(base string) (string, error) {
	if cfg, _ := config.LoadFromFile(base); cfg != nil {
		if cfg.EnableCustomCodeQLBundles && cfg.CodeQLCLIBundle != "" {
			if bin, err := bundleCodeQLBinary(cfg.CodeQLCLIBundle); err == nil {
				if _, err := os.Stat(bin); err == nil {
					return bin, nil
				}
			}
		} else if cfg.CodeQLCLI != "" {
			if bin, err := codeQLBinary(cfg.CodeQLCLI); err == nil {
				if _, err := os.Stat(bin); err == nil {
					return bin, nil
				}
			}
		}
	}

	path, err := exec.LookPath("codeql")
	if err != nil {
		return "", fmt.Errorf("codeql binary not found; run 'qlt codeql install' first")
	}
	return path, nil
}
